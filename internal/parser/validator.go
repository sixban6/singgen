package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// MaxInputSize 最大输入数据大小 (1MB)
	MaxInputSize = 1024 * 1024
	// MaxLineLength 单行最大长度（base64数据可能很长）
	MaxLineLength = 100000
	// MaxLines 最大行数
	MaxLines = 10000
)

// InputValidator 输入验证器
type InputValidator struct{}

// NewInputValidator 创建输入验证器
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateInput 验证输入数据的安全性和合理性
func (v *InputValidator) ValidateInput(raw []byte) error {
	// 检查大小限制
	if len(raw) > MaxInputSize {
		return fmt.Errorf("input size too large: %d bytes, max allowed: %d bytes", len(raw), MaxInputSize)
	}

	if len(raw) == 0 {
		return fmt.Errorf("empty input")
	}

	// 检查是否为有效的UTF-8
	if !utf8.Valid(raw) {
		return fmt.Errorf("invalid UTF-8 encoding")
	}

	data := string(raw)
	
	// 检查行数和行长度
	lines := strings.Split(data, "\n")
	if len(lines) > MaxLines {
		return fmt.Errorf("too many lines: %d, max allowed: %d", len(lines), MaxLines)
	}

	for i, line := range lines {
		if len(line) > MaxLineLength {
			return fmt.Errorf("line %d too long: %d characters, max allowed: %d", i+1, len(line), MaxLineLength)
		}
	}

	// 检查可疑字符和模式
	if err := v.checkSuspiciousContent(data); err != nil {
		return err
	}

	return nil
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

// checkSuspiciousContent 检查可疑内容
func (v *InputValidator) checkSuspiciousContent(data string) error {
	// 检查是否包含潜在危险的模式
	suspiciousPatterns := []string{
		"javascript:",
		"data:",
		"vbscript:",
		"<script",
		"</script>",
		"eval(",
		"exec(",
		"system(",
	}

	lowerData := strings.ToLower(data)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerData, pattern) {
			return fmt.Errorf("suspicious content detected: contains %s", pattern)
		}
	}

	// 检查非ASCII字符比例（防止乱码）
	asciiCount := 0
	totalCount := 0
	for _, r := range data {
		totalCount++
		if r < 128 {
			asciiCount++
		}
	}
	
	if totalCount > 100 { // 只对足够长的输入进行检查
		asciiRatio := float64(asciiCount) / float64(totalCount)
		if asciiRatio < 0.7 { // 如果ASCII字符少于70%，可能是二进制数据或损坏数据
			return fmt.Errorf("input contains too many non-ASCII characters (%.1f%% ASCII)", asciiRatio*100)
		}
	}

	return nil
}

// ExtractProtocolHints 从数据中提取协议提示信息
func (v *InputValidator) ExtractProtocolHints(data string) []string {
	var hints []string
	
	protocols := []string{"vmess://", "vless://", "trojan://", "ss://", "hysteria2://", "hy2://"}
	
	for _, protocol := range protocols {
		if strings.Contains(data, protocol) {
			protocolName := strings.TrimSuffix(protocol, "://")
			if protocolName == "hy2" {
				protocolName = "hysteria2"
			} else if protocolName == "ss" {
				protocolName = "shadowsocks"
			}
			hints = append(hints, protocolName)
		}
	}
	
	return hints
}