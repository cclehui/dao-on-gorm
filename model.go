package daoongorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
)

// 模型
type Model interface {
	TableName() string
	DBName() string
	DBClient() GormDBClient
	GetDaoBase() *DaoBase
	SetDaoBase(myDaoBase *DaoBase)
}

// 模型定义
type ModelDef struct {
	ModelType              reflect.Type
	FullName               string
	PKFields               map[string]*Field
	Fields                 map[string]*Field
	mapColumnNameFieldName map[string]string
}

func NewModelDef(m interface{}) *ModelDef {
	fullName, v := GetModelFullNameAndValue(m)

	t := v.Type()

	mo := &ModelDef{
		ModelType:              t,
		FullName:               fullName,
		mapColumnNameFieldName: make(map[string]string),
	}

	hasPrimaryKey := false
	mf := make(map[string]*Field)
	pkFields := make(map[string]*Field)

	for i := 0; i < t.NumField(); i++ {
		tf := t.Field(i)

		f := newField(&tf)
		if f.Tag.Ignore {
			continue
		}

		if f.IsPrimaryKey() {
			hasPrimaryKey = true
			pkFields[tf.Name] = f
		}

		mf[tf.Name] = f
		mo.mapColumnNameFieldName[f.ColumnName] = tf.Name
	}

	if !hasPrimaryKey {
		panic("没有设定主键")
	}

	mo.Fields = mf
	mo.PKFields = pkFields

	return mo
}

type Field struct {
	Name        string
	StructField *reflect.StructField
	Tag         *FieldTag
	Kind        reflect.Kind
	ColumnName  string
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

	return f
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
			cns := strings.Split(v, ":")
			if len(cns) == 2 {
				ft.ColumnName = strings.Trim(cns[1], " ")
			}
		}

		if strings.HasPrefix(v, "column_default:") {
			ft.HaveColumnDefault = true
			cns := strings.Split(v, ":")

			if len(cns) == 2 {
				ft.ColumnDefault = cns[1]
			}
		}
	}

	return ft
}

func GetModelFullNameAndValue(m interface{}) (string, reflect.Value) {
	if m == nil {
		panic("m不能是nil")
	}

	v := reflect.ValueOf(m).Elem()
	t := v.Type()

	return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name()), v
}
