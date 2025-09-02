package test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/stretchr/testify/assert"
)

// TestE2EWithRealTestNodes 端到端测试：使用test-nodes.txt中的真实节点数据
func TestE2EWithRealTestNodes(t *testing.T) {
	// 读取test-nodes.txt文件
	file, err := os.Open("../test-nodes.txt")
	if err != nil {
		t.Skipf("Skip E2E test: cannot open test-nodes.txt: %v", err)
		return
	}
	defer file.Close()

	var nodeUrls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			nodeUrls = append(nodeUrls, line)
		}
	}

	if len(nodeUrls) == 0 {
		t.Skip("Skip E2E test: no valid node URLs found in test-nodes.txt")
		return
	}

	// 解析节点
	p := &parser.MixedParser{}
	nodeData := strings.Join(nodeUrls, "\n")
	nodes, err := p.Parse([]byte(nodeData))
	if err != nil {
		t.Fatalf("Failed to parse nodes: %v", err)
	}

	if len(nodes) == 0 {
		t.Skip("Skip E2E test: no valid nodes parsed")
		return
	}

	t.Logf("Parsed %d nodes from test-nodes.txt", len(nodes))

	// 转换为outbound
	transformer := transformer.NewSingBoxTransformer()
	outbounds, err := transformer.Transform(nodes)
	if err != nil {
		t.Fatalf("Failed to transform nodes: %v", err)
	}

	// 测试用例：完全模拟用户需求场景
	config := map[string]any{
		"log": map[string]any{
			"level": "info",
		},
		"inbounds": []any{
			map[string]any{
				"type":   "mixed",
				"listen": "127.0.0.1",
				"port":   7890,
			},
		},
		"outbounds": []any{
			// 这个会被完全过滤掉
			map[string]any{
				"tag":       "AdBlock",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "exclude",
					"keywords": []string{".*"}, // 排除所有
				},
			},
			// 保留所有节点
			map[string]any{
				"tag":       "Proxy",
				"type":      "selector",
				"outbounds": []any{"{all}"},
			},
			// 只保留美国节点
			map[string]any{
				"tag":       "US-Nodes",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"美国"},
				},
			},
			// 这个引用不存在的节点，会被删除
			map[string]any{
				"tag":       "NonExistent",
				"type":      "selector",
				"outbounds": []any{"不存在的节点1", "不存在的节点2"},
			},
			// 这个引用将被删除的AdBlock，最终也会被删除
			map[string]any{
				"tag":       "ChainToEmpty",
				"type":      "selector",
				"outbounds": []any{"AdBlock"},
			},
			// 直连
			map[string]any{
				"tag":  "direct",
				"type": "direct",
			},
			// 阻断
			map[string]any{
				"tag":  "block",
				"type": "block",
			},
		},
		"route": map[string]any{
			"rules": []any{
				map[string]any{
					"outbound": "direct",
					"domain":   []string{"example.com"},
				},
			},
			"auto_detect_interface": true,
		},
	}

	// 处理配置
	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	// 验证结果
	outboundsResult := config["outbounds"].([]any)
	
	// 打印调试信息
	t.Logf("实际outbounds数量: %d", len(outboundsResult))
	for i, outbound := range outboundsResult {
		ob := outbound.(map[string]any)
		tag := ob["tag"].(string)
		outboundsArray, hasOutbounds := ob["outbounds"].([]any)
		if hasOutbounds {
			t.Logf("  [%d] %s: %d个outbound - %v", i, tag, len(outboundsArray), outboundsArray)
		} else {
			t.Logf("  [%d] %s: 无outbounds字段", i, tag)
		}
	}
	
	// 计算预期的outbound数量
	expectedOutbounds := 4 // Proxy, US-Nodes, direct, block (AdBlock被删除，NonExistent被删除，ChainToEmpty被删除)
	
	// 验证数量
	assert.Equal(t, expectedOutbounds, len(outboundsResult), "outbounds数量不匹配")

	// 验证具体内容
	var hasProxy, hasUSNodes, hasDirect, hasBlock bool
	var proxyOutbounds, usOutbounds []any

	for _, outbound := range outboundsResult {
		ob := outbound.(map[string]any)
		tag := ob["tag"].(string)
		
		switch tag {
		case "Proxy":
			hasProxy = true
			proxyOutbounds = ob["outbounds"].([]any)
		case "US-Nodes":
			hasUSNodes = true
			usOutbounds = ob["outbounds"].([]any)
		case "direct":
			hasDirect = true
		case "block":
			hasBlock = true
		case "AdBlock", "NonExistent", "ChainToEmpty":
			t.Errorf("不应该存在的outbound: %s", tag)
		}
	}

	assert.True(t, hasProxy, "应该保留Proxy outbound")
	assert.True(t, hasUSNodes, "应该保留US-Nodes outbound")
	assert.True(t, hasDirect, "应该保留direct outbound")
	assert.True(t, hasBlock, "应该保留block outbound")

	// 验证Proxy包含所有节点
	assert.Equal(t, len(outbounds), len(proxyOutbounds), "Proxy应该包含所有节点")

	// 验证US-Nodes只包含美国节点
	assert.Greater(t, len(usOutbounds), 0, "US-Nodes应该包含至少一个美国节点")
	assert.Less(t, len(usOutbounds), len(outbounds), "US-Nodes应该少于所有节点")

	// 验证US-Nodes中确实都是美国节点
	for _, nodeTag := range usOutbounds {
		tag := nodeTag.(string)
		assert.Contains(t, tag, "美国", "US-Nodes中应该只包含美国节点，但发现: %s", tag)
	}

	// 验证其他配置未被影响
	assert.Equal(t, "info", config["log"].(map[string]any)["level"])
	assert.True(t, config["route"].(map[string]any)["auto_detect_interface"].(bool))

	t.Logf("✅ E2E测试成功:")
	t.Logf("  - 总节点数: %d", len(outbounds))
	t.Logf("  - Proxy节点数: %d", len(proxyOutbounds))
	t.Logf("  - US节点数: %d", len(usOutbounds))
	t.Logf("  - 删除的outbound: AdBlock, NonExistent, ChainToEmpty")
	t.Logf("  - 保留的outbound: Proxy, US-Nodes, direct, block")
}

