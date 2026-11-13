package dao

import (
	"context"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/global"
	"github.com/cclehui/dao-on-gorm/examples/model"
	"github.com/cclehui/dao-on-gorm/examples/utils"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// 直接从库中查数据
type CclehuiTestASelector struct{}

func (selector *CclehuiTestASelector) GetDataByID(ctx context.Context, dataID int) (
	*model.CclehuiTestADao, error) {
	tempDao := &model.CclehuiTestADao{}

	db := global.GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	newDB = newDB.Select("*").
		Where(" id = ? ", dataID).
		Take(tempDao)

	if newDB.Error != nil && newDB.Error != gorm.ErrRecordNotFound {
		return nil, errors.WithStack(newDB.Error)
	}

	newDaoBase, _ := daoongorm.NewDaoBaseNoLoad(tempDao)
	tempDao.SetDaoBase(newDaoBase)

	return tempDao, nil
}

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

// GetDataListParams 列表查询参数.
//
// Name / NameLike 是两种过滤示例: Name 精确匹配, NameLike 模糊匹配。
// 两者同时传时是 AND 关系, 要二选一时由调用方只填一个。
type GetDataListParams struct {
	LastID   int    // 游标: 只取 id 大于它的行, 0 表示从头开始
	Limit    int    // 每页条数, <=0 时用默认值
	Name     string // 精确匹配 name, 空串表示不过滤
	NameLike string // 模糊匹配 name (包含), 空串表示不过滤
}

const defaultListLimit = 100

// GetDataList 列表查询：单行 DAO 不承担 list, 直接在只读连接上 Scan 到切片.
// 用 id > lastID 的游标分页而不是 OFFSET: 翻页时不会因为前面的行被删掉而漏数据.
func (selector *CclehuiTestASelector) GetDataList(ctx context.Context, params *GetDataListParams) (
	[]*model.CclehuiTestADao, error) {
	if params == nil {
		params = &GetDataListParams{}
	}

	tempDao := &model.CclehuiTestADao{}

	db := global.GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	limit := params.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}

	if params.Name != "" {
		newDB = newDB.Where(" name = ? ", params.Name)
	}

	if params.NameLike != "" {
		// 用户输入里的 % 和 _ 是 LIKE 的通配符, 不转义就会把 "a_b" 匹配成 "axb".
		newDB = newDB.Where(" name like ? escape '\\\\' ", utils.SQLUtil.LikeContains(params.NameLike))
	}

	list := make([]*model.CclehuiTestADao, 0)

	newDB = newDB.Select("*").
		Where(" id > ? ", params.LastID).
		Order(" id asc ").
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
