package daoongorm

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cclehui/dao-on-gorm/internal"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type DaoBase struct {
	modelImpl         Model // 模型的实现
	modelReflectValue reflect.Value
	modelDef          *ModelDef

	tx           GormDBClient `gorm:"-" json:"-"`
	isReadOnly   bool         `gorm:"-" json:"-"`
	isLoaded     bool         `gorm:"-" json:"-"`
	isLoadFromDB bool         `gorm:"-" json:"-"` // 是否是从库中加载
	isNewRow     bool         `gorm:"-" json:"-"`

	oldDataMap map[string]interface{} // load后的数据

	// option
	useCache           bool           // 是否启用缓存
	newForceCache      bool           // 记录不存在是否也缓存
	cacheUtil          CacheInterface // 缓存具体实现
	cacheExpireTS      int            // 缓存过期时间戳
	fieldNameCreatedAt string
	fieldNameUpdatedAt string
}

func NewDaoBase(ctx context.Context, model Model, readOnly bool, options ...Option) (*DaoBase, error) {
	return newDaoBaseFull(ctx, model, readOnly, nil, options...)
}

// 支持事务
func NewDaoBaseWithTX(ctx context.Context, model Model, tx GormDBClient, options ...Option) (*DaoBase, error) {
	return newDaoBaseFull(ctx, model, false, tx, options...)
}

func newDaoBaseFull(ctx context.Context, model Model, readOnly bool, tx GormDBClient, options ...Option) (
	*DaoBase, error) {
	fullName, v := GetModelFullNameAndValue(model)

	modelDef, ok := modelDefCache[fullName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("模型未注册, %s", fullName))
	}

	daoBase := &DaoBase{}
	daoBase.useCache = true
	daoBase.cacheUtil = globalCacheUtil
	daoBase.cacheExpireTS = DaoCacheExpire
	daoBase.fieldNameCreatedAt = FieldNameCreatedAt
	daoBase.fieldNameUpdatedAt = FieldNameUpdatedAt

	for _, option := range options {
		option.Apply(daoBase)
	}

	daoBase.modelImpl = model
	daoBase.modelReflectValue = v
	daoBase.modelDef = modelDef
	daoBase.isReadOnly = readOnly
	daoBase.tx = tx
	daoBase.isNewRow = true

	err := daoBase.Load(ctx)

	// load后的数据 先存成map 可以用于新旧数据比较
	daoBase.oldDataMap = internal.StructToMap(daoBase.modelImpl)

	return daoBase, err
}

func (daoBase *DaoBase) TableName() string {
	return daoBase.modelImpl.TableName()
}

func (daoBase *DaoBase) DBClient() GormDBClient {
	if daoBase.tx != nil {
		return daoBase.tx
	}

	return daoBase.modelImpl.DBClient()
}

func (daoBase *DaoBase) Load(ctx context.Context) error {
	if daoBase.isLoaded {
		return nil
	}

	if daoBase.primaryKeyIsEmpty() { // 主键为空
		daoBase.setFieldDefaultValue() // 设置默认值
		daoBase.isNewRow = true
		daoBase.isLoaded = true

		return nil
	}

	// 从缓存中获取 只读模式才从缓存中获取
	if daoBase.useCache && daoBase.isReadOnly &&
		daoBase.getFromCache(ctx) {
		daoBase.isNewRow = false
		daoBase.isLoaded = true

		return nil
	}

	// 主键条件
	pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()

	db := daoBase.DBClient()

	var newDB *gorm.DB

	if daoBase.isReadOnly {
		newDB = db.ReadOnlyTable(ctx, daoBase.TableName()).
			Where(pkFieldCondsStr, pkFieldValues...).Limit(1).Scan(daoBase.modelImpl)
	} else {
		if daoBase.tx != nil {
			newDB = db.Table(ctx, daoBase.TableName()).Set("gorm:query_option", "FOR UPDATE").
				Where(pkFieldCondsStr, pkFieldValues...).Limit(1).Scan(daoBase.modelImpl)
		} else {
			newDB = db.Table(ctx, daoBase.TableName()).
				Where(pkFieldCondsStr, pkFieldValues...).Limit(1).Scan(daoBase.modelImpl)
		}
	}

	if newDB.Error != nil {
		if errors.Is(newDB.Error, gorm.ErrRecordNotFound) {
			daoBase.isNewRow = true
		} else {
			return newDB.Error
		}
	} else if newDB.RowsAffected < 1 {
		daoBase.isNewRow = true
	} else {
		daoBase.isNewRow = false
	}

	daoBase.isLoaded = true
	daoBase.isLoadFromDB = true

	// 写入缓存
	if daoBase.useCache &&
		(!daoBase.isNewRow || daoBase.newForceCache) {
		daoBase.setCache(ctx)
	}

	return nil
}

