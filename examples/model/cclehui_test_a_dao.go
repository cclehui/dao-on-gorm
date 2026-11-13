package model

import (
	"context"
	"encoding/json"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/global"
	"github.com/pkg/errors"
)

type CclehuiTestADao struct {
	ID      int       `gorm:"column:id;primaryKey" structs:"id" json:"id"`
	Version int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight  float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age     time.Time `gorm:"column:age" structs:"age" json:"age"`
	Name    string    `gorm:"column:name" structs:"name" json:"name"`
	// Extra 存 JSON 文本: 本库只映射基础类型, 不支持 map/struct.
	// 必须给默认值 "{}", 否则新建行会写入空串, MySQL 对 JSON 列直接报 3140.
	Extra     string    `gorm:"column:extra;type:json;column_default:{}" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at;precision:3" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;precision:3" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

// 模型注册到 daoongorm, 供 NewDaoBase 按类型查找 ModelDef.
func init() {
	daoongorm.RegisterModel(&CclehuiTestADao{})
}

func NewCclehuiTestADao(ctx context.Context, myDao *CclehuiTestADao, readOnly bool, options ...daoongorm.Option) (*CclehuiTestADao, error) {
	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, readOnly, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

// 支持事务
func NewCclehuiTestADaoWithTX(ctx context.Context,
	myDao *CclehuiTestADao, tx *daoongorm.DBClient, options ...daoongorm.Option) (*CclehuiTestADao, error) {

	daoBase, err := daoongorm.NewDaoBaseWithTX(ctx, myDao, tx, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestADao) DBName() string {
	return global.GetDBClient().GetDBClientConfig().DSN.DBName
}

func (myDao *CclehuiTestADao) TableName() string {
	return "cclehui_test_a"
}

func (myDao *CclehuiTestADao) DBClient() daoongorm.DBClientInterface {
	return global.GetDBClient()
}

func (myDao *CclehuiTestADao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestADao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}

// CclehuiTestAExtra 是 extra 列的 JSON 结构.
// 本库只映射基础类型, 所以列在 DAO 上是 string, 由 GetExtra 手工解码.
// 新增字段时注意: 库里已有的数据不会自动补字段, 解码后得到零值.
type CclehuiTestAExtra struct {
	Tag   string `json:"tag"`
	Score int    `json:"score"`
}

// GetExtra 把 extra 列的 JSON 文本解码成结构体.
//
// 空串按空对象处理: 老数据/手写 SQL 插入的行可能是空串, 不应该让读取方直接报错.
func (myDao *CclehuiTestADao) GetExtra() (*CclehuiTestAExtra, error) {
	extra := &CclehuiTestAExtra{}

	if myDao.Extra == "" {
		return extra, nil
	}

	if err := json.Unmarshal([]byte(myDao.Extra), extra); err != nil {
		return nil, errors.Wrapf(err, "解析 extra 失败, table:%s, id:%d, extra:%s",
			myDao.TableName(), myDao.ID, myDao.Extra)
	}

	return extra, nil
}
