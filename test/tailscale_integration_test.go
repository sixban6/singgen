package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestTailscaleConfigIntegration(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// 测试用例1: TSAuthKey 不为空时生成完整配置
	t.Run("WithAuthKey", func(t *testing.T) {
		outbounds := []transformer.Outbound{
			{
				Type:       "vmess",
				Tag:        "test-node-1",
				Server:     "example.com",
				ServerPort: 443,
				UUID:       "12345678-abcd-1234-5678-123456789abc",
			},
			{
				Type:       "vless",
				Tag:        "test-node-2",
				Server:     "example.org",
				ServerPort: 443,
				UUID:       "87654321-dcba-4321-5678-123456789abc",
			},
		}

		opts := config.TemplateOptions{
			TSAuthKey:   "ts1234567890abcdef",
			TSLanIPCIDR: "192.168.1.0/24",
			MirrorURL:   "https://mirror.example.com",
			RemoveEmoji: true,
		}

		result := tmpl.InjectWithOptions(outbounds, opts)

		// 验证基本配置
		if result == nil {
			t.Fatal("Expected non-nil config")
		}

		// 验证 endpoints 存在且配置正确
		if result.Endpoints == nil {
			t.Fatal("Expected endpoints to be present")
		}

		if len(result.Endpoints) != 1 {
			t.Fatalf("Expected exactly 1 endpoint, got %d", len(result.Endpoints))
		}

		endpoint := result.Endpoints[0]
		if endpoint["tag"] != "ts-node" {
			t.Errorf("Expected endpoint tag 'ts-node', got %v", endpoint["tag"])
		}

		if authKey, ok := endpoint["auth_key"].(string); !ok || authKey != "ts1234567890abcdef" {
			t.Errorf("Expected auth_key 'ts1234567890abcdef', got %v", endpoint["auth_key"])
		}

		if routes, ok := endpoint["advertise_routes"].([]string); !ok || len(routes) != 1 {
			t.Errorf("Expected advertise_routes to have exactly 1 route")
		} else {
			if routes[0] != "192.168.1.0/24" {
				t.Errorf("Expected route '192.168.1.0/24', got %v", routes[0])
			}
		}

		// 验证代理节点被正确注入
		foundTestNode1 := false
		foundTestNode2 := false
		for _, outbound := range result.Outbounds {
			if outbound["tag"] == "test-node-1" {
				foundTestNode1 = true
				if outbound["type"] != "vmess" {
					t.Errorf("Expected test-node-1 to be vmess, got %v", outbound["type"])
				}
			}
			if outbound["tag"] == "test-node-2" {
				foundTestNode2 = true
				if outbound["type"] != "vless" {
					t.Errorf("Expected test-node-2 to be vless, got %v", outbound["type"])
				}
			}
		}

		if !foundTestNode1 {
			t.Error("Expected to find test-node-1 in outbounds")
		}
		if !foundTestNode2 {
			t.Error("Expected to find test-node-2 in outbounds")
		}

		// 验证路由规则包含 ts-node
		if !hasTSTRule(t, result.Route, "ts-node") {
			t.Error("Expected to find ts-node route rule")
		}

		// 验证 DNS 规则包含 dns_tailscale
		if !hasDNSTRule(t, result.DNS, "dns_tailscale") {
			t.Error("Expected to find dns_tailscale DNS rule")
		}

		// 验证 DNS 服务包含 dns_tailscale
		if !hasDNSServer(t, result.DNS, "dns_tailscale") {
			t.Error("Expected to find dns_tailscale DNS server")
		}

		// 验证配置可以序列化为 JSON
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config to JSON: %v", err)
		}

		if len(jsonData) == 0 {
			t.Error("Expected non-empty JSON output")
		}

		fmt.Printf("Config with TSAuthKey (size: %d bytes):\n%s\n", len(jsonData), jsonData)
	})

	// 测试用例2: TSAuthKey 为空时删除所有 Tailscale 配置
	t.Run("WithoutAuthKey", func(t *testing.T) {
		outbounds := []transformer.Outbound{
			{
				Type:       "vmess",
				Tag:        "test-node",
				Server:     "example.com",
				ServerPort: 443,
			},
		}

		opts := config.TemplateOptions{
			TSAuthKey:   "",
			TSLanIPCIDR: "192.168.1.0/24",
			MirrorURL:   "https://mirror.example.com",
		}

		result := tmpl.InjectWithOptions(outbounds, opts)

		// 验证 endpoints 被删除
		if result.Endpoints != nil && len(result.Endpoints) > 0 {
			t.Error("Expected endpoints to be empty or nil")
		}

		// 验证路由规则不包含 ts-node
		if hasTSTRule(t, result.Route, "ts-node") {
			t.Error("Expected ts-node route rule to be removed")
		}

		// 验证 DNS 规则不包含 dns_tailscale
		if hasDNSTRule(t, result.DNS, "dns_tailscale") {
			t.Error("Expected dns_tailscale DNS rule to be removed")
		}

		// 验证 DNS 服务不包含 dns_tailscale
		if hasDNSServer(t, result.DNS, "dns_tailscale") {
			t.Error("Expected dns_tailscale DNS server to be removed")
		}

		// 验证配置可以序列化为 JSON
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config to JSON: %v", err)
		}

		fmt.Printf("Config without TSAuthKey (size: %d bytes):\n%s\n", len(jsonData), jsonData)
	})

	// 测试用例3: 使用不同的 TSLanIPCIDR
	t.Run("WithDifferentCIDR", func(t *testing.T) {
		outbounds := []transformer.Outbound{}

		opts := config.TemplateOptions{
			TSAuthKey:   "ts9876543210",
			TSLanIPCIDR: "10.0.0.0/16",
			MirrorURL:   "https://mirror.example.com",
		}

		result := tmpl.InjectWithOptions(outbounds, opts)

		if result.Endpoints == nil {
			t.Fatal("Expected endpoints to be present")
		}

		endpoint := result.Endpoints[0]
		if routes, ok := endpoint["advertise_routes"].([]string); ok && len(routes) > 0 {
			if routes[0] != "10.0.0.0/16" {
				t.Errorf("Expected route '10.0.0.0/16', got %v", routes[0])
			}
		} else {
			t.Error("Expected advertise_routes to be set")
		}

		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config to JSON: %v", err)
		}

		fmt.Printf("Config with different CIDR (size: %d bytes):\n%s\n", len(jsonData), jsonData)
	})
}

