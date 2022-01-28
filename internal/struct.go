package internal

import (
	"reflect"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

// 将struct转化为map，支持嵌套的struct
// 未导出的字段不会转化到map中
// 支持的tag为structs:"map的key"  structs:"-"表明这个字段被忽略，不会转化到map中
// 底层使用github.com/fatih/structs，今后可能会统一为github.com/mitchellh/mapstructure
func StructToMap(s interface{}) map[string]interface{} {
	return structs.Map(s)
}

// 将map转化为struct，支持弱类型，比如map中为"12313"的值，可以被转化到struct中类型为int的>字段中
// 支持的tag为mapstructure:"map的key"，map中为此key的字段将被转化到tag所在的字段中
// 底层使用github.com/mitchellh/mapstructure
func MapToStruct(m map[string]interface{}, ptr interface{}) error {
	config := &mapstructure.DecoderConfig{
		DecodeHook: func(source reflect.Kind, target reflect.Kind, data interface{}) (interface{}, error) {
			// 默认对于空字符串，转化为数值类型会报错，这里将空字符串设置为"0"，即空字符串转化为数值型为0
			if source == reflect.String && data.(string) == "" && (target == reflect.Int || target == reflect.Int16 || target == reflect.Int32 ||
				target == reflect.Int64 || target == reflect.Int8 || target == reflect.Uint || target == reflect.Uint16 ||
				target == reflect.Uint32 || target == reflect.Uint64 || target == reflect.Uint8 || target == reflect.Float32 ||
				target == reflect.Float64) {
				return "0", nil
			}
			return data, nil
		},
		WeaklyTypedInput: true,
		Result:           ptr,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	err = decoder.Decode(m)
	if err != nil {
		return err
	}

	return nil
}
