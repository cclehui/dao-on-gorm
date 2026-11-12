package daoongorm

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
)

// 模型
type Model interface {
	TableName() string
	DBName() string
	DBClient() DBClientInterface
	GetDaoBase() *DaoBase
	SetDaoBase(myDaoBase *DaoBase)
}

// 模型定义
type ModelDef struct {
	ModelType    reflect.Type
	FullName     string
	PKFields     map[string]*Field
	Fields       map[string]*Field
	PKFieldNames []string // 排序后的主键字段名: where 条件的顺序稳定, 便于比对慢日志

	mapColumnNameFieldName map[string]string
}

func NewModelDef(m interface{}) *ModelDef {
	fullName, v := GetModelFullNameAndValue(m)

	t := v.Type()

	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("model 必须是指向 struct 的指针, %s 实际是 %s", fullName, t.Kind()))
	}

	mo := &ModelDef{
		ModelType:              t,
		FullName:               fullName,
		mapColumnNameFieldName: make(map[string]string),
	}

	mf := make(map[string]*Field)
	pkFields := make(map[string]*Field)
	seenColumn := make(map[string]string)

	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)

		// 内嵌的 daoBase 等非导出字段没有 gorm tag, 直接跳过, 不要参与建表映射.
		if tf.PkgPath != "" {
			continue
		}

		f := newField(&tf)
		if f.Tag.Ignore {
			continue
		}

		if err := f.validate(); err != nil {
			panic(fmt.Sprintf("model %s 字段 %s 不合法: %v", fullName, tf.Name, err))
		}

		// 两列映射到同一个列名是配置错误: gorm 会静默取最后一个, 这里提前报出来.
		if existField, ok := seenColumn[f.ColumnName]; ok {
			panic(fmt.Sprintf("model %s 字段 %s 与 %s 映射到同一列 %s",
				fullName, tf.Name, existField, f.ColumnName))
		}
		seenColumn[f.ColumnName] = tf.Name

		if f.IsPrimaryKey() {
			pkFields[tf.Name] = f
		}

		mf[tf.Name] = f
		mo.mapColumnNameFieldName[f.ColumnName] = tf.Name
	}

	if len(mf) == 0 {
		panic(fmt.Sprintf("model %s 没有可映射的字段", fullName))
	}

	if len(pkFields) == 0 {
		panic(fmt.Sprintf("model %s 没有设定主键", fullName))
	}

	pkNames := make([]string, 0, len(pkFields))
	for name := range pkFields {
		pkNames = append(pkNames, name)
	}
	sort.Strings(pkNames)

	mo.Fields = mf
	mo.PKFields = pkFields
	mo.PKFieldNames = pkNames

	return mo
}

type Field struct {
	Name        string
	StructField *reflect.StructField
	Tag         *FieldTag
	Kind        reflect.Kind
	ColumnName  string
	// TimePrecision 时间列的小数秒精度, 来自 gorm 原生的 precision:3.
	// 0 表示 model 没有声明, 此时不做截断, 原样交给 gorm/driver 处理.
	TimePrecision int
}

func newField(sf *reflect.StructField) *Field {
	ft := NewFieldTag(sf.Tag.Get("gorm"))
	columnName := ft.ColumnName

	if columnName == "" {
		columnName = strcase.ToSnake(sf.Name)
	}

	f := &Field{
		Name:        sf.Name,
		StructField: sf,
		Tag:         ft,
		Kind:        sf.Type.Kind(),
		ColumnName:  columnName,
		//Nullable:    nullable,
	}

	if sf.Type == timeType {
		f.TimePrecision = ft.Precision
	}

	return f
}

// validate 在注册阶段拦截无法读写的字段类型.
//
// 反射赋值只支持基础类型和 time.Time; 其它类型 (map/slice/指针) 在 setFieldDefaultValue
// 里会被静默跳过, 最终表现为"列一直是零值", 很难排查, 所以注册时就失败.
func (f *Field) validate() error {
	if f.Name == "" || f.ColumnName == "" {
		return fmt.Errorf("字段名或列名为空")
	}

	switch f.Kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return nil
	case reflect.Struct:
		if f.StructField.Type == timeType {
			return nil
		}

		return fmt.Errorf("struct 类型只支持 time.Time, 实际是 %s", f.StructField.Type)
	default:
		return fmt.Errorf("不支持的类型 %s", f.StructField.Type)
	}
}

func (f *Field) IsPrimaryKey() bool {
	return f.Tag.IsPK
}

type FieldTag struct {
	ColumnName        string
	Val               string
	IsPK              bool
	Ignore            bool
	ColumnDefault     string
	HaveColumnDefault bool // 是否有设置默认值
	// Precision gorm 原生的 precision:3, 用于 datetime(3) 这类小数秒精度
	Precision int
}

func NewFieldTag(tag string) *FieldTag {
	ft := &FieldTag{Val: tag}

	if tag == "" || tag == "-" {
		ft.Ignore = true
	}

	s := strings.Split(tag, ";")

	for _, v := range s {
		if strings.EqualFold(v, "primaryKey") {
			ft.IsPK = true
		}

		if strings.HasPrefix(v, "column:") {
			// SplitN: 列名里出现冒号时不能被截断.
			if cns := strings.SplitN(v, ":", 2); len(cns) == 2 {
				ft.ColumnName = strings.TrimSpace(cns[1])
			}
		}

		// 兼容 gorm 原生的 default: 写法, 两者语义相同.
		if strings.HasPrefix(v, "column_default:") || strings.HasPrefix(v, "default:") {
			ft.HaveColumnDefault = true

			if cns := strings.SplitN(v, ":", 2); len(cns) == 2 {
				ft.ColumnDefault = strings.TrimSpace(cns[1])
			}
		}

		if strings.HasPrefix(v, "precision:") {
			if cns := strings.SplitN(v, ":", 2); len(cns) == 2 {
				if precision, err := strconv.Atoi(strings.TrimSpace(cns[1])); err == nil {
					ft.Precision = precision
				}
			}
		}
	}

	return ft
}

// GetModelFullNameAndValue 返回 pkg.Type 和可修改的 struct 值.
//
// 必须是 *struct: 值类型无法通过反射回写, 曾经的实现会直接 panic 在 Elem() 上,
// 这里换成能看懂的报错.
func GetModelFullNameAndValue(m interface{}) (string, reflect.Value) {
	if m == nil {
		panic("model 不能是 nil")
	}

	rv := reflect.ValueOf(m)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		panic(fmt.Sprintf("model 必须是指向 struct 的非空指针, 实际是 %T", m))
	}

	v := rv.Elem()
	t := v.Type()

	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("model 必须指向 struct, 实际指向 %s", t.Kind()))
	}

	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name()), v
}
