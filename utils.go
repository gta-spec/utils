package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"unsafe"
)

// IsNil 检查给定的值是否为nil
func IsNil(v any) bool {
	if v == nil {
		return true
	}
	
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	// 引用类型
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
		// 值类型
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool, reflect.String:
		// 非引用类型，检查是否为零值
		return reflect.DeepEqual(v, reflect.Zero(rv.Type()).Interface())
		// 值类型(数组)
	case reflect.Array:
		return rv.Len() == 0
		// 结构体是值类型，检查所有字段是否为零值
	case reflect.Struct:
		return reflect.DeepEqual(v, reflect.Zero(rv.Type()).Interface())
	default:
		return false
	}
}

// Ternary 三目运算函数
// condition: 条件表达式
// trueValue: 条件为真时返回的值
// falseValue: 条件为假时返回的值
func Ternary[T any](condition bool, trueValue T, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}

// FirstNonNil (泛型)查找第一个非零值
func FirstNonNil[T comparable](value T, defaultValues ...T) T {
	var zero T
	if value != zero {
		return value
	}
	
	// 遍历默认值列表，返回第一个非零值
	for _, defaultValue := range defaultValues {
		if defaultValue != zero {
			return defaultValue
		}
	}
	
	// 如果没有找到非零默认值，则返回类型的零值
	return zero
}

// GetStructProperty 传入一个对象 通过反射获取属性
func GetStructProperty(object any, props ...string) map[string]any {
	t := reflect.TypeOf(object)
	v := reflect.ValueOf(object)
	
	// 处理指针类型
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	
	ret := make(map[string]any)
	
	// 如果没有指定属性，则返回所有字段
	if len(props) == 0 {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			ret[field.Name] = value.Interface()
		}
		return ret
	}
	
	// 查找指定的字段
	for _, prop := range props {
		if field := v.FieldByName(prop); field.IsValid() {
			ret[prop] = field.Interface()
		}
	}
	
	return ret
}

// GetStructMethods 传入一个对象 通过反射获取属性, 如果没有指定属性名则返回所有方法
func GetStructMethods(object any, names ...string) map[string]any {
	v := reflect.ValueOf(object)
	
	ret := make(map[string]any)
	numMethods := v.NumMethod()
	
	for i := 0; i < numMethods; i++ {
		method := v.Type().Method(i)
		name := method.Name
		if len(names) == 0 || slices.Contains(names, name) {
			ret[name] = v.Method(i).Interface()
		}
	}
	
	return ret
}

// SnakeCase 下划线命名法
func SnakeCase(s string) string {
	if len(s) == 0 {
		return ""
	}
	
	buf := make([]byte, 0, len(s)*2)
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				prev := s[i-1]
				nextLower := false
				if i+1 < len(s) {
					next := s[i+1]
					nextLower = next >= 'a' && next <= 'z'
				}
				if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') || (prev >= 'A' && prev <= 'Z' && nextLower) {
					buf = append(buf, '_')
				}
			}
			buf = append(buf, c+32)
		} else {
			buf = append(buf, c)
		}
	}
	return string(buf)
}

// PascalCase 帕斯卡命名法(首字母大写)
func PascalCase(s string) string {
	buf := make([]byte, 0, len(s))
	nextUpper := true
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_':
			nextUpper = true
		case nextUpper:
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			buf = append(buf, c)
			nextUpper = false
		default:
			buf = append(buf, c)
		}
	}
	return string(buf)
}

// CamelCase 驼峰命名法(首字母小写)
func CamelCase(s string) string {
	buf := make([]byte, 0, len(s))
	nextUpper := false
	firstChar := true
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_':
			nextUpper = true
		case firstChar:
			if c >= 'A' && c <= 'Z' {
				c += 32
			}
			buf = append(buf, c)
			firstChar = false
		case nextUpper:
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			buf = append(buf, c)
			nextUpper = false
		default:
			buf = append(buf, c)
		}
	}
	return string(buf)
}

// Clear 安全的清零操作（默认推荐）
func Clear[T any](ptr *T) {
	var zero T
	*ptr = zero
}

// ClearUnsafe 使用 unsafe 的快速清零操作（性能优先）
func ClearUnsafe[T any](ptr *T) {
	clear((*[1]T)(unsafe.Pointer(ptr))[:])
}

// GetLocalIPHex 获取本机IP并转换为8位十六进制字符串
func GetLocalIPHex() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "00000000"
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ip := ipNet.IP.To4()
				return fmt.Sprintf("%02x%02x%02x%02x", ip[0], ip[1], ip[2], ip[3])
			}
		}
	}
	return "00000000"
}

// Stack 获取当前 goroutine 的调用栈信息
// skip: 跳过前几层调用（0表示从调用Stack的位置开始）
func Stack(skip int) string {
	var buf strings.Builder
	
	// +1 是为了跳过 Stack 函数本身
	for i := skip + 1; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		// 获取函数名
		fn := runtime.FuncForPC(pc)
		funcName := "???"
		if fn != nil {
			funcName = fn.Name()
			// 简化函数名，去掉包路径
			if idx := strings.LastIndex(funcName, "/"); idx != -1 {
				funcName = funcName[idx+1:]
			}
		}
		
		// 简化文件路径，只保留最后两部分
		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx != -1 {
			if idx2 := strings.LastIndex(file[:idx], "/"); idx2 != -1 {
				shortFile = file[idx2+1:]
			}
		}
		
		buf.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", funcName, shortFile, line))
	}
	
	return buf.String()
}

// GoModFilepath 从指定目录开始向上递归查找 go.mod 文件路径
// dir: 起始查找目录，如果为空则使用当前工作目录
// 返回值: go.mod  文件的绝对路径，如果未找到则返回空字符串
func GoModFilepath(dir string) string {
	// 如果目录为空，使用当前工作目录
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	
	// 将路径转换为绝对路径
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	
	for {
		goModPath := filepath.Join(absDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath
		}
		
		parent := filepath.Dir(absDir)
		// 如果已经到达根目录，停止查找
		if parent == absDir {
			break
		}
		absDir = parent
	}
	return ""
}
