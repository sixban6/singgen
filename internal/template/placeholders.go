package template

import (
	"net"
	"regexp"
	"strings"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/logx"
)

// 模板占位符的默认值（选项为空时使用）
const (
	defaultDNSLocal           = "114.114.114.114"
	defaultClientSubnet       = "223.5.5.5/24"
	defaultExternalController = "127.0.0.1:9090"
)

// placeholderPattern 匹配形如 {dns_local} 的占位符；{all} 由 TemplateProcessor 单独处理
var placeholderPattern = regexp.MustCompile(`\{[a-z_]+\}`)

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// renderPlaceholders 一次遍历替换模板中的所有显式占位符。
// 只有写成占位符的值才会被替换，模板中其它同值字面量不受影响。
func renderPlaceholders(cfg map[string]any, o config.TemplateOptions) {
	mirror := o.MirrorURL
	values := map[string]string{
		"{dns_local}":           orDefault(o.DNSLocalServer, defaultDNSLocal),
		"{client_subnet}":       orDefault(o.ClientSubnet, defaultClientSubnet),
		"{external_controller}": orDefault(o.ExternalController, defaultExternalController),
		"{ts_auth_key}":         o.TSAuthKey,
		"{ts_lan_ipcidr}":       o.TSLanIPCIDR,
	}

	walkStrings(cfg, func(s string) string {
		if !strings.Contains(s, "{") {
			return s
		}
		// 镜像为空时连同分隔符一起去掉："{mirror_url}/https://..." -> "https://..."
		if mirror == "" {
			s = strings.ReplaceAll(s, constant.MirrorURLPlaceholder+"/", "")
		}
		s = strings.ReplaceAll(s, constant.MirrorURLPlaceholder, mirror)
		return placeholderPattern.ReplaceAllStringFunc(s, func(ph string) string {
			if v, ok := values[ph]; ok {
				return v
			}
			return ph // 未知占位符（如 {all}）原样保留
		})
	})

	if unresolved := findUnresolved(cfg); len(unresolved) > 0 {
		logx.Or(o.Logger).Warn("Template contains unknown placeholders", "placeholders", unresolved)
	}
}

// findUnresolved 返回替换后仍残留的占位符（排除 {all}）
func findUnresolved(cfg map[string]any) []string {
	seen := map[string]bool{}
	var result []string
	walkStrings(cfg, func(s string) string {
		for _, ph := range placeholderPattern.FindAllString(s, -1) {
			if ph != "{all}" && !seen[ph] {
				seen[ph] = true
				result = append(result, ph)
			}
		}
		return s
	})
	return result
}

// walkStrings 递归遍历配置树并用 fn 的返回值替换每个字符串值
func walkStrings(obj any, fn func(string) string) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			if str, ok := value.(string); ok {
				v[key] = fn(str)
			} else {
				walkStrings(value, fn)
			}
		}
	case []any:
		for i, item := range v {
			if str, ok := item.(string); ok {
				v[i] = fn(str)
			} else {
				walkStrings(item, fn)
			}
		}
	case []map[string]any:
		for _, item := range v {
			walkStrings(item, fn)
		}
	}
}

// syncAPIServiceListen 将 sing-box 1.14+ api service 的监听地址与 external controller 主机保持一致，
// 避免绑定到部署机不存在的 IP
func syncAPIServiceListen(cfg map[string]any, externalController string) {
	host := externalController
	if h, _, err := net.SplitHostPort(externalController); err == nil {
		host = h
	}
	if services := asMapSlice(cfg["services"]); len(services) > 0 {
		services[0]["listen"] = host
	}
}
