package daoongorm

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// DaoBase 单行数据的读写入口.
//
// 一个 DaoBase 绑定一个 model 指针, 生命周期内只操作这一行:
// 主键为空时 Save 走 Create, 主键非空时先 Load, 再 Save 走 Update.
// 缓存和 created_at/updated_at 的自动填充都在这一层完成, 调用方不用自己拼 SQL.
type DaoBase struct {
	modelImpl         Model // 模型的实现
	modelReflectValue reflect.Value
	modelDef          *ModelDef

	tx           DBClientInterface `gorm:"-" json:"-"`
	isReadOnly   bool              `gorm:"-" json:"-"`
	isLoaded     bool              `gorm:"-" json:"-"`
	isLoadFromDB bool              `gorm:"-" json:"-"` // 是否是从库中加载
	isNewRow     bool              `gorm:"-" json:"-"`

	oldDataMap map[string]interface{} // load后的数据, key 为列名

	// option
	useCache           bool           // 是否启用缓存
	newForceCache      bool           // 记录不存在是否也缓存
	cacheUtil          CacheInterface // 缓存具体实现
	cacheExpireTS      int            // 缓存过期时间戳
	fieldNameCreatedAt string
	fieldNameUpdatedAt string
}

// savedHash 用于判定"这次 Save 有没有真的改动字段".
//
// 只记字段值的 hash, 不依赖 map 的 == 比较: 早先的实现直接比较 interface{},
// 遇到 time.Time 之外的非可比较类型会 panic, 且 tag 名和列名不一致时永远判断为"有变化".
func (daoBase *DaoBase) columnSnapshot() map[string]interface{} {
	return daoBase.GetColumnMapView()
}

func NewDaoBase(ctx context.Context, model Model, readOnly bool, options ...Option) (*DaoBase, error) {
	return newDaoBaseFull(ctx, model, readOnly, nil, options...)
}

// 支持事务
func NewDaoBaseWithTX(ctx context.Context, model Model, tx DBClientInterface, options ...Option) (*DaoBase, error) {
	return newDaoBaseFull(ctx, model, false, tx, options...)
}

func newDaoBaseFull(ctx context.Context, model Model, readOnly bool, tx DBClientInterface, options ...Option) (
	*DaoBase, error) {
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
	daoBase.isReadOnly = readOnly
	daoBase.tx = tx
	daoBase.isNewRow = true

	err := daoBase.Load(ctx)
	if err != nil {
		return nil, err
	}

	// load 后的数据存一份, 供 Save 时判断字段是否真的变化
	daoBase.oldDataMap = daoBase.columnSnapshot()

	return daoBase, nil
}

func (daoBase *DaoBase) applyOptionDefaults() {
	daoBase.useCache = true
	daoBase.cacheUtil = globalCacheUtil
	daoBase.cacheExpireTS = DaoCacheExpire
	daoBase.fieldNameCreatedAt = FieldNameCreatedAt
	daoBase.fieldNameUpdatedAt = FieldNameUpdatedAt
}

func (daoBase *DaoBase) TableName() string {
	return daoBase.modelImpl.TableName()
}

func (daoBase *DaoBase) DBClient() DBClientInterface {
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
	if daoBase.useCache && daoBase.isReadOnly && daoBase.getFromCache(ctx) {
		daoBase.isNewRow = false
		daoBase.isLoaded = true

		return nil
	}

	pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()

	db := daoBase.DBClient()

	var newDB *gorm.DB

	if daoBase.isReadOnly {
		newDB = db.ReadOnlyTable(ctx, daoBase.TableName()).
			Where(pkFieldCondsStr, pkFieldValues...).Limit(1).Scan(daoBase.modelImpl)
	} else {
		// 事务内用 FOR UPDATE 锁住这一行, 避免两个事务同时读到旧值再互相覆盖.
		// 用 gorm 的 clause.Locking 而不是 Set("gorm:query_option"): 后者在 gorm v2 已不生效.
		query := db.Table(ctx, daoBase.TableName())
		if daoBase.tx != nil {
			query = query.Clauses(lockForUpdate())
		}

		newDB = query.Where(pkFieldCondsStr, pkFieldValues...).Limit(1).Scan(daoBase.modelImpl)
	}

	if newDB.Error != nil {
		return newDB.Error
	}

	daoBase.isNewRow = newDB.RowsAffected < 1
	daoBase.isLoaded = true
	daoBase.isLoadFromDB = true

	// 写入缓存: 事务内不写, 否则未提交的数据会被其它请求读到
	if daoBase.useCache && daoBase.tx == nil && (!daoBase.isNewRow || daoBase.newForceCache) {
		daoBase.setCache(ctx)
	}

	return nil
}

