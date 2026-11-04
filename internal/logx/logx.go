// Package logx 提供内部组件共用的日志辅助函数。
// 内部组件不再依赖全局 logger，而是由调用方注入 *slog.Logger；未注入时静默丢弃。
package logx

import (
	"log/slog"
	"net/url"
	"regexp"
	"strings"
)

var discard = slog.New(slog.DiscardHandler)

// Or 返回 l；l 为 nil 时返回一个丢弃所有输出的 logger。
func Or(l *slog.Logger) *slog.Logger {
	if l == nil {
		return discard
	}
	return l
}

// RedactURL 去掉节点链接中的凭据，只保留协议、地址与名称，避免把密码写进日志。
func RedactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	scheme, rest, ok := strings.Cut(raw, "://")
	if !ok {
		return "<invalid>"
	}
	fragment := ""
	if i := strings.Index(rest, "#"); i >= 0 {
		fragment = rest[i:]
		if decoded, err := url.PathUnescape(fragment); err == nil {
			fragment = decoded
		}
		rest = rest[:i]
	}
	if i := strings.Index(rest, "?"); i >= 0 {
		rest = rest[:i]
	}
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		return scheme + "://***@" + rest[i+1:] + fragment
	}
	// 无 userinfo（如 vmess:// base64）：整体视为敏感
	return scheme + "://***" + fragment
}

var userinfoPattern = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://)[^@\s"/]+@`)

// Err 返回去除了链接凭据的错误信息（url.Parse 等错误会原样包含输入链接）。
func Err(err error) string {
	if err == nil {
		return ""
	}
	return userinfoPattern.ReplaceAllString(err.Error(), "${1}***@")
}
