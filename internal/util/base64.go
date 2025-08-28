package util

import (
	"encoding/base64"
	"strings"
)

func DecodeBase64URLSafe(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func EncodeBase64URLSafe(b []byte) string {
	return base64.URLEncoding.EncodeToString(b)
}

func DecodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

func EncodeBase64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}