func (daoBase *DaoBase) Create(ctx context.Context) error {
	if daoBase.isReadOnly {
		return errors.New("readOnly")
	}

	daoBase.createAutoFillValue() // 自动填充某些字段

	db := daoBase.DBClient()

	err := db.Table(ctx, daoBase.TableName()).Create(daoBase.modelImpl).Error
	if err != nil {
		return err
	}

	daoBase.isNewRow = false
	daoBase.isLoaded = true

	// 写入缓存 事务情况下不写缓存
	if daoBase.useCache && daoBase.tx == nil {
		daoBase.setCache(ctx)
	}

	return nil
}

func (daoBase *DaoBase) Update(ctx context.Context) error {
	if daoBase.isReadOnly {
		return errors.New("readOnly")
	}

	if !daoBase.isLoaded {
		return errors.New("需要先Load")
	}

	if daoBase.primaryKeyIsEmpty() {
		return errors.New("主键ID为空")
	}

	daoBase.updateAutoFillValue() // 自动填充某些字段

	updateValueMap := make(map[string]interface{}) // 要入库的值

	for fieldName, field := range daoBase.modelDef.Fields {
		if !field.IsPrimaryKey() { // 主键处理
			updateValueMap[field.ColumnName] = daoBase.modelReflectValue.FieldByName(fieldName).Interface()
		}
	}

	// 主键条件
	pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()

	db := daoBase.DBClient()

	newDB := db.Table(ctx, daoBase.TableName()).
		Where(pkFieldCondsStr, pkFieldValues...).Updates(updateValueMap)

	if newDB.Error != nil {
		return newDB.Error
	}

	// 写入缓存
	if daoBase.useCache {
		if daoBase.tx == nil {
			daoBase.setCache(ctx)
		} else { // 事务情况下需要删除缓存
			daoBase.deleteCache(ctx)
		}
	}

	return nil
}

// 自动识别创建还是更新
func (daoBase *DaoBase) Save(ctx context.Context) error {
	if daoBase.IsNewRow() {
		return daoBase.Create(ctx)
	}

	return daoBase.Update(ctx)
}

func (daoBase *DaoBase) Delete(ctx context.Context) error {
	if daoBase.isReadOnly {
		return errors.New("readOnly")
	}

	if daoBase.primaryKeyIsEmpty() {
		return errors.New("主键ID为空")
	}

	var db GormDBClient
	if daoBase.tx != nil {
		db = daoBase.tx
	} else {
		db = daoBase.DBClient()
	}

	// 主键条件
	// pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()
	// sqlStr := fmt.Sprintf("DELETE FROM `%s` WHERE %s", daoBase.TableName(), pkFieldCondsStr)
	// newDB := db.Exec(ctx, sqlStr, pkFieldValues...)
	// TODO

	newDB := db.Table(ctx, daoBase.TableName()).Delete(daoBase.modelImpl)

	if newDB.Error != nil {
		return newDB.Error
	}

	// 删除缓存
	if daoBase.useCache {
		daoBase.deleteCache(ctx)
	}

	return nil
}

func (daoBase *DaoBase) IsNewRow() bool {
	return daoBase.isNewRow
}

func (daoBase *DaoBase) IsLoadFromDB() bool {
	return daoBase.isLoadFromDB
}

func (daoBase *DaoBase) GetOldData() map[string]interface{} {
	return daoBase.oldDataMap
}

// 获取主键Where条件
func (daoBase *DaoBase) getPKFieldsWhereAndValues() (pkFieldCondsStr string, pkFieldValues []interface{}) {
	pkFieldConds := make([]string, 0)

	for fieldName, field := range daoBase.modelDef.PKFields {
		pkFieldConds = append(pkFieldConds, fmt.Sprintf("`%s` = ?", field.ColumnName))
		pkFieldValues = append(pkFieldValues, daoBase.modelReflectValue.FieldByName(fieldName).Interface())
	}

	pkFieldCondsStr = strings.Join(pkFieldConds, " and ")

	return pkFieldCondsStr, pkFieldValues
}

