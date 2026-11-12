package daoongorm

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/cclehui/dao-on-gorm/internal"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

// timeType 缓存 time.Time 的反射类型, 避免在热路径里反复 TypeOf.
var timeType = reflect.TypeOf(time.Time{})

// dao结构体转 map
//
// Deprecated: 用 GetColumnMapView / 显式赋值代替. 这个函数按 structs tag 取 key,
// 并且会无条件把 updated_at 改成当前时间, 和 DaoBase 的自动填充逻辑重复.
func DaoStructToMap(data interface{}) map[string]interface{} {
	newMap := internal.StructToMap(data)

	if createdAt, ok := newMap["created_at"]; ok {
		if value, ok2 := createdAt.(time.Time); ok2 {
			if value.Unix() < 1 {
				newMap["created_at"] = time.Now()
			}
		}
	}

	if _, ok := newMap["updated_at"]; ok {
		newMap["updated_at"] = time.Now()
	}

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

// JoinPKValues 把主键值拼成稳定的字符串, 用做缓存 key 后缀.
//
// 联合主键时按字段名排序, 保证同一个结构体无论字段顺序如何都得到同一个 key;
// 值之间用 \x1f 分隔, 避免 ("a","bc") 和 ("ab","c") 拼出同一个 key.
func JoinPKValues(values []string) string {
	sorted := make([]string, len(values))
	copy(sorted, values)
	sort.Strings(sorted)

	return fmt.Sprintf("%q", sorted)
}

func panicCannotHandleFieldType(fullName, fieldName, fieldTypeName string, err error) {
	err2 := errors.Wrapf(err, "反射设置字段值失败，fullName:%s, fieldName:%s type:%s",
		fullName, fieldName, fieldTypeName)
	panic(err2)
}

func isEmptyTime(value reflect.Value) bool {
	data := value.Interface()

	if timeValue, ok := data.(time.Time); ok {
		return timeValue.IsZero()
	}

	return false
}

// roundToFieldPrecision 按字段声明的 precision 舍入时间.
//
// 为什么需要: gorm 的 precision 只影响 DDL 和 driver 写法, **不会**改你传进去的值
// (实测 gorm v1.31.2: 传 .123456789 进 datetime(3), 库里是 .123, 但结构体仍是 .123456789),
// 于是内存值和落库值不一致, "字段是否变化" 的判断会一直认为 updated_at 变了.
//
// 为什么用 Round 而不是 Truncate: MySQL 写 datetime(3) 时是四舍五入
// (.123600 -> .124000), 不是截断. 用 Truncate 会让内存值比库里小 1ms.
// gorm 自己的 mysql.NowFunc 也是 Round.
//
// precision <= 0 表示该列没有声明精度, 原样返回, 交给 gorm/driver 决定.
func roundToFieldPrecision(value time.Time, precision int) time.Time {
	if precision <= 0 {
		return value
	}

	if precision > 9 {
		precision = 9
	}

	// precision 是"秒的小数位数": 3 位 = 毫秒 = 1e6 ns.
	unit := time.Duration(1)
	for i := 0; i < 9-precision; i++ {
		unit *= 10
	}

	return value.Round(unit)
}

// valuesEqual 判断两个字段值是否语义相等.
//
// time.Time 必须用 Equal: 它带 loc 指针和单调时钟, DeepEqual 会把
// "同一时刻但 Location 实例不同" 判成变化, 导致每次 Save 都写一遍所有时间列.
func valuesEqual(left, right interface{}) bool {
	leftTime, leftOk := left.(time.Time)
	rightTime, rightOk := right.(time.Time)

	if leftOk && rightOk {
		return leftTime.Equal(rightTime)
	}

	return reflect.DeepEqual(left, right)
}

// lockForUpdate 事务内读取单行时加排他锁.
func lockForUpdate() clause.Expression {
	return clause.Locking{Strength: "UPDATE"}
}
