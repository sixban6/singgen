package test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestDebugChainProxyCleanup(t *testing.T) {
	// 读取test-nodes.txt
	content, err := os.ReadFile("../test-nodes.txt")
	if err != nil {
		t.Skipf("无法读取test-nodes.txt: %v", err)
		return
	}

	// 解析节点
	lines := strings.Split(string(content), "\n")
	var nodeUrls []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			nodeUrls = append(nodeUrls, line)
		}
	}

	p := &parser.MixedParser{}
	nodeData := strings.Join(nodeUrls, "\n")
	nodes, err := p.Parse([]byte(nodeData))
	if err != nil {
		t.Fatalf("解析节点失败: %v", err)
	}

	transformer := transformer.NewSingBoxTransformer()
	outbounds, err := transformer.Transform(nodes)
	if err != nil {
		t.Fatalf("转换节点失败: %v", err)
	}

	// 最小化测试配置
	config := map[string]any{
		"outbounds": []any{
			// ChainProxy过滤sec_，应该被删除
			map[string]any{
				"tag":       "ChainProxy",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": []any{map[string]any{
					"action":   "include",
					"keywords": []string{"sec_"},
				}},
			},
			// DNS引用ChainProxy，应该被清理
			map[string]any{
				"tag":       "DNS",
				"type":      "selector",
				"outbounds": []any{"AutoSelect-HK", "ChainProxy", "block"},
			},
			// 正常的outbound
			map[string]any{
				"tag":       "AutoSelect-HK",
				"type":      "urltest",
				"outbounds": []any{"{all}"},
				"filter": []any{map[string]any{
					"action":   "include",
					"keywords": []string{"香港|港|HK|Hong Kong|🇭🇰"},
				}},
			},
		},
	}

	t.Logf("处理前:")
	t.Logf("ChainProxy存在: %v", hasOutbound(config, "ChainProxy"))
	t.Logf("DNS引用: %v", getOutboundReferences(config, "DNS"))

	// 处理配置
	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	t.Logf("处理后:")
	t.Logf("ChainProxy存在: %v", hasOutbound(config, "ChainProxy"))
	t.Logf("DNS引用: %v", getOutboundReferences(config, "DNS"))

	// 验证结果
	if hasOutbound(config, "ChainProxy") {
		t.Error("ChainProxy应该被删除")
	}

	dnsRefs := getOutboundReferences(config, "DNS")
	for _, ref := range dnsRefs {
		if ref == "ChainProxy" {
			t.Error("DNS不应该再引用ChainProxy")
		}
	}

	// 输出最终配置用于调试
	finalJson, _ := json.MarshalIndent(config, "", "  ")
	t.Logf("最终配置:\n%s", string(finalJson))
}

func hasOutbound(config map[string]any, tag string) bool {
	outbounds := config["outbounds"].([]any)
	for _, outbound := range outbounds {
		ob := outbound.(map[string]any)
		if ob["tag"] == tag {
			return true
		}
	}
	return false
}

func getOutboundReferences(config map[string]any, tag string) []string {
	outbounds := config["outbounds"].([]any)
	for _, outbound := range outbounds {
		ob := outbound.(map[string]any)
		if ob["tag"] == tag {
			if refs, exists := ob["outbounds"].([]any); exists {
				var result []string
				for _, ref := range refs {
					if str, ok := ref.(string); ok {
						result = append(result, str)
					}
				}
				return result
			}
		}
	}
	return nil
}