// 判定主键是否是空
func (daoBase *DaoBase) primaryKeyIsEmpty() bool {
	for fieldName, field := range daoBase.modelDef.PKFields {
		vf := daoBase.modelReflectValue.FieldByName(fieldName)

		switch field.Kind {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8:
			if vf.Int() == 0 {
				return true
			}
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint8:
			if vf.Uint() == 0 {
				return true
			}
		case reflect.String:
			if vf.String() == "" {
				return true
			}
		case reflect.Float32, reflect.Float64:
			if vf.Float() == 0.0 {
				return true
			}
		case reflect.Bool:
			if !vf.Bool() {
				return true
			}
		case reflect.Array, reflect.Chan, reflect.Complex128, reflect.Complex64,
			reflect.Func, reflect.Interface, reflect.Invalid, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Uintptr, reflect.UnsafePointer:
			if vf.String() == "" {
				return true
			}
		}
	}

	return false
}

// 主键转字符串
func (daoBase *DaoBase) PKFieldsStr() string {
	pkFields := daoBase.modelDef.PKFields // 主键处理 start
	pkKeys := make([]string, 0)
	pkValueMap := make(map[string]string)

	for _, field := range pkFields {
		if !field.Tag.IsPK {
			continue
		}

		fieldName := field.Name
		fieldValueStr := ReflectValueToStr(field.Kind, daoBase.modelReflectValue.FieldByName(fieldName))

		pkKeys = append(pkKeys, fieldName)
		pkValueMap[fieldName] = fieldValueStr
	}

	sort.Strings(pkKeys)

	valueSlice := make([]string, 0)

	for _, key := range pkKeys {
		valueSlice = append(valueSlice, pkValueMap[key])
	}

	suffix := strings.Join(valueSlice, "_")

	return suffix
}

func (daoBase *DaoBase) cacheKey() string {
	// 主键转字符串
	suffix := daoBase.PKFieldsStr()

	dbName := daoBase.modelImpl.DBName()
	tableName := daoBase.TableName()

	return fmt.Sprintf("%s:%s:%s:%s", DaoCachePrefix,
		dbName, tableName, suffix)
}

func (daoBase *DaoBase) GetUniqKey() string {
	dbName := daoBase.modelImpl.DBName()
	tableName := daoBase.TableName()

	suffix := daoBase.PKFieldsStr()

	return fmt.Sprintf("%s:%s:%s", dbName, tableName, suffix)
}

// 从缓存中读取
func (daoBase *DaoBase) getFromCache(ctx context.Context) bool {
	cacheUtil := daoBase.cacheUtil
	cacheKey := daoBase.cacheKey()

	if hit, err := cacheUtil.Get(ctx, cacheKey, daoBase.modelImpl); hit && err == nil {
		return true
	}

	return false
}

// 写入缓存
func (daoBase *DaoBase) setCache(ctx context.Context) {
	cacheUtil := daoBase.cacheUtil
	cacheKey := daoBase.cacheKey()
	expireTS := daoBase.cacheExpireTS

	err := cacheUtil.Set(ctx, cacheKey, daoBase.modelImpl, expireTS)
	if err != nil {
		logger.Errorc(ctx, "dao setCache key:%s error:%+v", cacheKey, err)
		daoBase.deleteCache(ctx) // 更新失败删除
	}
}

// 删除缓存
func (daoBase *DaoBase) deleteCache(ctx context.Context) {
	cacheUtil := daoBase.cacheUtil
	cacheKey := daoBase.cacheKey()

	err := cacheUtil.Del(ctx, cacheKey)
	if err != nil {
		logger.Errorc(ctx, "dao deleteCache key:%s, error:%+v", cacheKey, err)
	}
}

