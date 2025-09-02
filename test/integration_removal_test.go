package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationWithFilter(t *testing.T) {
	// 测试与原有过滤功能的集成
	config := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "AdBlock",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "exclude",
					"keywords": []string{".*"}, // 排除所有
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
				"outbounds": []any{"A", "B", "C"},
				"filter": map[string]any{
					"action":   "exclude",
					"keywords": []string{".*"}, // 排除所有
				},
			},
		},
	}

	outbounds := []transformer.Outbound{
		{Tag: "🇺🇸 US Node", Type: "vmess"},
		{Tag: "🇯🇵 JP Node", Type: "vmess"},
		{Tag: "Regular Node", Type: "vmess"},
	}

	expected := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "Proxy",
				"type":      "selector",
				"outbounds": []any{"🇺🇸 US Node", "🇯🇵 JP Node", "Regular Node"},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	assert.Equal(t, expected, config)
}

func TestComplexCascading(t *testing.T) {
	// 测试复杂的级联删除场景
	config := map[string]any{
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"outbound": "level1",
				},
			},
		},
		"outbounds": []any{
			map[string]any{
				"tag":       "level1",
				"type":      "selector",
				"outbounds": []any{"level2", "backup"},
			},
			map[string]any{
				"tag":       "level2",
				"type":      "selector",
				"outbounds": []any{"level3"},
			},
			map[string]any{
				"tag":       "level3",
				"type":      "selector",
				"outbounds": []any{"level4"},
			},
			map[string]any{
				"tag":       "level4",
				"type":      "selector",
				"outbounds": []any{}, // 空的
			},
			map[string]any{
				"tag":       "backup",
				"type":      "selector",
				"outbounds": []any{"realnode"},
			},
		},
	}

	outbounds := []transformer.Outbound{
		{Tag: "realnode", Type: "vmess"},
	}

	expected := map[string]any{
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"outbound": "level1",
				},
			},
		},
		"outbounds": []any{
			map[string]any{
				"tag":       "level1",
				"type":      "selector",
				"outbounds": []any{"level2", "backup"},
			},
			map[string]any{
				"tag":       "backup",
				"type":      "selector",
				"outbounds": []any{"realnode"},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	assert.Equal(t, expected, config)
}

func TestMixedScenario(t *testing.T) {
	// 混合场景：{all}占位符 + 过滤 + 级联删除
	config := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "FilteredUS",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🇺🇸"},
				},
			},
			map[string]any{
				"tag":       "FilteredJP",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include", 
					"keywords": []string{"🇯🇵"},
				},
			},
			map[string]any{
				"tag":       "FilteredFR", // 法国节点不存在，应该被删除
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
			map[string]any{
				"tag":       "Manual",
				"type":      "selector",
				"outbounds": []any{"FilteredFR"}, // 引用将被删除的outbound
			},
		},
	}

	outbounds := []transformer.Outbound{
		{Tag: "🇺🇸 US Node 1", Type: "vmess"},
		{Tag: "🇺🇸 US Node 2", Type: "vmess"},
		{Tag: "🇯🇵 JP Node", Type: "vmess"},
		{Tag: "🇩🇪 DE Node", Type: "vmess"},
	}

	expected := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "FilteredUS",
				"type":      "selector",
				"outbounds": []any{"🇺🇸 US Node 1", "🇺🇸 US Node 2"},
			},
			map[string]any{
				"tag":       "FilteredJP",
				"type":      "selector",
				"outbounds": []any{"🇯🇵 JP Node"},
			},
			map[string]any{
				"tag":       "All",
				"type":      "selector",
				"outbounds": []any{"🇺🇸 US Node 1", "🇺🇸 US Node 2", "🇯🇵 JP Node", "🇩🇪 DE Node"},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	assert.Equal(t, expected, config)
}