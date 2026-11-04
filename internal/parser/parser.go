package parser

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
)

type Parser interface {
	Accept(mediaTypeHint string, raw []byte) bool
	Parse(raw []byte) ([]model.Node, error)
}

// Registry 协议名 -> 解析器工厂。仅在 init 阶段写入，之后只读。
var Registry = make(map[string]func() Parser)

var registryMu sync.Mutex

func Register(name string, factory func() Parser) {
	registryMu.Lock()
	defer registryMu.Unlock()
	Registry[name] = factory
}

// FormatUnknown / FormatMixed 为 DetectFormat 的特殊返回值
const (
	FormatUnknown = "unknown"
	FormatMixed   = "mixed"
)

// schemeProtocols 链接前缀 -> 协议名（新增协议时在此登记一次即可）
var schemeProtocols = map[string]string{
	"vmess":     constant.ProtocolVMess,
	"vless":     constant.ProtocolVLESS,
	"trojan":    constant.ProtocolTrojan,
	"hysteria2": constant.ProtocolHysteria2,
	"hy2":       constant.ProtocolHysteria2,
	"anytls":    constant.ProtocolAnyTLS,
	"tuic":      constant.ProtocolTuic,
	"ss":        constant.ProtocolSS,
}

// protocolOf 返回一行节点链接对应的协议名
func protocolOf(line string) (string, bool) {
	scheme, _, ok := strings.Cut(line, "://")
	if !ok {
		return "", false
	}
	proto, ok := schemeProtocols[strings.ToLower(scheme)]
	return proto, ok
}

// SplitLines 规范化订阅内容：清理控制字符、按需 base64 解码（标准 / URL-safe），返回非空行。
// 整个订阅只解码一次。
func SplitLines(raw []byte) []string {
	validator := NewInputValidator()
	data := validator.SanitizeString(string(raw))
	if isLikelyBase64(data) {
		compact := strings.Join(strings.Fields(data), "")
		if decoded, err := util.DecodeBase64(compact); err == nil {
			data = validator.SanitizeString(string(decoded))
		} else if decoded, err := util.DecodeBase64URLSafe(compact); err == nil {
			data = validator.SanitizeString(string(decoded))
		}
	}
	if data == "" {
		return nil
	}
	return strings.Split(data, "\n")
}

// detectLines 根据各行协议得出整体格式
func detectLines(lines []string) string {
	format := ""
	for _, line := range lines {
		proto, ok := protocolOf(line)
		if !ok {
			continue
		}
		if format == "" {
			format = proto
		} else if format != proto {
			return FormatMixed
		}
	}
	if format == "" {
		return FormatUnknown
	}
	return format
}

// DetectFormat 检测订阅格式：单一协议名、"mixed" 或 "unknown"
func DetectFormat(raw []byte) string {
	return detectLines(SplitLines(raw))
}

// DetectFormatWithHint 保留以兼容旧调用；提示信息已不再需要。
//
// Deprecated: use DetectFormat.
func DetectFormatWithHint(raw []byte, _ string) string {
	return DetectFormat(raw)
}

// ParseSubscription 解析订阅中的全部节点链接（保持原始顺序）。
// onlyProtocol 非空时只解析该协议的链接（"mixed" 表示全部）。
// 返回解析出的节点与检测到的格式；无任何有效节点时返回 constant.ErrParseFailed。
func ParseSubscription(raw []byte, onlyProtocol string, log *slog.Logger) ([]model.Node, string, error) {
	if onlyProtocol == FormatMixed {
		onlyProtocol = ""
	}
	if onlyProtocol != "" {
		if _, ok := Registry[onlyProtocol]; !ok {
			return nil, FormatUnknown, fmt.Errorf("no parser available for format: %s", onlyProtocol)
		}
	}

	lines := SplitLines(raw)
	format := detectLines(lines)

	parsers := make(map[string]Parser)
	var nodes []model.Node
	for _, line := range lines {
		proto, ok := protocolOf(line)
		if !ok || (onlyProtocol != "" && proto != onlyProtocol) {
			continue
		}
		p, ok := parsers[proto]
		if !ok {
			factory, exists := Registry[proto]
			if !exists {
				continue
			}
			p = WithLogger(factory(), log)
			parsers[proto] = p
		}
		// 单行解析失败时由各协议解析器记录（已脱敏的）警告
		parsed, err := p.Parse([]byte(line))
		if err != nil {
			continue
		}
		nodes = append(nodes, parsed...)
	}

	if len(nodes) == 0 {
		return nil, format, constant.ErrParseFailed
	}
	return nodes, format, nil
}

// isLikelyBase64 checks if the data looks like base64 encoded content
func isLikelyBase64(data string) bool {
	// Check if it's mostly base64 characters and has no protocol prefixes
	if strings.Contains(data, "://") {
		return false
	}

	// Base64 data should not contain spaces or newlines (except at the end)
	trimmed := strings.TrimSpace(data)
	if trimmed == "" || strings.Contains(trimmed, " ") {
		return false
	}

	// Should be mostly base64 characters (A-Z, a-z, 0-9, +, /, =, and URL-safe - _)
	base64Chars := 0
	for _, r := range trimmed {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' || r == '-' || r == '_' {
			base64Chars++
		}
	}

	// If more than 90% are base64 characters, likely base64
	return float64(base64Chars)/float64(len(trimmed)) > 0.9
}