// 设置字段的默认值
func (daoBase *DaoBase) setFieldDefaultValue() {
	for n, f := range daoBase.modelDef.Fields {
		sf := daoBase.modelReflectValue.FieldByName(n)
		fieldTypeName := f.StructField.Type.Name()

		if !sf.CanSet() {
			err := errors.New(fmt.Sprintf("字段不能设置 fullName:%s, fieldName:%s",
				daoBase.modelDef.FullName, n))
			panic(err)
		}

		// 非空类型
		switch f.Kind {
		case reflect.Bool:
			var boolVal bool
			if !f.Tag.HaveColumnDefault {
				boolVal = false
			} else {
				if f.Tag.ColumnDefault == "0" {
					boolVal = false
				} else {
					boolVal = true
				}
			}

			sf.SetBool(boolVal)
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
			var int64Val int64
			if !f.Tag.HaveColumnDefault {
				int64Val = 0
			} else {
				i, err := strconv.ParseInt(f.Tag.ColumnDefault, 10, 64)
				if err != nil {
					panicCannotHandleFieldType(daoBase.modelDef.FullName, n, fieldTypeName, err)
				}
				int64Val = i
			}

			sf.SetInt(int64Val)
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
			var uint64Val uint64
			if !f.Tag.HaveColumnDefault {
				uint64Val = 0
			} else {
				i, err := strconv.ParseUint(f.Tag.ColumnDefault, 10, 64)
				if err != nil {
					panicCannotHandleFieldType(daoBase.modelDef.FullName, n, fieldTypeName, err)
				}
				uint64Val = i
			}

			sf.SetUint(uint64Val)
		case reflect.Float32, reflect.Float64:
			var float64Val float64
			if !f.Tag.HaveColumnDefault {
				float64Val = 0
			} else {
				f, err := strconv.ParseFloat(f.Tag.ColumnDefault, 64)
				if err != nil {
					panicCannotHandleFieldType(daoBase.modelDef.FullName, n, fieldTypeName, err)
				}

				float64Val = f
			}

			sf.SetFloat(float64Val)
		case reflect.String:
			if !f.Tag.HaveColumnDefault {
				sf.SetString("")
			} else {
				sf.SetString(f.Tag.ColumnDefault)
			}

			//	case reflect.Struct:
			//		if fieldTypeName == "Time" {
			//			now := time.Now()
			//			if !f.Tag.HaveColumnDefault {
			//				sf.Set(reflect.ValueOf(now))
			//			} else {
			//				if strings.EqualFold(f.Tag.ColumnDefault, "CURRENT_TIMESTAMP") {
			//					sf.Set(reflect.ValueOf(now))
			//				} else {
			//					sf.Set(reflect.ValueOf(php.Strtotime(f.Tag.ColumnDefault)))
			//				}
			//			}
			//		}
			// TODO
		case reflect.Array, reflect.Chan, reflect.Complex128, reflect.Complex64,
			reflect.Func, reflect.Interface, reflect.Invalid, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Uintptr, reflect.UnsafePointer:
			// 其他类型不支持作为model的struct的字段类型，在RegisterModel时已经做过check了，这里忽略
			break
		}
	}
}

// create 自动填充某些字段值
func (daoBase *DaoBase) createAutoFillValue() {
	for n, f := range daoBase.modelDef.Fields {
		sf := daoBase.modelReflectValue.FieldByName(n)
		fieldTypeName := f.StructField.Type.Name()

		if !sf.CanSet() {
			continue
		}

		switch f.Kind {
		case reflect.Struct:
			if fieldTypeName == TypeNameTime &&
				(f.Name == daoBase.fieldNameCreatedAt || f.Name == daoBase.fieldNameUpdatedAt) &&
				isEmptyTime(sf) {
				now := time.Now()
				sf.Set(reflect.ValueOf(now))
			}
		case reflect.Array, reflect.Bool, reflect.Chan, reflect.Complex128,
			reflect.Complex64, reflect.Float32, reflect.Float64, reflect.Func,
			reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8,
			reflect.Interface, reflect.Invalid, reflect.Map, reflect.Ptr, reflect.Slice,
			reflect.String, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uint8, reflect.Uintptr, reflect.UnsafePointer:
			break
		}
	}
}

// update 自动填充某些字段值
func (daoBase *DaoBase) updateAutoFillValue() {
	for n, f := range daoBase.modelDef.Fields {
		sf := daoBase.modelReflectValue.FieldByName(n)
		fieldTypeName := f.StructField.Type.Name()

		if !sf.CanSet() {
			continue
		}

		switch f.Kind {
		case reflect.Struct:
			if fieldTypeName == TypeNameTime &&
				(f.Name == daoBase.fieldNameUpdatedAt) &&
				sf.Interface() == daoBase.oldDataMap[f.ColumnName] {
				now := time.Now()
				sf.Set(reflect.ValueOf(now))
			}
		case reflect.Array, reflect.Bool, reflect.Chan, reflect.Complex128,
			reflect.Complex64, reflect.Float32, reflect.Float64, reflect.Func,
			reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int8,
			reflect.Interface, reflect.Invalid, reflect.Map, reflect.Ptr, reflect.Slice,
			reflect.String, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Uint8, reflect.Uintptr, reflect.UnsafePointer:
			break
		}
	}
}
