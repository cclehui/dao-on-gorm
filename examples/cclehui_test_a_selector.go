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
