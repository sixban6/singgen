package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestRealNodesIntegration(t *testing.T) {
	// 模拟真实节点数据（基于test-nodes.txt）
	realNodes := []transformer.Outbound{
		{Tag: "🇺🇸 美国-SS-节点1", Type: "shadowsocks"},
		{Tag: "🇯🇵 日本-SS-节点2", Type: "shadowsocks"},
		{Tag: "🇭🇰 香港-SS-节点3-⚡", Type: "shadowsocks"},
		{Tag: "🇸🇬 新加坡-SS-4🔥", Type: "shadowsocks"},
		{Tag: "🇬🇧 英国-SS-5💎", Type: "shadowsocks"},
		{Tag: "🚀 美国-HY2-1-🎯", Type: "hysteria2"},
		{Tag: "⚡ 日本-HY2-2-💫", Type: "hysteria2"},
		{Tag: "🔥 香港-HY2-3-🌟", Type: "hysteria2"},
		{Tag: "💎 新加坡-HY2-4-✨", Type: "hysteria2"},
		{Tag: "🌈 英国-HY2-5-🎪", Type: "hysteria2"},
	}

	tests := []struct {
		name     string
		config   map[string]any
		expected map[string]any
	}{
		{
			name: "用户需求场景 - AdBlock被完全过滤删除",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "AdBlock",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "exclude",
							"keywords": []string{".*"}, // 排除所有节点
						},
					},
					map[string]any{
						"tag":       "Proxy",
						"type":      "selector",
						"outbounds": []any{"{all}"},
					},
					map[string]any{
						"tag":       "ABC",
						"type":      "selector",
						"outbounds": []any{"A", "B", "C"}, // 不存在的节点
						"filter": map[string]any{
							"action":   "exclude",
							"keywords": []string{".*"},
						},
					},
				},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":  "Proxy",
						"type": "selector",
						"outbounds": []any{
							"🇺🇸 美国-SS-节点1", "🇯🇵 日本-SS-节点2", "🇭🇰 香港-SS-节点3-⚡", "🇸🇬 新加坡-SS-4🔥", "🇬🇧 英国-SS-5💎",
							"🚀 美国-HY2-1-🎯", "⚡ 日本-HY2-2-💫", "🔥 香港-HY2-3-🌟", "💎 新加坡-HY2-4-✨", "🌈 英国-HY2-5-🎪",
						},
					},
				},
			},
		},
		{
			name: "按国家过滤的真实场景",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "US-Nodes",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"🇺🇸|🚀.*美国"},
						},
					},
					map[string]any{
						"tag":       "JP-Nodes",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"🇯🇵|⚡.*日本"},
						},
					},
					map[string]any{
						"tag":       "HK-Nodes",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"🇭🇰|🔥.*香港"},
						},
					},
					map[string]any{
						"tag":       "FR-Nodes", // 法国节点不存在，应被删除
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"🇫🇷"},
						},
					},
					map[string]any{
						"tag":       "All",
						"type":      "selector",
						"outbounds": []any{"{all}"},
					},
				},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "US-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇺🇸 美国-SS-节点1", "🚀 美国-HY2-1-🎯"},
					},
					map[string]any{
						"tag":       "JP-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇯🇵 日本-SS-节点2", "⚡ 日本-HY2-2-💫"},
					},
					map[string]any{
						"tag":       "HK-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇭🇰 香港-SS-节点3-⚡", "🔥 香港-HY2-3-🌟"},
					},
					map[string]any{
						"tag":  "All",
						"type": "selector",
						"outbounds": []any{
							"🇺🇸 美国-SS-节点1", "🇯🇵 日本-SS-节点2", "🇭🇰 香港-SS-节点3-⚡", "🇸🇬 新加坡-SS-4🔥", "🇬🇧 英国-SS-5💎",
							"🚀 美国-HY2-1-🎯", "⚡ 日本-HY2-2-💫", "🔥 香港-HY2-3-🌟", "💎 新加坡-HY2-4-✨", "🌈 英国-HY2-5-🎪",
						},
					},
				},
			},
		},
		{
			name: "协议类型过滤场景",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "SS-Only",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"SS"},
						},
					},
					map[string]any{
						"tag":       "HY2-Only",
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"HY2"},
						},
					},
					map[string]any{
						"tag":       "VMess-Only", // VMess节点不存在，应被删除
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"VMess"},
						},
					},
					map[string]any{
						"tag":       "Chain", // 引用将被删除的VMess-Only
						"type":      "selector",
						"outbounds": []any{"VMess-Only", "SS-Only"},
					},
				},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "SS-Only",
						"type":      "selector",
						"outbounds": []any{"🇺🇸 美国-SS-节点1", "🇯🇵 日本-SS-节点2", "🇭🇰 香港-SS-节点3-⚡", "🇸🇬 新加坡-SS-4🔥", "🇬🇧 英国-SS-5💎"},
					},
					map[string]any{
						"tag":       "HY2-Only",
						"type":      "selector",
						"outbounds": []any{"🚀 美国-HY2-1-🎯", "⚡ 日本-HY2-2-💫", "🔥 香港-HY2-3-🌟", "💎 新加坡-HY2-4-✨", "🌈 英国-HY2-5-🎪"},
					},
					map[string]any{
						"tag":       "Chain",
						"type":      "selector",
						"outbounds": []any{"SS-Only"},
					},
				},
			},
		},
		{
			name: "复杂emoji匹配测试",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Fire-Nodes", // 匹配带火emoji的节点
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"🔥"},
						},
					},
					map[string]any{
						"tag":       "Lightning-Nodes", // 匹配闪电emoji的节点
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"⚡"},
						},
					},
					map[string]any{
						"tag":       "Diamond-Nodes", // 匹配钻石emoji的节点
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"💎"},
						},
					},
					map[string]any{
						"tag":       "Crown-Nodes", // 皇冠emoji不存在，应被删除
						"type":      "selector",
						"outbounds": []any{"{all}"},
						"filter": map[string]any{
							"action":   "include",
							"keywords": []string{"👑"},
						},
					},
				},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Fire-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇸🇬 新加坡-SS-4🔥", "🔥 香港-HY2-3-🌟"},
					},
					map[string]any{
						"tag":       "Lightning-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇭🇰 香港-SS-节点3-⚡", "⚡ 日本-HY2-2-💫"},
					},
					map[string]any{
						"tag":       "Diamond-Nodes",
						"type":      "selector",
						"outbounds": []any{"🇬🇧 英国-SS-5💎", "💎 新加坡-HY2-4-✨"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := template.NewTemplateProcessor()
			processor.ProcessAllPlaceholders(tt.config, realNodes)
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestRealNodesCascadingDeletion(t *testing.T) {
	realNodes := []transformer.Outbound{
		{Tag: "🇺🇸 美国-SS-节点1", Type: "shadowsocks"},
		{Tag: "🇯🇵 日本-SS-节点2", Type: "shadowsocks"},
	}

	config := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "Level1",
				"type":      "selector",
				"outbounds": []any{"Level2", "🇺🇸 美国-SS-节点1"}, // 包含有效节点
			},
			map[string]any{
				"tag":       "Level2",
				"type":      "selector",
				"outbounds": []any{"Level3"},
			},
			map[string]any{
				"tag":       "Level3",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🇫🇷"}, // 法国节点不存在
				},
			},
			map[string]any{
				"tag":       "Orphan", // 完全孤立的空节点
				"type":      "selector",
				"outbounds": []any{"NonExistent"},
			},
		},
	}

	expected := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "Level1",
				"type":      "selector",
				"outbounds": []any{"🇺🇸 美国-SS-节点1"},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, realNodes)
	assert.Equal(t, expected, config)
}
