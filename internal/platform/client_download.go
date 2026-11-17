package platform

import (
	"strings"

	"github.com/sixban6/singgen/internal/config"
)

// sanitizeEmptyDirectHTTPClients 清理 http_clients 中指向「空 direct 出站」的 detour 字段。
//
// 原因: sing-box 1.14+ 引入统一 http_clients 并将 remote rule_set 的 download_detour 标记为弃用
// （1.16 移除），同时对 http_client 的 detour 做了严格校验：若 detour 指向仅含 tag/type 的空 direct
// 出站（如 {"type":"direct","tag":"#DC"}），运行时初始化规则集会直接报错：
//
//	"detour to an empty direct outbound makes no sense"
//
// 省略空 direct 的 detour（保留 {"tag":"hc-direct"}）即可直接使用主机默认网络栈直连下载，
// 同时满足显式 default_http_client 要求，零弃用告警、零启动崩溃。
func sanitizeEmptyDirectHTTPClients(cfg *config.Config) {
	if cfg == nil || len(cfg.HTTPClients) == 0 {
		return
	}

	emptyDirectTags := make(map[string]bool)
	for _, ob := range cfg.Outbounds {
		typ, _ := ob["type"].(string)
		if !strings.EqualFold(strings.TrimSpace(typ), "direct") {
			continue
		}
		tag, _ := ob["tag"].(string)
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if isEmptyDirectOutbound(ob) {
			emptyDirectTags[tag] = true
		}
	}
	if len(emptyDirectTags) == 0 {
		return
	}

	for _, hc := range cfg.HTTPClients {
		if hc == nil {
			continue
		}
		if detour, _ := hc["detour"].(string); emptyDirectTags[strings.TrimSpace(detour)] {
			delete(hc, "detour")
		}
	}
}

func isEmptyDirectOutbound(ob map[string]any) bool {
	for k, v := range ob {
		if k == "type" || k == "tag" || v == nil {
			continue
		}
		return false
	}
	return true
}