func (daoBase *DaoBase) Create(ctx context.Context) error {
	if daoBase.isReadOnly {
		return errors.New("readOnly")
	}

	daoBase.createAutoFillValue() // 自动填充某些字段
	daoBase.roundTimeFields()     // 按列精度归一化, 让内存值和落库值一致

	db := daoBase.DBClient()

	if err := db.Table(ctx, daoBase.TableName()).Create(daoBase.modelImpl).Error; err != nil {
		return err
	}

	daoBase.isNewRow = false
	daoBase.isLoaded = true
	daoBase.oldDataMap = daoBase.columnSnapshot()

	// 写入缓存 事务情况下不写缓存
	daoBase.refreshCache(ctx)

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
	daoBase.roundTimeFields()     // 按列精度归一化, 让内存值和落库值一致

	// 只更新真正变化的列: 全列更新会把并发读到的其他字段的旧值一起写回去.
	updateValueMap := daoBase.changedColumnValues()

	if len(updateValueMap) == 0 {
		return nil
	}

	pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()

	db := daoBase.DBClient()

	newDB := db.Table(ctx, daoBase.TableName()).
		Where(pkFieldCondsStr, pkFieldValues...).Updates(updateValueMap)

	if newDB.Error != nil {
		return newDB.Error
	}

	daoBase.oldDataMap = daoBase.columnSnapshot()

	// 写入缓存. 事务内不能直接写: 事务可能回滚, 提前失效也会让并发读把旧值灌回缓存.
	daoBase.refreshCache(ctx)

	return nil
}

// Save 自动识别创建还是更新
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

	pkFieldCondsStr, pkFieldValues := daoBase.getPKFieldsWhereAndValues()

	newDB := daoBase.DBClient().Table(ctx, daoBase.TableName()).
		Where(pkFieldCondsStr, pkFieldValues...).Delete(daoBase.modelImpl)

	if newDB.Error != nil {
		return newDB.Error
	}

	daoBase.invalidateCache(ctx)

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

// getPKFieldsWhereAndValues 获取主键 Where 条件. 顺序固定为 ModelDef.PKFieldNames,
// 保证联合主键生成的 SQL 稳定.
func (daoBase *DaoBase) getPKFieldsWhereAndValues() (pkFieldCondsStr string, pkFieldValues []interface{}) {
	pkFieldConds := make([]string, 0, len(daoBase.modelDef.PKFieldNames))
	pkFieldValues = make([]interface{}, 0, len(daoBase.modelDef.PKFieldNames))

	for _, fieldName := range daoBase.modelDef.PKFieldNames {
		field := daoBase.modelDef.PKFields[fieldName]
		pkFieldConds = append(pkFieldConds, fmt.Sprintf("`%s` = ?", field.ColumnName))
		pkFieldValues = append(pkFieldValues, daoBase.modelReflectValue.FieldByName(fieldName).Interface())
	}

	pkFieldCondsStr = strings.Join(pkFieldConds, " and ")

	return pkFieldCondsStr, pkFieldValues
}

// primaryKeyIsEmpty 判定主键是否是空. 只处理注册阶段允许的类型, 不用 vf.String()
// 去猜 struct/map 的空值 (那样要么 panic 要么给出错误结论).
func (daoBase *DaoBase) primaryKeyIsEmpty() bool {
	for fieldName := range daoBase.modelDef.PKFields {
		vf := daoBase.modelReflectValue.FieldByName(fieldName)

		switch vf.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if vf.Int() == 0 {
				return true
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
		default:
			return true
		}
	}

	return false
}

// PKFieldsStr 主键转字符串, 用于缓存 key 和联合主键的唯一标识.
func (daoBase *DaoBase) PKFieldsStr() string {
	pkValues := make([]string, 0, len(daoBase.modelDef.PKFieldNames))

	for _, fieldName := range daoBase.modelDef.PKFieldNames {
		field := daoBase.modelDef.PKFields[fieldName]
		pkValues = append(pkValues, ReflectValueToStr(field.Kind,
			daoBase.modelReflectValue.FieldByName(fieldName)))
	}

	sort.Strings(pkValues)

	return strings.Join(pkValues, "_")
}

