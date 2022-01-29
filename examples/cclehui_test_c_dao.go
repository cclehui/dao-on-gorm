package dao

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
)

// 非自增主键
type CclehuiTestCDao struct {
	ColumnID  int       `gorm:"column:column_id;primaryKey" structs:"column_id" json:"column_id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra     string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

func NewCclehuiTestCDao(ctx context.Context, myDao *CclehuiTestCDao, readOnly bool, options ...daoongorm.Option) (*CclehuiTestCDao, error) {
	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, readOnly, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

// 支持事务
func NewCclehuiTestCDaoWithTX(ctx context.Context,
	myDao *CclehuiTestCDao, tx *DBClientDemo, options ...daoongorm.Option) (*CclehuiTestCDao, error) {

	daoBase, err := daoongorm.NewDaoBaseWithTX(ctx, myDao, tx, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestCDao) DBName() string {
	return GetDBClient().GetDBClientConfig().DSN.DBName
}

func (myDao *CclehuiTestCDao) TableName() string {
	return "cclehui_test_c"
}

func (myDao *CclehuiTestCDao) DBClient() daoongorm.GormDBClient {
	return GetDBClient()
}

func (myDao *CclehuiTestCDao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestCDao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}
