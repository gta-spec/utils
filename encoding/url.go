package _encoding

import (
	"net/url"
	"strings"
)

// EncodeURI 对应 JavaScript 的 encodeURI()
// 对 URI 进行编码，但保留特殊字符（如 : / ? # & = 等）
func EncodeURI(uri string) string {
	var result strings.Builder
	for _, char := range uri {
		if shouldEscapeInURI(char) {
			encoded := percentEncode(char)
			result.WriteString(encoded)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// shouldEscapeInURI 判断字符是否需要在 URI 中编码
func shouldEscapeInURI(char rune) bool {
	if (char >= 'A' && char <= 'Z') ||
		(char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') {
		return false
	}

	switch char {
	case ';', ',', '/', '?', ':', '@', '&', '=', '+', '$', '-', '_', '.', '!', '~', '*', '\'', '(', ')', '#':
		return false
	}

	return true
}

// shouldEscapeInURIComponent 判断字符是否需要在 URI 组件中编码
func shouldEscapeInURIComponent(char rune) bool {
	if (char >= 'A' && char <= 'Z') ||
		(char >= 'a' && char <= 'z') ||
		(char >= '0' && char <= '9') {
		return false
	}

	switch char {
	case '-', '_', '.', '!', '~', '*', '\'', '(', ')':
		return false
	}

	return true
}

// percentEncode 将字符编码为 %XX 格式
func percentEncode(char rune) string {
	buf := make([]byte, 4)
	n := 0

	if char <= 0x7F {
		buf[0] = byte(char)
		n = 1
	} else if char <= 0x7FF {
		buf[0] = byte(0xC0 | (char >> 6))
		buf[1] = byte(0x80 | (char & 0x3F))
		n = 2
	} else if char <= 0xFFFF {
		buf[0] = byte(0xE0 | (char >> 12))
		buf[1] = byte(0x80 | ((char >> 6) & 0x3F))
		buf[2] = byte(0x80 | (char & 0x3F))
		n = 3
	} else {
		buf[0] = byte(0xF0 | (char >> 18))
		buf[1] = byte(0x80 | ((char >> 12) & 0x3F))
		buf[2] = byte(0x80 | ((char >> 6) & 0x3F))
		buf[3] = byte(0x80 | (char & 0x3F))
		n = 4
	}

	var result strings.Builder
	for i := 0; i < n; i++ {
		result.WriteString("%")
		result.WriteString(strings.ToUpper(formatByte(buf[i])))
	}
	return result.String()
}

// formatByte 将字节转换为两位十六进制字符串
func formatByte(b byte) string {
	const hexChars = "0123456789ABCDEF"
	return string([]byte{hexChars[b>>4], hexChars[b&0x0F]})
}

// DecodeURI 对应 JavaScript 的 decodeURI()
// 对 encodeURI 编码的 URI 进行解码
func DecodeURI(encodedURI string) (string, error) {
	decoded := strings.ReplaceAll(encodedURI, "+", "%20")
	return url.PathUnescape(decoded)
}

// EncodeURIComponent 对应 JavaScript 的 encodeURIComponent()
// 对 URI 组件进行编码，编码所有特殊字符
func EncodeURIComponent(component string) string {
	var result strings.Builder
	for _, char := range component {
		if shouldEscapeInURIComponent(char) {
			encoded := percentEncode(char)
			result.WriteString(encoded)
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// DecodeURIComponent 对应 JavaScript 的 decodeURIComponent()
// 对 encodeURIComponent 编码的组件进行解码
func DecodeURIComponent(encodedComponent string) (string, error) {
	decoded := strings.ReplaceAll(encodedComponent, "+", "%20")
	return url.QueryUnescape(decoded)
}

// EncodeURIWithParams 完整的 URI 编码（包含查询参数处理）
func EncodeURIWithParams(baseURL string, params map[string]string) string {
	if len(params) == 0 {
		return EncodeURI(baseURL)
	}

	var queryParts []string
	for key, value := range params {
		encodedKey := EncodeURIComponent(key)
		encodedValue := EncodeURIComponent(value)
		queryParts = append(queryParts, encodedKey+"="+encodedValue)
	}

	queryString := strings.Join(queryParts, "&")
	return EncodeURI(baseURL) + "?" + queryString
}
