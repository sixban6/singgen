package test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

// TestDemoOutput 生成演示配置文件输出
func TestDemoOutput(t *testing.T) {
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

	if len(nodeUrls) == 0 {
		t.Skip("test-nodes.txt中没有有效节点")
		return
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

	// 用户需求场景的完整配置
	config := map[string]any{
		"log": map[string]any{
			"level":      "info",
			"timestamp":  true,
			"output":     "sing-box.log",
			"disabled":   false,
		},
		"dns": map[string]any{
			"servers": []any{
				map[string]any{
					"tag":     "google",
					"address": "8.8.8.8",
					"detour":  "Proxy",
				},
				map[string]any{
					"tag":      "local",
					"address":  "223.5.5.5",
					"detour":   "direct",
				},
			},
			"rules": []any{
				map[string]any{
					"outbound": []string{"any"},
					"server":   "local",
				},
			},
		},
		"inbounds": []any{
			map[string]any{
				"type":   "mixed",
				"listen": "127.0.0.1",
				"port":   7890,
				"users":  []any{},
			},
		},
		"outbounds": []any{
			// 这个会被完全过滤删除（用户需求1）
			map[string]any{
				"tag":       "AdBlock",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"exclude": []string{".*"}, // 过滤掉所有节点
				},
			},
			// 保留所有节点
			map[string]any{
				"tag":       "Proxy",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"default":   "auto",
			},
			// 只保留美国节点
			map[string]any{
				"tag":       "US-Nodes",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"include": []string{"美国"},
				},
			},
			// 只保留日本节点
			map[string]any{
				"tag":       "JP-Nodes",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"include": []string{"日本"},
				},
			},
			// 高速节点（emoji过滤）
			map[string]any{
				"tag":       "High-Speed",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"include": []string{"🚀|⚡|🔥"},
				},
			},
			// 这个引用不存在的节点A、B、C，应该被删除（用户需求2）
			map[string]any{
				"tag":       "ABC",
				"type":      "selector",
				"outbounds": []any{"A", "B", "C"},
				"filter": map[string]any{
					"exclude": []string{".*"},
				},
			},
			// 级联删除测试：引用将被删除的AdBlock
			map[string]any{
				"tag":       "ChainToEmpty",
				"type":      "selector",
				"outbounds": []any{"AdBlock", "fallback"},
			},
			// 法国节点（不存在，应被删除）
			map[string]any{
				"tag":       "FR-Nodes",
				"type":      "selector",
				"outbounds": []any{"{all}"},
				"filter": map[string]any{
					"include": []string{"🇫🇷|法国"},
				},
			},
			// 自动选择
			map[string]any{
				"tag":  "auto",
				"type": "urltest",
				"outbounds": []any{
					"🇺🇸 美国-SS-节点1",
					"🚀 美国-HY2-1-🎯", 
					"🇯🇵 日本-SS-节点2",
				},
				"url":      "http://www.gstatic.com/generate_204",
				"interval": "300s",
				"tolerance": 50,
			},
			// 负载均衡
			map[string]any{
				"tag":  "fallback",
				"type": "selector",
				"outbounds": []any{
					"🇸🇬 新加坡-SS-4🔥",
					"🇬🇧 英国-SS-5💎",
				},
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
					"protocol": "dns",
					"outbound": "dns-out",
				},
				map[string]any{
					"domain": []string{
						"googleapis.cn",
						"gstatic.com",
					},
					"outbound": "Proxy",
				},
				map[string]any{
					"domain_suffix": []string{".cn"},
					"outbound":      "direct",
				},
				map[string]any{
					"geoip": "private",
					"outbound": "direct",
				},
			},
			"auto_detect_interface": true,
		},
		"experimental": map[string]any{
			"clash_api": map[string]any{
				"external_controller": "127.0.0.1:9090",
				"external_ui":         "ui",
				"secret":             "",
				"external_ui_download_url": "https://mirror.ghproxy.com/https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip",
				"external_ui_download_detour": "Proxy",
			},
		},
	}

	// 保存处理前的配置
	beforeJson, _ := json.MarshalIndent(config, "", "  ")
	err = os.WriteFile("demo_config_before.json", beforeJson, 0644)
	if err != nil {
		t.Fatalf("保存处理前配置失败: %v", err)
	}

	t.Logf("✅ 生成处理前配置: demo_config_before.json")
	t.Logf("   - 节点总数: %d", len(outbounds))
	t.Logf("   - outbound配置数: %d", len(config["outbounds"].([]any)))
	
	// 处理配置
	processor := template.NewTemplateProcessor()
	processor.ProcessAllPlaceholders(config, outbounds)

	// 保存处理后的配置
	afterJson, _ := json.MarshalIndent(config, "", "  ")
	err = os.WriteFile("demo_config_after.json", afterJson, 0644)
	if err != nil {
		t.Fatalf("保存处理后配置失败: %v", err)
	}

	// 统计结果
	outboundsResult := config["outbounds"].([]any)
	var deletedOutbounds []string
	var keptOutbounds []string
	
	// 预期应该被删除的outbound
	expectedDeleted := []string{"AdBlock", "ABC", "ChainToEmpty", "FR-Nodes"}
	
	// 检查哪些被删除了
	resultTags := make(map[string]bool)
	for _, outbound := range outboundsResult {
		ob := outbound.(map[string]any)
		tag := ob["tag"].(string)
		resultTags[tag] = true
		keptOutbounds = append(keptOutbounds, tag)
	}
	
	for _, expected := range expectedDeleted {
		if !resultTags[expected] {
			deletedOutbounds = append(deletedOutbounds, expected)
		}
	}

	t.Logf("✅ 生成处理后配置: demo_config_after.json")
	t.Logf("   - 处理后outbound数: %d", len(outboundsResult))
	t.Logf("   - 删除的outbound: %v", deletedOutbounds)
	t.Logf("   - 保留的outbound: %v", keptOutbounds)

	// 输出节点详情
	fmt.Printf("\n🎯 节点处理详情:\n")
	for _, outbound := range outboundsResult {
		ob := outbound.(map[string]any)
		tag := ob["tag"].(string)
		if outboundsArray, hasOutbounds := ob["outbounds"].([]any); hasOutbounds {
			fmt.Printf("  📌 %s (%d个节点):\n", tag, len(outboundsArray))
			for i, nodeTag := range outboundsArray {
				if i < 3 {
					fmt.Printf("     - %s\n", nodeTag)
				} else if i == 3 {
					fmt.Printf("     - ... 还有%d个节点\n", len(outboundsArray)-3)
					break
				}
			}
		} else {
			fmt.Printf("  📌 %s (特殊outbound)\n", tag)
		}
	}

	fmt.Printf("\n📁 输出文件:\n")
	fmt.Printf("   - demo_config_before.json (处理前)\n")
	fmt.Printf("   - demo_config_after.json (处理后)\n")
}