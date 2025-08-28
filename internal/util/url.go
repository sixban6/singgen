package util

import (
	"net/url"
)

func ParseURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}

func QueryMap(u *url.URL) map[string]string {
	m := make(map[string]string)
	for k, v := range u.Query() {
		if len(v) > 0 {
			m[k] = v[0]
		}
	}
	return m
}

func QueryValue(u *url.URL, key string) string {
	return u.Query().Get(key)
}