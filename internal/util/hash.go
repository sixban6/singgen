package util

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
)

func MD5String(s string) string {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

func SHA256String(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func MD5Bytes(b []byte) string {
	h := md5.Sum(b)
	return fmt.Sprintf("%x", h)
}

func SHA256Bytes(b []byte) string {
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h)
}