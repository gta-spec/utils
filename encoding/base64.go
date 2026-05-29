package _encoding

import (
	"encoding/base64"
)

// Btoa 对应 JavaScript 的 btoa() 函数
// 将字符串编码为 Base64 格式
// Binary to ASCII
func Btoa(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// Atob 对应 JavaScript 的 atob() 函数
// 将 Base64 编码的字符串解码为原始字符串
// ASCII to Binary
func Atob(encoded string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
