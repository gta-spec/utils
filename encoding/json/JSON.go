package _json

import (
	jsonv2 "github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

var (
	indent string
)

func SetIndent(space string) {
	indent = space
}

type Options struct {
	Space string
}

// Stringify Go 对象转换为 JSON 字符串
//
// 如果 value 实现了 MarshalJSON() ([]byte, error) 方法（即实现了 jsonv2.Marshaler 接口），
// 则 jsonv2.Marshal 会自动调用该自定义方法进行序列化。
func Stringify(value any, opts ...*Options) (string, error) {
	var options *Options
	if len(opts) > 0 {
		options = opts[0]
	}

	if options != nil && options.Space != "" {
		bytes, err := jsonv2.Marshal(value, jsontext.WithIndent(options.Space))
		if err != nil {
			return "", err
		}
		return string(bytes), nil
	}

	bytes, err := jsonv2.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// StringifyWithIndent 使用全局配置的缩进格式将 Go 对象转换为 JSON 字符串
//
// 该函数是 Stringify 的便捷方法，使用通过 SetIndent() 设置的全局缩进配置。
// 如果 value 实现了 MarshalJSON() ([]byte, error) 方法（即实现了 jsonv2.Marshaler 接口），
// 则 jsonv2.Marshal 会自动调用该自定义方法进行序列化。
//
// 使用前必须先调用 SetIndent() 设置缩进字符串，否则会使用空字符串（紧凑格式）。
//
// 示例：
//
//	SetIndent("  ")  // 设置 2 个空格缩进
//	jsonStr, _ := StringifyWithIndent(data)
//
// 参数:
//   - value: 要序列化的 Go 对象
//
// 返回:
//   - string: 格式化后的 JSON 字符串
//   - error: 序列化错误
func StringifyWithIndent(value any) (string, error) {
	return Stringify(value, &Options{
		Space: indent,
	})
}

// Parse JSON 字符串解析为 Go 对象
//
// 如果 value 指向的类型实现了 UnmarshalJSON([]byte) error 方法（即实现了 jsonv2.Unmarshaler 接口），
// 则 jsonv2.Unmarshal 会自动调用该自定义方法进行反序列化。
func Parse[T any](str string) (T, error) {
	var result T
	err := jsonv2.Unmarshal([]byte(str), &result)
	return result, err
}
