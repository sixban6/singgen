package parser

import (
	"strings"
)

// isTruthyAny 解析 JSON 中可能以 bool / number / string 表示的布尔值
func isTruthyAny(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0
	case int:
		return val != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}
