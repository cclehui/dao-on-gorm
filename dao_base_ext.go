package database

import (
	"fmt"
	"reflect"
	"time"

	"git2.qingtingfm.com/car/car-common/base"
	"github.com/pkg/errors"
)

// dao默认转map结构
func (daoBase *DaoBase) GetDefaultMapView() map[string]interface{} {
	tempMap := base.StructToMap(daoBase.modelImpl)

	for _, field := range daoBase.modelDef.Fields {
		if field.Kind == reflect.Struct {
			// 获取具体某列的实例
			valueInterface := daoBase.modelReflectValue.FieldByName(field.Name).Interface()

			if valueTime, ok := valueInterface.(time.Time); ok {
				tempMap[field.ColumnName] = valueTime.Format("2006-01-02 15:04:05")
			}
		}
	}

	return tempMap
}

// 不加载数据的 获取model的daobase
// 目的为了获取对应的daobase 方法拿到底层的模型函数
func NewDaoBaseNoLoad(model Model) (*DaoBase, error) {
	fullName, v := GetModelFullNameAndValue(model)

	modelDef, ok := modelDefCache[fullName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("模型未注册, %s", fullName))
	}

	daoBase := &DaoBase{}

	daoBase.modelImpl = model
	daoBase.modelReflectValue = v
	daoBase.modelDef = modelDef
	daoBase.isReadOnly = true
	daoBase.tx = nil
	daoBase.isNewRow = false

	// load后的数据 先存成map 可以用于新旧数据比较
	daoBase.oldDataMap = base.StructToMap(daoBase.modelImpl)

	return daoBase, nil
}

// 批量Load dao 只读模式 5个并发
// err只返回最早出现的error
/*
func MultiLoadDaoList(ctx context.Context, modelImplsInterface interface{}) error {
	modelImplSlice := base.InterfaceToSlice(modelImplsInterface)

	//.errSync := sync.Once{}

	for _, modelImplInterface := range modelImplSlice {
		modelImpl, ok := modelImplInterface.(Model)
		if !ok {
			log.Errorc(ctx, "MultiLoadDaoList, error:%+v",
				errors.New("非database.Model类型"))
			continue
		}

		daoBase, err := NewDaoBase(ctx, modelImpl, true) // 只读模式

		if err != nil {
			log.Errorc(ctx, "MultiLoadDaoList, error:%+v", errors.WithStack(err))
			continue
		}

		modelReflectValue := reflect.ValueOf(modelImpl).Elem()

		daoBaseValue := modelReflectValue.FieldByName("daoBase")
		if _, ok := daoBaseValue.Interface().(*DaoBase); !ok {
			log.Errorc(ctx, "MultiLoadDaoList, error:%+v",
				errors.New(fmt.Sprintf("modelImpl, %#v, daoBase字段异常", modelImpl)))
			continue
		}

		daoBaseValue.Set(reflect.ValueOf(daoBase))
	}

	return nil
}
*/
// TODO
