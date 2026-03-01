package singgen_test

import (
	"context"
	"os"
	"testing"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/pkg/singgen"
	"github.com/stretchr/testify/assert"
)

// 测试 Tailscale 配置的完整生成流程
func TestTailscaleConfigGeneration(t *testing.T) {
	testSubscription := `vmess://eyJhZGQiOiJ0ZXN0MS5leGFtcGxlLmNvbSIsImFpZCI6MCwiYWxwbiI6IiIsImhvc3QiOiIiLCJpZCI6IjEyMzQ1Njc4LWFiY2QtMTIzNC1hYmNkLTEyMzQ1Njc4OWFiYyIsIm5ldCI6InRjcCIsInBhdGgiOiIiLCJwb3J0IjoxMDgwOSwicHMiOiJUZXN0IE5vZGUgMSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`

	ctx := context.Background()
	tmpFile := createTestFile(t, testSubscription)
	defer os.Remove(tmpFile)

	t.Run("With Tailscale AuthKey and IP CIDR", func(t *testing.T) {
		cfg, err := singgen.GenerateConfig(ctx, tmpFile,
			singgen.WithTemplate("v1.12"),
			singgen.WithTSAuthKey("ts1234567890abcdef"),
			singgen.WithTSLanIPCIDR("100.64.0.0/24"),
		)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证 endpoints 配置
		if cfg.Endpoints != nil && len(cfg.Endpoints) > 0 {
			var endpoint map[string]any
			for _, ep := range cfg.Endpoints {
				if tag, tagOk := ep["tag"].(string); tagOk && tag == "ts-node" {
					endpoint = ep
					break
				}
			}
			assert.NotNil(t, endpoint, "Expected ts-node endpoint configuration")
			assert.Equal(t, "ts1234567890abcdef", endpoint["auth_key"])
			assert.True(t, endpoint["accept_routes"].(bool))
			assert.Contains(t, endpoint["advertise_routes"], "100.64.0.0/24")
		} else {
			t.Fatal("Expected endpoints configuration to be present")
		}

		// 验证 DNS servers 中有 dns_tailscale
		if cfg.DNS != nil {
			servers, ok := cfg.DNS["servers"].([]any)
			assert.True(t, ok, "Expected DNS servers to be a list")

			var found bool
			for _, server := range servers {
				if serverMap, ok := server.(map[string]any); ok {
					if tag, ok := serverMap["tag"].(string); ok && tag == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Expected dns_tailscale server in DNS servers")
		}

		// 验证 DNS rules 中有引用 dns_tailscale 的规则
		if cfg.DNS != nil && cfg.DNS["rules"] != nil {
			rules, ok := cfg.DNS["rules"].([]any)
			assert.True(t, ok, "Expected DNS rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if server, ok := ruleMap["server"].(string); ok && server == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Expected DNS rule with dns_tailscale for ts.net")
		}

		// 验证 route rules 中有 ts-node 的路由规则
		if cfg.Route != nil && cfg.Route["rules"] != nil {
			rules, ok := cfg.Route["rules"].([]any)
			assert.True(t, ok, "Expected route rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if outbound, ok := ruleMap["outbound"].(string); ok && outbound == "ts-node" {
						found = true
						break
					}
				}
			}
			assert.True(t, found, "Expected route rule with ts-node")
		}
	})

	t.Run("Without Tailscale AuthKey (empty string)", func(t *testing.T) {
		cfg, err := singgen.GenerateConfig(ctx, tmpFile,
			singgen.WithTemplate("v1.12"),
			singgen.WithTSAuthKey(""),
		)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证 endpoints 已被删除
		if cfg.Endpoints != nil && len(cfg.Endpoints) > 0 {
			var found bool
			for _, ep := range cfg.Endpoints {
				if tag, tagOk := ep["tag"].(string); tagOk && tag == "ts-node" {
					found = true
					break
				}
			}
			assert.False(t, found, "Expected ts-node endpoint to be removed")
		} else {
			t.Fatal("Expected endpoints to exist")
		}

		// 验证 DNS servers 中没有 dns_tailscale
		if cfg.DNS != nil {
			servers, ok := cfg.DNS["servers"].([]any)
			assert.True(t, ok, "Expected DNS servers to be a list")

			var found bool
			for _, server := range servers {
				if serverMap, ok := server.(map[string]any); ok {
					if tag, ok := serverMap["tag"].(string); ok && tag == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected dns_tailscale server to be removed from DNS servers")
		}

		// 验证 DNS rules 中没有引用 dns_tailscale 的规则
		if cfg.DNS != nil && cfg.DNS["rules"] != nil {
			rules, ok := cfg.DNS["rules"].([]any)
			assert.True(t, ok, "Expected DNS rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if server, ok := ruleMap["server"].(string); ok && server == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected DNS rule referencing dns_tailscale to be removed")
		}

		// 验证 route rules 中没有 ts-node 的路由规则
		if cfg.Route != nil && cfg.Route["rules"] != nil {
			rules, ok := cfg.Route["rules"].([]any)
			assert.True(t, ok, "Expected route rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if outbound, ok := ruleMap["outbound"].(string); ok && outbound == "ts-node" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected route rule referencing ts-node to be removed")
		}
	})

	t.Run("Without Tailscale AuthKey (not provided)", func(t *testing.T) {
		cfg, err := singgen.GenerateConfig(ctx, tmpFile,
			singgen.WithTemplate("v1.12"),
		)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证 endpoints 已被删除
		if cfg.Endpoints != nil && len(cfg.Endpoints) > 0 {
			var found bool
			for _, ep := range cfg.Endpoints {
				if tag, tagOk := ep["tag"].(string); tagOk && tag == "ts-node" {
					found = true
					break
				}
			}
			assert.False(t, found, "Expected ts-node endpoint to be removed")
		} else {
			t.Fatal("Expected endpoints to exist")
		}

		// 验证 DNS servers 中没有 dns_tailscale
		if cfg.DNS != nil {
			servers, ok := cfg.DNS["servers"].([]any)
			assert.True(t, ok, "Expected DNS servers to be a list")

			var found bool
			for _, server := range servers {
				if serverMap, ok := server.(map[string]any); ok {
					if tag, ok := serverMap["tag"].(string); ok && tag == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected dns_tailscale server to be removed from DNS servers")
		}

		// 验证 DNS rules 中没有引用 dns_tailscale 的规则
		if cfg.DNS != nil && cfg.DNS["rules"] != nil {
			rules, ok := cfg.DNS["rules"].([]any)
			assert.True(t, ok, "Expected DNS rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if server, ok := ruleMap["server"].(string); ok && server == "dns_tailscale" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected DNS rule referencing dns_tailscale to be removed")
		}

		// 验证 route rules 中没有 ts-node 的路由规则
		if cfg.Route != nil && cfg.Route["rules"] != nil {
			rules, ok := cfg.Route["rules"].([]any)
			assert.True(t, ok, "Expected route rules to be a list")

			var found bool
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]any); ok {
					if outbound, ok := ruleMap["outbound"].(string); ok && outbound == "ts-node" {
						found = true
						break
					}
				}
			}
			assert.False(t, found, "Expected route rule referencing ts-node to be removed")
		}
	})

	t.Run("With Custom Tailscale IP CIDR", func(t *testing.T) {
		cfg, err := singgen.GenerateConfig(ctx, tmpFile,
			singgen.WithTemplate("v1.12"),
			singgen.WithTSAuthKey("ts1234567890abcdef"),
			singgen.WithTSLanIPCIDR("10.0.0.0/16"),
		)
		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// 验证 advertise_routes 包含自定义 IP CIDR
		if cfg.Endpoints != nil && len(cfg.Endpoints) > 0 {
			var endpoint map[string]any
			for _, ep := range cfg.Endpoints {
				if tag, tagOk := ep["tag"].(string); tagOk && tag == "ts-node" {
					endpoint = ep
					break
				}
			}
			assert.NotNil(t, endpoint, "Expected ts-node endpoint configuration")
			assert.Contains(t, endpoint["advertise_routes"], "10.0.0.0/16")
		}
	})
}

// 测试 Option 函数的可用性
func TestTailscaleOptions(t *testing.T) {
	t.Run("WithTSAuthKey", func(t *testing.T) {
		generator := singgen.NewGenerator(singgen.WithTSAuthKey("test-auth-key"))
		assert.NotNil(t, generator)
	})

	t.Run("WithTSLanIPCIDR", func(t *testing.T) {
		generator := singgen.NewGenerator(singgen.WithTSLanIPCIDR("10.0.0.0/16"))
		assert.NotNil(t, generator)
	})

	t.Run("Both Tailscale Options", func(t *testing.T) {
		generator := singgen.NewGenerator(
			singgen.WithTSAuthKey("test-auth-key"),
			singgen.WithTSLanIPCIDR("10.0.0.0/16"),
		)
		assert.NotNil(t, generator)
	})
}

// 测试 TemplateOptions 的结构
func TestTemplateOptionsWithTailscale(t *testing.T) {
	opts := config.TemplateOptions{
		TSAuthKey:   "test-auth-key",
		TSLanIPCIDR: "10.0.0.0/16",
	}

	assert.Equal(t, "test-auth-key", opts.TSAuthKey)
	assert.Equal(t, "10.0.0.0/16", opts.TSLanIPCIDR)
}
