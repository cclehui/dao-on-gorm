package dao

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
)

type CclehuiTestBDao struct {
	ColumnID     int       `gorm:"column:column_id;primaryKey" structs:"column_id" json:"column_id"`
	Version      int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight       float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age          time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra        string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAtNew time.Time `gorm:"column:created_at_new" structs:"created_at_new" json:"created_at_new"`
	UpdatedAtNew time.Time `gorm:"column:updated_at_new" structs:"updated_at_new" json:"updated_at_new"`

	daoBase *daoongorm.DaoBase
}

func NewCclehuiTestBDao(ctx context.Context, myDao *CclehuiTestBDao, readOnly bool, options ...daoongorm.Option) (*CclehuiTestBDao, error) {
	options = append(options,
		daoongorm.OptionSetFieldNameCreatedAt("CreatedAtNew"),
		daoongorm.OptionSetFieldNameUpdatedAt("UpdatedAtNew"))
	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, readOnly, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

// 支持事务
func NewCclehuiTestBDaoWithTX(ctx context.Context,
	myDao *CclehuiTestBDao, tx *DBClientDemo, options ...daoongorm.Option) (*CclehuiTestBDao, error) {
	options = append(options,
		daoongorm.OptionSetFieldNameCreatedAt("CreatedAtNew"),
		daoongorm.OptionSetFieldNameUpdatedAt("UpdatedAtNew"))

	daoBase, err := daoongorm.NewDaoBaseWithTX(ctx, myDao, tx, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestBDao) DBName() string {
	return GetDBClient().GetDBClientConfig().DSN.DBName
}

func (myDao *CclehuiTestBDao) TableName() string {
	return "cclehui_test_b"
}

func (myDao *CclehuiTestBDao) DBClient() daoongorm.GormDBClient {
	return GetDBClient()
}

func (myDao *CclehuiTestBDao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestBDao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}
