package dao

import (
	"context"
	"fmt"
	"strings"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/global"
	"github.com/cclehui/dao-on-gorm/examples/model"
	"github.com/cclehui/dao-on-gorm/examples/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// 直接从库中查数据
type CclehuiTestASelector struct{}

func (selector *CclehuiTestASelector) GetDataCountByID(ctx context.Context, dataID int) (int64, error) {
	tempDao := &model.CclehuiTestADao{}

	db := global.GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	var result int64

	newDB = newDB.Where(" id = ? ", dataID).Count(&result)

	if newDB.Error != nil && newDB.Error != gorm.ErrRecordNotFound {
		return result, errors.WithStack(newDB.Error)
	}

	return result, nil
}

// GetDataListParams 列表查询参数, GetDataList 与 GetDataListCount 共用.
//
// Name / NameLike 是两种过滤示例: Name 精确匹配, NameLike 模糊匹配。
// 两者同时传时是 AND 关系, 要二选一时由调用方只填一个。
type GetDataListParams struct {
	LastID    int    // 游标: 只取 id 大于它的行, 0 表示从头开始
	Limit     int    // 每页条数, <=0 时用默认值
	Name      string // 精确匹配 name, 空串表示不过滤
	NameLike  string // 模糊匹配 name (包含), 空串表示不过滤
	OrderBy   string // 排序字段, 只允许白名单里的列名; 空串默认 id
	OrderDesc bool   // true 按 OrderDesc 降序, 否则升序
}

// GetDataList 列表查询：单行 DAO 不承担 list, 直接在只读连接上 Scan 到切片.
//
// 默认按 id 升序做游标分页(id > LastID), 翻页期间删行也不会漏数据。
// 注意: 游标是 id, 所以改 OrderBy 之后 LastID 不再是有效的翻页游标,
// 换排序字段时要自己决定翻页方式(或者只取前 N 条)。
func (selector *CclehuiTestASelector) GetDataList(ctx context.Context, params *GetDataListParams) (
	[]*model.CclehuiTestADao, error) {
	if params == nil {
		params = &GetDataListParams{}
	}

	orderClause, err := selector.orderClause(params)
	if err != nil {
		return nil, err
	}

	tempDao := &model.CclehuiTestADao{}

	db := global.GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	limit := selector.defaultLimit(params)

	newDB = selector.applyFilters(newDB, params)

	list := make([]*model.CclehuiTestADao, 0)

	newDB = newDB.Select("*").
		Where(" id > ? ", params.LastID).
		Order(orderClause).
		Limit(limit).
		Find(&list)

	if newDB.Error != nil {
		return nil, errors.WithStack(newDB.Error)
	}

	// Scan 出来的结构体没有 daoBase, 补一个只读实例, 调用方才能继续用统一视图
	for _, item := range list {
		newDaoBase, _ := daoongorm.NewDaoBaseNoLoad(item)
		item.SetDaoBase(newDaoBase)
	}

	return list, nil
}

// GetDataListCount 统计满足 GetDataListParams 的行数.
//
// 口径与 GetDataList 一致: 过滤条件(含游标 LastID)相同, 忽略 Limit 与 OrderBy(它们不影响行数)。
// 所以传同一份 params 时, 这个值就是"从该游标往后一共还有多少行"。
func (selector *CclehuiTestASelector) GetDataListCount(ctx context.Context, params *GetDataListParams) (int64, error) {
	if params == nil {
		params = &GetDataListParams{}
	}

	tempDao := &model.CclehuiTestADao{}

	db := global.GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	newDB = selector.applyFilters(newDB, params)

	var result int64

	newDB = newDB.Where(" id > ? ", params.LastID).Count(&result)

	if newDB.Error != nil {
		return 0, errors.WithStack(newDB.Error)
	}

	return result, nil
}

// defaultLimit 每页条数默认值, 只在 params 上生效, 不暴露成包级常量.
func (selector *CclehuiTestASelector) defaultLimit(params *GetDataListParams) int {
	if params.Limit > 0 {
		return params.Limit
	}

	return 100
}

// orderColumn 把 OrderBy 映射成安全的列名.
//
// ORDER BY 的列名是 SQL 标识符, 不能用 ? 绑定, 所以只能从固定集合里取,
// 绝不把调用方的字符串直接拼进 SQL. 想支持新列就在这里加一个 case.
func (selector *CclehuiTestASelector) orderColumn(params *GetDataListParams) (string, error) {
	name := strings.ToLower(strings.TrimSpace(params.OrderBy))

	switch name {
	case "":
		return "id", nil
	case "id", "name", "version", "weight", "age", "created_at", "updated_at":
		return name, nil
	default:
		return "", errors.Errorf("unsupported order by %q", params.OrderBy)
	}
}

// orderClause 拼出 `ORDER BY` 片段: 列名来自白名单, 方向只有 asc/desc 两种.
func (selector *CclehuiTestASelector) orderClause(params *GetDataListParams) (string, error) {
	column, err := selector.orderColumn(params)
	if err != nil {
		return "", err
	}

	direction := "asc"
	if params.OrderDesc {
		direction = "desc"
	}

	return fmt.Sprintf("`%s` %s", column, direction), nil
}

// applyFilters 把过滤条件挂到 query 上. list 与 count 共用, 保证两者口径一致.
func (selector *CclehuiTestASelector) applyFilters(query *gorm.DB,
	params *GetDataListParams) *gorm.DB {
	if params.Name != "" {
		query = query.Where(" name = ? ", params.Name)
	}

	if params.NameLike != "" {
		// 用户输入里的 % 和 _ 是 LIKE 的通配符, 不转义就会把 "a_b" 匹配成 "axb".
		query = query.Where(" name like ? escape '\\\\' ", utils.SQLUtil.LikeContains(params.NameLike))
	}

	return query
}
