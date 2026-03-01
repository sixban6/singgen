package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestTailscaleConfigWithAuthKey(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	outbounds := []transformer.Outbound{
		{
			Type:       "vmess",
			Tag:        "test-node",
			Server:     "example.com",
			ServerPort: 443,
		},
	}

	// 测试 TSAuthKey 不为空时生成 endpoints 配置
	opts := config.TemplateOptions{
		TSAuthKey:   "ts1234567890",
		TSLanIPCIDR: "192.168.1.0/24",
		MirrorURL:   "https://mirror.example.com",
	}

	configResult := tmpl.InjectWithOptions(outbounds, opts)

	if configResult == nil {
		t.Error("Expected non-nil config")
	}

	// 验证 endpoints 被正确生成
	if configResult.Endpoints == nil {
		t.Fatal("Expected endpoints to be present")
	}

	if len(configResult.Endpoints) == 0 {
		t.Error("Expected at least one endpoint")
	}

	// 验证 ts-node endpoint 配置
	foundTSNode := false
	for _, endpoint := range configResult.Endpoints {
		if endpoint["tag"] == "ts-node" {
			foundTSNode = true

			// 验证 auth_key 被正确设置
			if authKey, ok := endpoint["auth_key"].(string); !ok || authKey != "ts1234567890" {
				t.Errorf("Expected auth_key 'ts1234567890', got %v", endpoint["auth_key"])
			}

			// 验证 advertise_routes 被正确设置
			if routes, ok := endpoint["advertise_routes"].([]string); !ok || len(routes) == 0 {
				t.Errorf("Expected advertise_routes to be set")
			} else {
				if routes[0] != "192.168.1.0/24" {
					t.Errorf("Expected advertise_routes[0] '192.168.1.0/24', got %v", routes[0])
				}
			}

			// 验证其他必需字段
			if endpoint["type"] != "tailscale" {
				t.Errorf("Expected type 'tailscale', got %v", endpoint["type"])
			}

			if endpoint["system_interface"] != true {
				t.Errorf("Expected system_interface true, got %v", endpoint["system_interface"])
			}

			if endpoint["system_interface_name"] != "tailscale0" {
				t.Errorf("Expected system_interface_name 'tailscale0', got %v", endpoint["system_interface_name"])
			}

			if endpoint["accept_routes"] != true {
				t.Errorf("Expected accept_routes true, got %v", endpoint["accept_routes"])
			}

			break
		}
	}

	if !foundTSNode {
		t.Error("Expected to find ts-node endpoint")
	}

	// 验证 route.rules 中包含 ts-node 规则
	if configResult.Route == nil {
		t.Error("Expected route config")
	}

	if routeRules, ok := configResult.Route["rules"]; ok {
		rulesArray, ok := routeRules.([]any)
		if !ok {
			t.Error("Expected rules to be an array")
		}

		hasTSTRule := false
		for _, rule := range rulesArray {
			if ruleMap, ok := rule.(map[string]any); ok {
				if outbound, ok := ruleMap["outbound"].(string); ok && outbound == "ts-node" {
					hasTSTRule = true
					break
				}
			}
		}

		if !hasTSTRule {
			t.Error("Expected to find ts-node route rule")
		}
	} else {
		t.Error("Expected route.rules to exist")
	}

	// 验证 dns.rules 中包含 dns_tailscale 规则
	if configResult.DNS == nil {
		t.Error("Expected DNS config")
	}

	if dnsRules, ok := configResult.DNS["rules"]; ok {
		rulesArray, ok := dnsRules.([]any)
		if !ok {
			t.Error("Expected DNS rules to be an array")
		}

		hasDNSTRule := false
		for _, rule := range rulesArray {
			if ruleMap, ok := rule.(map[string]any); ok {
				if server, ok := ruleMap["server"].(string); ok && server == "dns_tailscale" {
					hasDNSTRule = true
					break
				}
			}
		}

		if !hasDNSTRule {
			t.Error("Expected to find dns_tailscale DNS rule")
		}
	} else {
		t.Error("Expected DNS.rules to exist")
	}

	// 验证 dns.servers 中包含 dns_tailscale 服务
	if configResult.DNS == nil {
		t.Error("Expected DNS config")
	}

	if dnsServers, ok := configResult.DNS["servers"]; ok {
		serversArray, ok := dnsServers.([]any)
		if !ok {
			t.Error("Expected DNS servers to be an array")
		}

		hasDNSServer := false
		for _, server := range serversArray {
			if serverMap, ok := server.(map[string]any); ok {
				if tag, ok := serverMap["tag"].(string); ok && tag == "dns_tailscale" {
					hasDNSServer = true
					break
				}
			}
		}

		if !hasDNSServer {
			t.Error("Expected to find dns_tailscale DNS server")
		}
	} else {
		t.Error("Expected DNS.servers to exist")
	}
}

