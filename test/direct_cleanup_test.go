package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/stretchr/testify/assert"
)

func TestDirectCleanup(t *testing.T) {
	// 从test-config.json中提取一个简化的例子
	config := map[string]any{
		"outbounds": []any{
			// DNS引用了不存在的ChainProxy  
			map[string]any{
				"tag":       "DNS",
				"type":      "selector", 
				"outbounds": []any{"AutoSelect-HK", "ChainProxy", "block"},
			},
			map[string]any{
				"tag":  "AutoSelect-HK",
				"type": "urltest",
			},
			map[string]any{
				"tag":  "block",
				"type": "socks",
			},
			// 注意：没有ChainProxy这个outbound
		},
	}

	// 手动构建existingTags（不包含ChainProxy）
	existingTags := map[string]bool{
		"DNS":           true,
		"AutoSelect-HK": true,
		"block":         true,
		// ChainProxy 不存在
	}

	t.Logf("清理前DNS引用: %v", getOutboundReferences(config, "DNS"))

	// 直接调用清理方法
	processor := template.NewTemplateProcessor()
	processor.CleanInvalidReferencesPublic(config, existingTags)

	dnsAfter := getOutboundReferences(config, "DNS")
	t.Logf("清理后DNS引用: %v", dnsAfter)

	// 验证ChainProxy引用被清理
	for _, ref := range dnsAfter {
		assert.NotEqual(t, "ChainProxy", ref, "ChainProxy引用应该被清理")
	}

	// 验证有效引用保留
	assert.Contains(t, dnsAfter, "AutoSelect-HK")
	assert.Contains(t, dnsAfter, "block")
}