// 辅助函数：检查路由规则中是否包含指定的 outbound
func hasTSTRule(t *testing.T, route map[string]any, target string) bool {
	if route == nil {
		return false
	}

	rules, ok := route["rules"].([]any)
	if !ok {
		return false
	}

	for _, rule := range rules {
		if ruleMap, ok := rule.(map[string]any); ok {
			if outbound, ok := ruleMap["outbound"].(string); ok && outbound == target {
				return true
			}
		}
	}

	return false
}

// 辅助函数：检查 DNS 规则中是否包含指定的 server
func hasDNSTRule(t *testing.T, dns map[string]any, target string) bool {
	if dns == nil {
		return false
	}

	rules, ok := dns["rules"].([]any)
	if !ok {
		return false
	}

	for _, rule := range rules {
		if ruleMap, ok := rule.(map[string]any); ok {
			if server, ok := ruleMap["server"].(string); ok && server == target {
				return true
			}
		}
	}

	return false
}

// 辅助函数：检查 DNS 服务中是否包含指定的 tag
func hasDNSServer(t *testing.T, dns map[string]any, target string) bool {
	if dns == nil {
		return false
	}

	servers, ok := dns["servers"].([]any)
	if !ok {
		return false
	}

	for _, server := range servers {
		if serverMap, ok := server.(map[string]any); ok {
			if tag, ok := serverMap["tag"].(string); ok && tag == target {
				return true
			}
		}
	}

	return false
}