// TestE2EComplexEmojiFiltering 测试复杂emoji过滤场景
func TestE2EComplexEmojiFiltering(t *testing.T) {
	// 读取test-nodes.txt文件
	file, err := os.Open("../test-nodes.txt")
	if err != nil {
		t.Skipf("Skip E2E emoji test: cannot open test-nodes.txt: %v", err)
		return
	}
	defer file.Close()

	var nodeUrls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			nodeUrls = append(nodeUrls, line)
		}
	}

	if len(nodeUrls) == 0 {
		t.Skip("Skip E2E emoji test: no valid node URLs found in test-nodes.txt")
		return
	}

	// 解析和转换节点
	p := &parser.MixedParser{}
	nodeData := strings.Join(nodeUrls, "\n")
	nodes, err := p.Parse([]byte(nodeData))
	if err != nil {
		t.Fatalf("Failed to parse nodes: %v", err)
	}

	transformer := transformer.NewSingBoxTransformer()
	outbounds, err := transformer.Transform(nodes)
	if err != nil {
		t.Fatalf("Failed to transform nodes: %v", err)
	}

	// 测试各种emoji过滤
	config := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "US-Flags", // 🇺🇸
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🇺🇸"},
				},
			},
			map[string]any{
				"tag":       "JP-Flags", // 🇯🇵
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🇯🇵"},
				},
			},
			map[string]any{
				"tag":       "Fire-Emoji", // 🔥
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🔥"},
				},
			},
			map[string]any{
				"tag":       "Lightning-Emoji", // ⚡
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"⚡"},
				},
			},
			map[string]any{
				"tag":       "France-Flags", // 🇫🇷 - 不存在，应被删除
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"🇫🇷"},
				},
			},
			map[string]any{
				"tag":       "Crown-Emoji", // 👑 - 不存在，应被删除  
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"action":   "include",
					"keywords": []string{"👑"},
				},
			},
		},
	}

	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	// 验证结果 - France-Flags和Crown-Emoji应该被删除
	outboundsResult := config["outbounds"].([]any)
	
	var foundTags []string
	for _, outbound := range outboundsResult {
		ob := outbound.(map[string]any)
		tag := ob["tag"].(string)
		foundTags = append(foundTags, tag)
		
		// 确保删除的outbound不存在
		assert.NotEqual(t, "France-Flags", tag, "France-Flags应该被删除")
		assert.NotEqual(t, "Crown-Emoji", tag, "Crown-Emoji应该被删除")
	}

	t.Logf("✅ Emoji过滤测试成功:")
	t.Logf("  - 保留的outbound: %v", foundTags)
	t.Logf("  - 删除的outbound: France-Flags, Crown-Emoji")
}