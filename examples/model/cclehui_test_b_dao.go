package model

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/cclehui/dao-on-gorm/examples/global"
)

type CclehuiTestBDao struct {
	ColumnID     int       `gorm:"column:column_id;primaryKey" structs:"column_id" json:"column_id"`
	Version      int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight       float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age          time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra        string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAtNew time.Time `gorm:"column:created_at_new;precision:3" structs:"created_at_new" json:"created_at_new"`
	UpdatedAtNew time.Time `gorm:"column:updated_at_new;precision:3" structs:"updated_at_new" json:"updated_at_new"`

	daoBase *daoongorm.DaoBase
}

// 模型注册到 daoongorm, 供 NewDaoBase 按类型查找 ModelDef.
func init() {
	daoongorm.RegisterModel(&CclehuiTestBDao{})
}

func NewCclehuiTestBDao(ctx context.Context, myDao *CclehuiTestBDao, options ...daoongorm.Option) (*CclehuiTestBDao, error) {
	// 这个模型的 created_at / updated_at 字段名和常规不同, 所有构造都要带上
	options = append(options,
		daoongorm.OptionSetFieldNameCreatedAt("CreatedAtNew"),
		daoongorm.OptionSetFieldNameUpdatedAt("UpdatedAtNew"))

	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestBDao) DBName() string {
	return global.GetDBClient().GetDBClientConfig().DSN.DBName
}

func (myDao *CclehuiTestBDao) TableName() string {
	return "cclehui_test_b"
}

func (myDao *CclehuiTestBDao) DBClient() daoongorm.DBClientInterface {
	return global.GetDBClient()
}

func (myDao *CclehuiTestBDao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestBDao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}