// cacheKey 拼缓存 key: [package_prefix:]daocache:db:table:pk.
//
// 空段会被跳过, 所以 DaoCachePackagePrefix 为空时和旧版本的 key 完全一致,
// 升级不会导致已有缓存整体失效.
func (daoBase *DaoBase) cacheKey() string {
	parts := make([]string, 0, 5)

	for _, part := range []string{
		DaoCachePackagePrefix,
		DaoCachePrefix,
		daoBase.modelImpl.DBName(),
		daoBase.TableName(),
		daoBase.PKFieldsStr(),
	} {
		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, ":")
}

func (daoBase *DaoBase) GetUniqKey() string {
	dbName := daoBase.modelImpl.DBName()
	tableName := daoBase.TableName()

	return fmt.Sprintf("%s:%s:%s", dbName, tableName, daoBase.PKFieldsStr())
}

// CacheKey 返回该行实际的缓存 key, 调用方在事务提交后需要手动失效时使用.
func (daoBase *DaoBase) CacheKey() string {
	return daoBase.cacheKey()
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

	if err := cacheUtil.Set(ctx, cacheKey, daoBase.modelImpl, daoBase.cacheExpireTS); err != nil {
		logger.Errorf(ctx, "dao setCache key:%s error:%+v", cacheKey, err)
		daoBase.deleteCache(ctx) // 更新失败删除
	}
}

// 删除缓存
func (daoBase *DaoBase) deleteCache(ctx context.Context) {
	cacheUtil := daoBase.cacheUtil
	cacheKey := daoBase.cacheKey()

	if err := cacheUtil.Del(ctx, cacheKey); err != nil {
		logger.Errorf(ctx, "dao deleteCache key:%s, error:%+v", cacheKey, err)
	}
}

// refreshCache 事务外写入最新值; 事务内改为提交后失效.
func (daoBase *DaoBase) refreshCache(ctx context.Context) {
	if !daoBase.useCache {
		return
	}

	if daoBase.tx == nil {
		daoBase.setCache(ctx)

		return
	}

	daoBase.invalidateCache(ctx)
}

// invalidateCache 事务外立即删缓存, 事务内推迟到 COMMIT 成功之后.
func (daoBase *DaoBase) invalidateCache(ctx context.Context) {
	if !daoBase.useCache {
		return
	}

	if daoBase.tx == nil {
		daoBase.deleteCache(ctx)

		return
	}

	if registry, ok := daoBase.tx.(afterCommitRegistry); ok {
		registry.AddAfterCommit(daoBase.DeleteCache)
	}
}

// DeleteCache 暴露给事务提交后的统一失效使用.
func (daoBase *DaoBase) DeleteCache(ctx context.Context) {
	if !daoBase.useCache {
		return
	}

	daoBase.deleteCache(ctx)
}

