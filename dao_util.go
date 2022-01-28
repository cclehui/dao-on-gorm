package daoongorm

import (
	"reflect"
	"strconv"
	"time"

	"github.com/cclehui/dao-on-gorm/internal"
	"github.com/pkg/errors"
)

// dao结构体转 map
func DaoStructToMap(data interface{}) map[string]interface{} {
	newMap := internal.StructToMap(data)

	if createdAt, ok := newMap["created_at"]; ok {
		if value, ok2 := createdAt.(time.Time); ok2 {
			if value.Unix() < 1 {
				newMap["created_at"] = time.Now()
			}
		}
	}

	// TODO 新旧数据比较如果没有更新才赋值成当前时间
	if _, ok := newMap["updated_at"]; ok {
		newMap["updated_at"] = time.Now()
	}

	// TODO 如果字段是主键， 那么不更新

	return newMap
}

func ReflectValueToStr(kind reflect.Kind, v reflect.Value) string {
	switch kind {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Array, reflect.Chan, reflect.Complex128, reflect.Complex64,
		reflect.Func, reflect.Interface, reflect.Invalid, reflect.Map,
		reflect.Ptr, reflect.Slice, reflect.Struct, reflect.Uintptr, reflect.UnsafePointer:
		return ""
	}

	return ""
}

func panicCannotHandleFieldType(fullName, fieldName, fieldTypeName string, err error) {
	err2 := errors.Wrapf(err, "反射设置字段值失败，fullName:%s, fieldName:%s type:%s",
		fullName, fieldName, fieldTypeName)
	panic(err2)
}

func isEmptyTime(value reflect.Value) bool {
	data := value.Interface()

	if timeValue, ok := data.(time.Time); ok {
		if timeValue.IsZero() {
			return true
		}
	}

	return false
}
