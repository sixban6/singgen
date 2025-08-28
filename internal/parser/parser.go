package parser

import (
	"strings"

	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
	"go.uber.org/zap"
)

type Parser interface {
	Accept(mediaTypeHint string, raw []byte) bool
	Parse(raw []byte) ([]model.Node, error)
}

var Registry = make(map[string]func() Parser)

func Register(name string, factory func() Parser) {
	Registry[name] = factory
}

// DetectFormat 智能检测数据格式，支持mediaTypeHint优化
func DetectFormat(raw []byte) string {
	return DetectFormatWithHint(raw, "")
}

// DetectFormatWithHint 使用提示信息检测格式以优化性能
func DetectFormatWithHint(raw []byte, mediaTypeHint string) string {
	// 验证输入
	validator := NewInputValidator()
	if err := validator.ValidateInput(raw); err != nil {
		if util.L != nil {
			util.L.Warn("Input validation failed", zap.Error(err))
		}
		return "unknown"
	}

	data := validator.SanitizeString(string(raw))
	
	// 根据mediaTypeHint进行快速检测
	if mediaTypeHint != "" {
		if quickFormat := detectWithHint(data, mediaTypeHint); quickFormat != "" {
			return quickFormat
		}
	}
	
	// Try to decode base64 if it looks like base64 data
	if isLikelyBase64(data) {
		if decoded, err := util.DecodeBase64(data); err == nil {
			data = validator.SanitizeString(string(decoded))
		}
	}
	
	// 提取协议提示进行优化
	protocolHints := validator.ExtractProtocolHints(data)
	if len(protocolHints) == 1 {
		// 如果只检测到一种协议，直接返回
		return protocolHints[0]
	}
	
	lines := strings.Split(data, "\n")
	var protocols []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "vmess://") {
			protocols = append(protocols, "vmess")
		} else if strings.HasPrefix(line, "vless://") {
			protocols = append(protocols, "vless")
		} else if strings.HasPrefix(line, "trojan://") {
			protocols = append(protocols, "trojan")
		} else if strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hy2://") {
			protocols = append(protocols, "hysteria2")
		} else if strings.HasPrefix(line, "ss://") {
			protocols = append(protocols, "shadowsocks")
		}
	}
	
	if len(protocols) == 0 {
		return "unknown"
	}
	
	if len(protocols) == 1 {
		return protocols[0]
	}
	
	firstProtocol := protocols[0]
	for _, protocol := range protocols[1:] {
		if protocol != firstProtocol {
			return "mixed"
		}
	}
	
	return firstProtocol
}

// detectWithHint 基于提示快速检测格式
func detectWithHint(data, hint string) string {
	switch strings.ToLower(hint) {
	case "vmess":
		if strings.Contains(data, `"v":`) && strings.Contains(data, `"ps":`) {
			return "vmess"
		}
	case "base64":
		if isLikelyBase64(data) {
			return DetectFormat([]byte(data))
		}
	case "json":
		// 检查是否是VMess JSON格式
		if strings.Contains(data, `"v":`) && strings.Contains(data, `"add":`) {
			return "vmess"
		}
	}
	return ""
}

// isLikelyBase64 checks if the data looks like base64 encoded content
func isLikelyBase64(data string) bool {
	// Check if it's mostly base64 characters and has no protocol prefixes
	if strings.Contains(data, "://") {
		return false
	}
	
	// Base64 data should not contain spaces or newlines (except at the end)
	trimmed := strings.TrimSpace(data)
	if strings.Contains(trimmed, " ") {
		return false
	}
	
	// Should be mostly base64 characters (A-Z, a-z, 0-9, +, /, =)
	base64Chars := 0
	for _, r := range trimmed {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' {
			base64Chars++
		}
	}
	
	// If more than 90% are base64 characters, likely base64
	return float64(base64Chars)/float64(len(trimmed)) > 0.9
}