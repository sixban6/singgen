package platform

import (
	"github.com/sixban6/singgen/internal/config"
)

// useLegacyRuleSetDownload 将客户端平台(iOS/macOS/Windows)的规则集下载通道
// 从 route.default_http_client(http_clients 机制)切换为 legacy 逐条 download_detour。
//
// 原因: sing-box 1.14.0 对 http_client 的"detour 指向空 direct 出站"校验过严,
// 经 hc-direct 的下载请求会直接报错
// ("detour to an empty direct outbound makes no sense"),
// 导致客户端上远程规则集下载失败; 而 legacy download_detour 路径的同类校验
// 已被上游修复(commit 9bee532, 包含于 1.14.0), 且该字段 1.16 才移除——
// 作为过渡期通道最可靠。上游修复 http_client 路径后可切换回 default_http_client。
func useLegacyRuleSetDownload(config *config.Config) {
	if config.Route != nil {
		delete(config.Route, "default_http_client")
	}
	config.HTTPClients = nil

	directTag := ""
	for _, ob := range config.Outbounds {
		if t, _ := ob["type"].(string); t == "direct" {
			if tag, _ := ob["tag"].(string); tag != "" {
				directTag = tag
				break
			}
		}
	}
	if directTag == "" || config.Route == nil {
		return
	}
	if ruleSets, ok := config.Route["rule_set"].([]map[string]any); ok {
		applyLegacyDetour(ruleSets, directTag)
		return
	}
	if ruleSets, ok := config.Route["rule_set"].([]any); ok {
		typed := make([]map[string]any, 0, len(ruleSets))
		for _, rs := range ruleSets {
			if m, ok := rs.(map[string]any); ok {
				typed = append(typed, m)
			}
		}
		applyLegacyDetour(typed, directTag)
	}
}

func applyLegacyDetour(ruleSets []map[string]any, directTag string) {
	for _, rs := range ruleSets {
		if t, _ := rs["type"].(string); t == "remote" {
			if _, has := rs["download_detour"]; !has {
				rs["download_detour"] = directTag
			}
		}
	}
}
