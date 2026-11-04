package parser

import (
	"strings"
	"unicode"
)

// InputValidator 输入验证器
type InputValidator struct{}

// NewInputValidator 创建输入验证器
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// SanitizeString 清理和规范化字符串
func (v *InputValidator) SanitizeString(s string) string {
	// 移除控制字符（除了换行符和制表符）
	var cleaned strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\t' || r == '\r' || !unicode.IsControl(r) {
			cleaned.WriteRune(r)
		}
	}

	result := cleaned.String()

	// 规范化空白字符
	result = strings.TrimSpace(result)

	// 移除多余的空行
	lines := strings.Split(result, "\n")
	var filteredLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			filteredLines = append(filteredLines, trimmed)
		}
	}

	return strings.Join(filteredLines, "\n")
}
