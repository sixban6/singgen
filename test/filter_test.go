package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/filter"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestNodeFilter(t *testing.T) {
	// 创建测试节点
	outbounds := []transformer.Outbound{
		{Tag: "🇭🇰 香港 01", Type: "shadowsocks"},
		{Tag: "🇭🇰 香港 02", Type: "shadowsocks"},
		{Tag: "🇯🇵 日本 01", Type: "shadowsocks"},
		{Tag: "🇺🇸 美国 01", Type: "shadowsocks"},
		{Tag: "🇭🇰 官网：www.example.com", Type: "shadowsocks"},
		{Tag: "流量剩余节点", Type: "shadowsocks"},
		{Tag: "sec_us1", Type: "hysteria2"},
		{Tag: "sec_us2", Type: "hysteria2"},
	}

	nodeFilter := filter.NewNodeFilter()

	// 测试包含过滤器（香港节点）
	t.Run("TestIncludeFilter", func(t *testing.T) {
		rules := []filter.FilterRule{
			{
				Action:   filter.ActionInclude,
				Keywords: []string{"香港|港|HK|Hong Kong|🇭🇰"},
			},
		}

		result := nodeFilter.Filter(outbounds, rules)
		expected := []string{"🇭🇰 香港 01", "🇭🇰 香港 02", "🇭🇰 官网：www.example.com"}

		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}

		for i, tag := range result {
			if tag != expected[i] {
				t.Errorf("Expected %s, got %s", expected[i], tag)
			}
		}
	})

	// 测试排除过滤器（排除包含关键词的节点）
	t.Run("TestExcludeFilter", func(t *testing.T) {
		rules := []filter.FilterRule{
			{
				Action:   filter.ActionExclude,
				Keywords: []string{"网站|地址|剩余|过期|时间|有效|官网|流量|sec"},
			},
		}

		result := nodeFilter.Filter(outbounds, rules)
		expected := []string{"🇭🇰 香港 01", "🇭🇰 香港 02", "🇯🇵 日本 01", "🇺🇸 美国 01"}

		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}

		for _, expectedTag := range expected {
			found := false
			for _, actualTag := range result {
				if actualTag == expectedTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tag %s not found in result", expectedTag)
			}
		}
	})

	// 测试组合过滤器（包含香港，但排除官网）
	t.Run("TestCombinedFilter", func(t *testing.T) {
		rules := []filter.FilterRule{
			{
				Action:   filter.ActionInclude,
				Keywords: []string{"香港|港|HK|Hong Kong|🇭🇰"},
			},
			{
				Action:   filter.ActionExclude,
				Keywords: []string{"官网"},
			},
		}

		result := nodeFilter.Filter(outbounds, rules)
		expected := []string{"🇭🇰 香港 01", "🇭🇰 香港 02"}

		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}

		for i, tag := range result {
			if tag != expected[i] {
				t.Errorf("Expected %s, got %s", expected[i], tag)
			}
		}
	})

	// 测试包含过滤器（sec前缀）
	t.Run("TestSecPrefixFilter", func(t *testing.T) {
		rules := []filter.FilterRule{
			{
				Action:   filter.ActionInclude,
				Keywords: []string{"sec_"},
			},
		}

		result := nodeFilter.Filter(outbounds, rules)
		expected := []string{"sec_us1", "sec_us2"}

		if len(result) != len(expected) {
			t.Errorf("Expected %d results, got %d", len(expected), len(result))
		}

		for i, tag := range result {
			if tag != expected[i] {
				t.Errorf("Expected %s, got %s", expected[i], tag)
			}
		}
	})

	// 测试空规则（应该返回所有节点）
	t.Run("TestEmptyRules", func(t *testing.T) {
		var rules []filter.FilterRule

		result := nodeFilter.Filter(outbounds, rules)

		if len(result) != len(outbounds) {
			t.Errorf("Expected %d results, got %d", len(outbounds), len(result))
		}
	})
}
