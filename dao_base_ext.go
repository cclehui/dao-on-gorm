package daoongorm

import (
	"fmt"
	"reflect"
	"time"

	"github.com/cclehui/dao-on-gorm/internal"
	"github.com/pkg/errors"
)

// GetDefaultMapView dao 转 map 的默认视图.
//
// 时间列统一格式化成 "2006-01-02 15:04:05", 方便打日志; key 取自 structs tag,
// 需要按列名取值时用 GetColumnMapView.
func (daoBase *DaoBase) GetDefaultMapView() map[string]interface{} {
	tempMap := internal.StructToMap(daoBase.modelImpl)

	for _, field := range daoBase.modelDef.Fields {
		if field.Kind != reflect.Struct {
			continue
		}

		valueInterface := daoBase.modelReflectValue.FieldByName(field.Name).Interface()

		if valueTime, ok := valueInterface.(time.Time); ok {
			tempMap[field.ColumnName] = valueTime.Format("2006-01-02 15:04:05")
		}
	}

	return tempMap
}

// GetColumnMapView 以列名为主键返回当前字段值.
//
// GetDefaultMapView 依赖 structs 的 tag, 列名和 tag 不一致时 key 就对不上;
// 这里统一用 ModelDef 解析出的列名, 调用方可以放心按列名取值.
func (daoBase *DaoBase) GetColumnMapView() map[string]interface{} {
	columnMap := make(map[string]interface{}, len(daoBase.modelDef.Fields))

	for fieldName, field := range daoBase.modelDef.Fields {
		columnMap[field.ColumnName] = daoBase.modelReflectValue.FieldByName(fieldName).Interface()
	}

	return columnMap
}

// NewDaoBaseNoLoad 不加载数据的获取 model 的 daobase.
//
// 给"已经用原生 SQL 查出数据"的场景用: 拿到 daoBase 后仍可调用 Save/Delete/CacheKey.
func NewDaoBaseNoLoad(model Model, options ...Option) (*DaoBase, error) {
	fullName, v := GetModelFullNameAndValue(model)

	modelDef, ok := modelDefCache[fullName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("模型未注册, %s", fullName))
	}

	daoBase := &DaoBase{}

	daoBase.applyOptionDefaults()
	for _, option := range options {
		option.Apply(daoBase)
	}

	daoBase.modelImpl = model
	daoBase.modelReflectValue = v
	daoBase.modelDef = modelDef
	daoBase.isReadOnly = true
	daoBase.tx = nil
	daoBase.isNewRow = false

	daoBase.oldDataMap = daoBase.columnSnapshot()

	return daoBase, nil
}
