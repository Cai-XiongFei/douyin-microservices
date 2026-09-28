package tool

import (
	"crypto/md5"
	"encoding/hex"
)

func Md5Encrypt(value string) string {
	result := md5.Sum([]byte(value))
	return hex.EncodeToString(result[:])
}
