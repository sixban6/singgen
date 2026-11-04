package template

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/sixban6/singgen/internal/transformer"
)

// collectReservedTags 收集模板中已占用的 outbound / endpoint tag，
// 防止订阅节点与模板分组（如 Proxy、Auto）或 endpoint（如 ts-node）重名。
func collectReservedTags(config map[string]any) map[string]bool {
	reserved := make(map[string]bool)
	for _, key := range []string{"outbounds", "endpoints"} {
		for _, item := range asMapSlice(config[key]) {
			if tag, ok := item["tag"].(string); ok && tag != "" {
				reserved[tag] = true
			}
		}
	}
	return reserved
}

// dedupeOutboundTags 保证每个节点 tag 非空且全局唯一。
// sing-box 在出现重复 outbound/endpoint tag 时会拒绝加载配置。
// 重名节点依次追加 " 2"、" 3" … 后缀，不修改传入的切片。
func dedupeOutboundTags(outbounds []transformer.Outbound, reserved map[string]bool) []transformer.Outbound {
	if len(outbounds) == 0 {
		return outbounds
	}

	used := make(map[string]bool, len(reserved)+len(outbounds))
	for tag := range reserved {
		used[tag] = true
	}

	result := make([]transformer.Outbound, len(outbounds))
	for i, ob := range outbounds {
		base := strings.TrimSpace(ob.Tag)
		if base == "" {
			base = net.JoinHostPort(ob.Server, strconv.Itoa(int(ob.ServerPort)))
		}

		tag := base
		for n := 2; used[tag]; n++ {
			tag = fmt.Sprintf("%s %d", base, n)
		}
		used[tag] = true

		result[i] = ob
		result[i].Tag = tag
	}
	return result
}

func asMapSlice(v any) []map[string]any {
	switch arr := v.(type) {
	case []map[string]any:
		return arr
	case []any:
		result := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	}
	return nil
}