func TestTailscaleConfigWithEmptyAuthKey(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	outbounds := []transformer.Outbound{
		{
			Type:       "vmess",
			Tag:        "test-node",
			Server:     "example.com",
			ServerPort: 443,
		},
	}

	// 测试 TSAuthKey 为空时，删除所有 Tailscale 相关配置
	opts := config.TemplateOptions{
		TSAuthKey:   "",
		TSLanIPCIDR: "192.168.1.0/24",
		MirrorURL:   "https://mirror.example.com",
	}

	configResult := tmpl.InjectWithOptions(outbounds, opts)

	if configResult == nil {
		t.Error("Expected non-nil config")
	}

	// 验证 endpoints 被删除
	if configResult.Endpoints != nil && len(configResult.Endpoints) > 0 {
		t.Error("Expected endpoints to be empty or nil")
	}

	// 验证 route.rules 中不包含 ts-node 规则
	if configResult.Route != nil {
		if routeRules, ok := configResult.Route["rules"]; ok {
			rulesArray, ok := routeRules.([]any)
			if !ok {
				t.Error("Expected route rules to be an array")
			}

			hasTSTRule := false
			for _, rule := range rulesArray {
				if ruleMap, ok := rule.(map[string]any); ok {
					if outbound, ok := ruleMap["outbound"].(string); ok && outbound == "ts-node" {
						hasTSTRule = true
						break
					}
				}
			}

			if hasTSTRule {
				t.Error("Expected ts-node route rule to be removed")
			}
		}
	}

	// 验证 dns.rules 中不包含 dns_tailscale 规则
	if configResult.DNS != nil {
		if dnsRules, ok := configResult.DNS["rules"]; ok {
			rulesArray, ok := dnsRules.([]any)
			if !ok {
				t.Error("Expected DNS rules to be an array")
			}

			hasDNSTRule := false
			for _, rule := range rulesArray {
				if ruleMap, ok := rule.(map[string]any); ok {
					if server, ok := ruleMap["server"].(string); ok && server == "dns_tailscale" {
						hasDNSTRule = true
						break
					}
				}
			}

			if hasDNSTRule {
				t.Error("Expected dns_tailscale DNS rule to be removed")
			}
		}
	}

	// 验证 dns.servers 中不包含 dns_tailscale 服务
	if configResult.DNS != nil {
		if dnsServers, ok := configResult.DNS["servers"]; ok {
			serversArray, ok := dnsServers.([]any)
			if !ok {
				t.Error("Expected DNS servers to be an array")
			}

			hasDNSServer := false
			for _, server := range serversArray {
				if serverMap, ok := server.(map[string]any); ok {
					if tag, ok := serverMap["tag"].(string); ok && tag == "dns_tailscale" {
						hasDNSServer = true
						break
					}
				}
			}

			if hasDNSServer {
				t.Error("Expected dns_tailscale DNS server to be removed")
			}
		}
	}
}

func TestTailscaleConfigUpdateExistingEndpoints(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// 先用空 TSAuthKey 创建配置，确保模板中存在初始的 endpoints 配置
	opts1 := config.TemplateOptions{
		TSAuthKey:   "",
		TSLanIPCIDR: "",
		MirrorURL:   "https://mirror.example.com",
	}

	_ = tmpl.InjectWithOptions([]transformer.Outbound{}, opts1)

	// 然后用非空 TSAuthKey 更新配置
	opts2 := config.TemplateOptions{
		TSAuthKey:   "updated_auth_key",
		TSLanIPCIDR: "10.0.0.0/16",
		MirrorURL:   "https://mirror.example.com",
	}

	config2 := tmpl.InjectWithOptions([]transformer.Outbound{}, opts2)

	if config2 == nil {
		t.Error("Expected non-nil config")
	}

	// 验证 endpoints 被正确更新
	if config2.Endpoints == nil {
		t.Fatal("Expected endpoints to be present")
	}

	if len(config2.Endpoints) == 0 {
		t.Error("Expected at least one endpoint")
	}

	foundTSNode := false
	for _, endpoint := range config2.Endpoints {
		if endpoint["tag"] == "ts-node" {
			foundTSNode = true

			// 验证 auth_key 被更新
			if authKey, ok := endpoint["auth_key"].(string); !ok || authKey != "updated_auth_key" {
				t.Errorf("Expected auth_key 'updated_auth_key', got %v", endpoint["auth_key"])
			}

			// 验证 advertise_routes 被更新
			if routes, ok := endpoint["advertise_routes"].([]string); !ok || len(routes) == 0 {
				t.Errorf("Expected advertise_routes to be set")
			} else {
				if routes[0] != "10.0.0.0/16" {
					t.Errorf("Expected advertise_routes[0] '10.0.0.0/16', got %v", routes[0])
				}
			}

			break
		}
	}

	if !foundTSNode {
		t.Error("Expected to find ts-node endpoint")
	}
}
