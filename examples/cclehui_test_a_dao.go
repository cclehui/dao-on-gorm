package dao

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
)

type CclehuiTestADao struct {
	ID        int       `gorm:"column:id;primaryKey" structs:"id" json:"id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra     string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

// 把模型注册到gaoongorm中去， 方便全局管理
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
	myDao *CclehuiTestADao, tx *DBClientDemo, options ...daoongorm.Option) (*CclehuiTestADao, error) {

	daoBase, err := daoongorm.NewDaoBaseWithTX(ctx, myDao, tx, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestADao) DBName() string {
	return "test"
}

func (myDao *CclehuiTestADao) TableName() string {
	return "cclehui_test_a"
}

func (myDao *CclehuiTestADao) DBClient() daoongorm.GormDBClient {
	return GetDBClient()
}

func (myDao *CclehuiTestADao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestADao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}
