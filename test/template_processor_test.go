package test

import (
	"fmt"
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestRemoveEmptyOutbounds(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]any
		outbounds []transformer.Outbound
		expected  map[string]any
	}{
		{
			name: "删除完全过滤的outbound",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "AdBlock",
						"type":      "selector",
						"outbounds": []any{}, // 空数组
					},
					map[string]any{
						"tag":       "Proxy",
						"type":      "selector",
						"outbounds": []any{"node1", "node2"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "node1", Type: "vmess"},
				{Tag: "node2", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Proxy",
						"type":      "selector",
						"outbounds": []any{"node1", "node2"},
					},
				},
			},
		},
		{
			name: "删除引用不存在节点的outbound",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "ABC",
						"type":      "selector",
						"outbounds": []any{"A", "B", "C"}, // 不存在的节点
					},
					map[string]any{
						"tag":       "Proxy",
						"type":      "selector",
						"outbounds": []any{"node1", "node2"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "node1", Type: "vmess"},
				{Tag: "node2", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Proxy",
						"type":      "selector",
						"outbounds": []any{"node1", "node2"},
					},
				},
			},
		},
		{
			name: "保留包含有效节点的outbound",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Mixed",
						"type":      "selector",
						"outbounds": []any{"node1", "nonexistent", "node2"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "node1", Type: "vmess"},
				{Tag: "node2", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Mixed",
						"type":      "selector",
						"outbounds": []any{"node1", "node2"},
					},
				},
			},
		},
		{
			name: "删除所有outbound后数组为空",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Empty1",
						"type":      "selector",
						"outbounds": []any{},
					},
					map[string]any{
						"tag":       "Empty2",
						"type":      "selector",
						"outbounds": []any{"nonexistent"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "node1", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := template.NewTemplateProcessor()
			processor.ProcessAllPlaceholders(tt.config, tt.outbounds)
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestRemoveEmptyOutboundsWithEmoji(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]any
		outbounds []transformer.Outbound
		expected  map[string]any
	}{
		{
			name: "带emoji的节点名称",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "US-Proxy",
						"type":      "selector",
						"outbounds": []any{"🇺🇸 US Node", "🇯🇵 JP Node"},
					},
					map[string]any{
						"tag":       "Empty",
						"type":      "selector",
						"outbounds": []any{"🇫🇷 FR Node"}, // 不存在
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "🇺🇸 US Node", Type: "vmess"},
				{Tag: "🇯🇵 JP Node", Type: "vmess"},
				{Tag: "Regular Node", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "US-Proxy",
						"type":      "selector",
						"outbounds": []any{"🇺🇸 US Node", "🇯🇵 JP Node"},
					},
				},
			},
		},
		{
			name: "混合emoji和常规节点",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Mixed",
						"type":      "selector",
						"outbounds": []any{"🚀 Fast", "normal", "🌟 Premium"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "🚀 Fast", Type: "vmess"},
				{Tag: "normal", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "Mixed",
						"type":      "selector",
						"outbounds": []any{"🚀 Fast", "normal"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := template.NewTemplateProcessor()
			processor.ProcessAllPlaceholders(tt.config, tt.outbounds)
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestCascadingDelete(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]any
		outbounds []transformer.Outbound
		expected  map[string]any
	}{
		{
			name: "级联删除A->B->C",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "A",
						"type":      "selector",
						"outbounds": []any{"B"},
					},
					map[string]any{
						"tag":       "B",
						"type":      "selector",
						"outbounds": []any{"C"},
					},
					map[string]any{
						"tag":       "C",
						"type":      "selector",
						"outbounds": []any{"nonexistent"}, // 不存在的节点
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "realnode", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{},
			},
		},
		{
			name: "部分级联删除",
			config: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "A",
						"type":      "selector",
						"outbounds": []any{"B", "realnode"},
					},
					map[string]any{
						"tag":       "B",
						"type":      "selector",
						"outbounds": []any{"C"},
					},
					map[string]any{
						"tag":       "C",
						"type":      "selector",
						"outbounds": []any{"nonexistent"},
					},
				},
			},
			outbounds: []transformer.Outbound{
				{Tag: "realnode", Type: "vmess"},
			},
			expected: map[string]any{
				"outbounds": []any{
					map[string]any{
						"tag":       "A",
						"type":      "selector",
						"outbounds": []any{"realnode"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := template.NewTemplateProcessor()
			processor.ProcessAllPlaceholders(tt.config, tt.outbounds)
			assert.Equal(t, tt.expected, tt.config)
		})
	}
}

func TestNestedStructures(t *testing.T) {
	config := map[string]any{
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"outbound": "proxy",
				},
			},
		},
		"outbounds": []any{
			map[string]any{
				"tag":       "empty",
				"type":      "selector",
				"outbounds": []any{},
			},
			map[string]any{
				"tag":       "proxy",
				"type":      "selector",
				"outbounds": []any{"node1"},
			},
		},
	}

	outbounds := []transformer.Outbound{
		{Tag: "node1", Type: "vmess"},
	}

	expected := map[string]any{
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"outbound": "proxy",
				},
			},
		},
		"outbounds": []any{
			map[string]any{
				"tag":       "proxy",
				"type":      "selector",
				"outbounds": []any{"node1"},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)
	assert.Equal(t, expected, config)
}

// 基准测试
func BenchmarkRemoveEmptyOutbounds(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size-%d", size), func(b *testing.B) {
			config := generateTestConfig(size)
			outbounds := generateTestOutbounds(size)
			processor := template.NewTemplateProcessor()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// 重新创建配置以避免修改原始数据
				testConfig := deepCopyConfig(config)
				processor.ProcessAllPlaceholders(testConfig, outbounds)
			}
		})
	}
}

// 辅助函数：生成测试配置
func generateTestConfig(size int) map[string]any {
	outbounds := make([]any, size)
	for i := 0; i < size; i++ {
		outbounds[i] = map[string]any{
			"tag":       fmt.Sprintf("outbound-%d", i),
			"type":      "selector",
			"outbounds": []any{fmt.Sprintf("node-%d", i%10)}, // 10个节点循环
		}
	}

	return map[string]any{
		"outbounds": outbounds,
	}
}

// 辅助函数：生成测试节点
func generateTestOutbounds(size int) []transformer.Outbound {
	outbounds := make([]transformer.Outbound, 10) // 只有10个真实节点
	for i := 0; i < 10; i++ {
		outbounds[i] = transformer.Outbound{
			Tag:  fmt.Sprintf("node-%d", i),
			Type: "vmess",
		}
	}
	return outbounds
}

// 辅助函数：深拷贝配置
func deepCopyConfig(original map[string]any) map[string]any {
	copy := make(map[string]any)
	for k, v := range original {
		switch val := v.(type) {
		case map[string]any:
			copy[k] = deepCopyConfig(val)
		case []any:
			newSlice := make([]any, len(val))
			for i, item := range val {
				if itemMap, ok := item.(map[string]any); ok {
					newSlice[i] = deepCopyConfig(itemMap)
				} else {
					newSlice[i] = item
				}
			}
			copy[k] = newSlice
		default:
			copy[k] = v
		}
	}
	return copy
}
