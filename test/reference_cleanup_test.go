package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestReferenceCleanup(t *testing.T) {
	// 模拟一个实际场景：ChainProxy被删除后其引用应该被清理
	config := map[string]any{
		"outbounds": []any{
			// ChainProxy会因为过滤失败被删除
			map[string]any{
				"tag":       "ChainProxy", 
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": []any{map[string]any{
					"action":   "include",
					"keywords": []string{"nonexistent"},
				}},
			},
			// DNS引用ChainProxy
			map[string]any{
				"tag":       "DNS",
				"type":      "selector",
				"outbounds": []any{"Valid", "ChainProxy", "AlsoValid"},
			},
			// 有效的outbound
			map[string]any{
				"tag":  "Valid",
				"type": "direct",
			},
			map[string]any{
				"tag":  "AlsoValid",
				"type": "block",
			},
		},
	}

	// 提供一些实际节点，但不匹配ChainProxy的过滤条件
	realOutbounds := []transformer.Outbound{
		{Tag: "🇺🇸 US Node", Type: "vmess"},
		{Tag: "🇯🇵 JP Node", Type: "vmess"},
	}

	// 处理前
	t.Logf("处理前ChainProxy存在: %v", hasOutboundInConfig(config, "ChainProxy"))
	dnsOutbounds := getOutboundReferences(config, "DNS")
	t.Logf("处理前DNS引用: %v", dnsOutbounds)

	// 执行处理
	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, realOutbounds)

	// 检查结果
	t.Logf("处理后ChainProxy存在: %v", hasOutboundInConfig(config, "ChainProxy"))
	dnsOutboundsAfter := getOutboundReferences(config, "DNS")
	t.Logf("处理后DNS引用: %v", dnsOutboundsAfter)

	// 验证ChainProxy被删除
	assert.False(t, hasOutboundInConfig(config, "ChainProxy"), "ChainProxy应该被删除")

	// 验证DNS不再引用ChainProxy
	for _, ref := range dnsOutboundsAfter {
		assert.NotEqual(t, "ChainProxy", ref, "DNS不应该再引用ChainProxy")
	}

	// 验证有效引用保留
	assert.Contains(t, dnsOutboundsAfter, "Valid")
	assert.Contains(t, dnsOutboundsAfter, "AlsoValid")
}

func hasOutboundInConfig(config map[string]any, tag string) bool {
	outbounds := config["outbounds"].([]any)
	for _, outbound := range outbounds {
		ob := outbound.(map[string]any)
		if ob["tag"] == tag {
			return true
		}
	}
	return false
}