package dao

import (
	"context"
	"time"

	daoongorm "github.com/cclehui/dao-on-gorm"
)

// 联合主键
type CclehuiTestDDao struct {
	UserID    int       `gorm:"column:user_id;primaryKey" structs:"user_id" json:"user_id"`
	ColumnID  int       `gorm:"column:column_id;primaryKey" structs:"column_id" json:"column_id"`
	Version   int64     `gorm:"column:version" structs:"version" json:"version"`
	Weight    float64   `gorm:"column:weight;column_default:1.9" structs:"weight" json:"weight"`
	Age       time.Time `gorm:"column:age" structs:"age" json:"age"`
	Extra     string    `gorm:"column:extra" structs:"extra" json:"extra"`
	CreatedAt time.Time `gorm:"column:created_at" structs:"created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" structs:"updated_at" json:"updated_at"`

	daoBase *daoongorm.DaoBase
}

func NewCclehuiTestDDao(ctx context.Context, myDao *CclehuiTestDDao, readOnly bool, options ...daoongorm.Option) (*CclehuiTestDDao, error) {
	daoBase, err := daoongorm.NewDaoBase(ctx, myDao, readOnly, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

// 支持事务
func NewCclehuiTestDDaoWithTX(ctx context.Context,
	myDao *CclehuiTestDDao, tx *DBClientDemo, options ...daoongorm.Option) (*CclehuiTestDDao, error) {

	daoBase, err := daoongorm.NewDaoBaseWithTX(ctx, myDao, tx, options...)

	myDao.daoBase = daoBase

	return myDao, err
}

func (myDao *CclehuiTestDDao) DBName() string {
	return GetDBClient().GetDBClientConfig().DSN.DBName
}

func (myDao *CclehuiTestDDao) TableName() string {
	return "cclehui_test_d"
}

func (myDao *CclehuiTestDDao) DBClient() daoongorm.GormDBClient {
	return GetDBClient()
}

func (myDao *CclehuiTestDDao) GetDaoBase() *daoongorm.DaoBase {
	return myDao.daoBase
}

func (myDao *CclehuiTestDDao) SetDaoBase(myDaoBase *daoongorm.DaoBase) {
	myDao.daoBase = myDaoBase
}
