package dao

import (
	"context"

	daoongorm "github.com/cclehui/dao-on-gorm"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// 直接从库中查数据
type CclehuiTestASelector struct{}

func (selector *CclehuiTestASelector) GetDataByID(ctx context.Context, dataID int) (
	*CclehuiTestADao, error) {
	tempDao := &CclehuiTestADao{}

	db := GetDBClient()
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
	tempDao := &CclehuiTestADao{}

	db := GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	var result int64

	newDB = newDB.Where(" id = ? ", dataID).Count(&result)

	if newDB.Error != nil && newDB.Error != gorm.ErrRecordNotFound {
		return result, errors.WithStack(newDB.Error)
	}

	return result, nil
}

// 列表查询：单行 DAO 不承担 list，直接在只读连接上 Scan 到切片。
// 用 id > lastID 的游标分页而不是 OFFSET：翻页时不会因为前面的行被删掉而漏数据。
func (selector *CclehuiTestASelector) GetDataList(ctx context.Context, lastID int, limit int) (
	[]*CclehuiTestADao, error) {
	tempDao := &CclehuiTestADao{}

	db := GetDBClient()
	newDB := db.ReadOnlyTable(ctx, tempDao.TableName())

	if limit <= 0 {
		limit = 100
	}

	list := make([]*CclehuiTestADao, 0)

	newDB = newDB.Select("*").
		Where(" id > ? ", lastID).
		Order(" id asc ").
		Limit(limit).
		Find(&list)

	if newDB.Error != nil {
		return nil, errors.WithStack(newDB.Error)
	}

	// Scan 出来的结构体没有 daoBase，补一个只读实例，调用方才能继续用统一视图
	for _, item := range list {
		newDaoBase, _ := daoongorm.NewDaoBaseNoLoad(item)
		item.SetDaoBase(newDaoBase)
	}

	return list, nil
}