// setFieldDefaultValue 设置字段的默认值 (主键为空 + 新建时的初始化).
func (daoBase *DaoBase) setFieldDefaultValue() {
	for n, f := range daoBase.modelDef.Fields {
		sf := daoBase.modelReflectValue.FieldByName(n)

		if !sf.CanSet() {
			continue
		}

		switch f.Kind {
		case reflect.Bool:
			boolVal := false
			if f.Tag.HaveColumnDefault {
				boolVal = f.Tag.ColumnDefault != "0" && f.Tag.ColumnDefault != "false"
			}

			sf.SetBool(boolVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			sf.SetInt(daoBase.parseIntDefault(f, f.StructField.Type.Name()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			uint64Val := uint64(0)
			if f.Tag.HaveColumnDefault {
				i, err := strconv.ParseUint(f.Tag.ColumnDefault, 10, 64)
				if err != nil {
					panicCannotHandleFieldType(daoBase.modelDef.FullName, n, f.StructField.Type.Name(), err)
				}
				uint64Val = i
			}

			sf.SetUint(uint64Val)
		case reflect.Float32, reflect.Float64:
			float64Val := float64(0)
			if f.Tag.HaveColumnDefault {
				fv, err := strconv.ParseFloat(f.Tag.ColumnDefault, 64)
				if err != nil {
					panicCannotHandleFieldType(daoBase.modelDef.FullName, n, f.StructField.Type.Name(), err)
				}
				float64Val = fv
			}

			sf.SetFloat(float64Val)
		case reflect.String:
			sf.SetString(f.Tag.ColumnDefault)
		}
	}
}

func (daoBase *DaoBase) parseIntDefault(f *Field, fieldTypeName string) int64 {
	if !f.Tag.HaveColumnDefault {
		return 0
	}

	i, err := strconv.ParseInt(f.Tag.ColumnDefault, 10, 64)
	if err != nil {
		panicCannotHandleFieldType(daoBase.modelDef.FullName, f.Name, fieldTypeName, err)
	}

	return i
}

// changedColumnValues 返回自 Load 之后真正变化的非主键列.
//
// 只更新变化的列: 全列更新会把并发场景下别人刚改的列覆盖回去.
func (daoBase *DaoBase) changedColumnValues() map[string]interface{} {
	updateValueMap := make(map[string]interface{})

	for fieldName, field := range daoBase.modelDef.Fields {
		if field.IsPrimaryKey() {
			continue
		}

		current := daoBase.modelReflectValue.FieldByName(fieldName).Interface()

		if oldValue, ok := daoBase.oldDataMap[field.ColumnName]; ok && valuesEqual(oldValue, current) {
			continue
		}

		updateValueMap[field.ColumnName] = current
	}

	return updateValueMap
}

// create 自动填充某些字段值
func (daoBase *DaoBase) createAutoFillValue() {
	now := time.Now()

	daoBase.fillTimeField(daoBase.fieldNameCreatedAt, now)
	daoBase.fillTimeField(daoBase.fieldNameUpdatedAt, now)
}

// updateAutoFillValue 自动填充 updated_at.
//
// 语义: 调用方没有显式修改 updated_at 时, Save 就把它推到当前时间 (touch);
// 调用方显式赋值时不覆盖. 因此无改动的 Save 仍会产生一条只更新 updated_at 的 SQL,
// 这既是原库的约定, 也保证了"有人 Save 过"这件事在 updated_at 上可见.
func (daoBase *DaoBase) updateAutoFillValue() {
	sf := daoBase.modelReflectValue.FieldByName(daoBase.fieldNameUpdatedAt)

	if !sf.IsValid() || sf.Kind() != reflect.Struct || sf.Type() != timeType || !sf.CanSet() {
		return
	}

	if oldValue, ok := daoBase.oldDataMap[daoBase.columnNameOf(daoBase.fieldNameUpdatedAt)]; ok &&
		!valuesEqual(oldValue, sf.Interface()) {
		// 调用方自己改过, 尊重它.
		return
	}

	sf.Set(reflect.ValueOf(daoBase.roundTimeForField(daoBase.fieldNameUpdatedAt, time.Now())))
}

// roundTimeFields 把所有声明了 precision 的时间列按精度舍入.
//
// 调用方显式赋值时也要归一化: gorm 的 precision 不改变传入值, 库里会被 MySQL 舍入,
// 不处理的话内存里的 CreatedAt/UpdatedAt 和库里的值永远不相等.
func (daoBase *DaoBase) roundTimeFields() {
	for fieldName, field := range daoBase.modelDef.Fields {
		if field.Kind != reflect.Struct || field.TimePrecision <= 0 {
			continue
		}

		sf := daoBase.modelReflectValue.FieldByName(fieldName)
		if !sf.IsValid() || sf.Type() != timeType || !sf.CanSet() {
			continue
		}

		value, ok := sf.Interface().(time.Time)
		if !ok || value.IsZero() {
			continue
		}

		sf.Set(reflect.ValueOf(roundToFieldPrecision(value, field.TimePrecision)))
	}
}

// roundTimeForField 只在字段声明了 precision 时舍入, 否则原样返回.
func (daoBase *DaoBase) roundTimeForField(fieldName string, value time.Time) time.Time {
	field, ok := daoBase.modelDef.Fields[fieldName]
	if !ok {
		return value
	}

	return roundToFieldPrecision(value, field.TimePrecision)
}

// columnNameOf 返回字段对应的列名, 字段不存在时返回空串.
func (daoBase *DaoBase) columnNameOf(fieldName string) string {
	if field, ok := daoBase.modelDef.Fields[fieldName]; ok {
		return field.ColumnName
	}

	return ""
}

// fillTimeField 只在字段是 time.Time 且为零值时填充.
func (daoBase *DaoBase) fillTimeField(fieldName string, value time.Time) {
	sf := daoBase.modelReflectValue.FieldByName(fieldName)

	if !sf.IsValid() || sf.Kind() != reflect.Struct || sf.Type() != timeType || !sf.CanSet() {
		return
	}

	if isEmptyTime(sf) {
		sf.Set(reflect.ValueOf(daoBase.roundTimeForField(fieldName, value)))
	}
}
