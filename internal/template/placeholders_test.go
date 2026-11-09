package template

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/config"
)

func TestRenderPlaceholdersOnlyTouchesPlaceholders(t *testing.T) {
	cfg := map[string]any{
		"dns": map[string]any{
			"servers": []any{
				map[string]any{"tag": "dns_local", "server": "{dns_local}"},
				// 与默认值相同的字面量不应被替换
				map[string]any{"tag": "extra", "server": "114.114.114.114"},
			},
			"rules": []any{map[string]any{"client_subnet": "{client_subnet}"}},
		},
		"route": map[string]any{
			"rule_set": []any{map[string]any{"url": "{mirror_url}/https://example.com/a.srs"}},
			"rules":    []any{map[string]any{"ip_cidr": []any{"223.5.5.5/24"}}},
		},
	}

	renderPlaceholders(cfg, config.TemplateOptions{DNSLocalServer: "223.6.6.6", ClientSubnet: "1.2.3.0/24"})

	servers := cfg["dns"].(map[string]any)["servers"].([]any)
	if got := servers[0].(map[string]any)["server"]; got != "223.6.6.6" {
		t.Errorf("dns_local = %v", got)
	}
	if got := servers[1].(map[string]any)["server"]; got != "114.114.114.114" {
		t.Errorf("literal DNS must not be replaced, got %v", got)
	}
	if got := cfg["dns"].(map[string]any)["rules"].([]any)[0].(map[string]any)["client_subnet"]; got != "1.2.3.0/24" {
		t.Errorf("client_subnet = %v", got)
	}
	route := cfg["route"].(map[string]any)
	if got := route["rules"].([]any)[0].(map[string]any)["ip_cidr"].([]any)[0]; got != "223.5.5.5/24" {
		t.Errorf("literal CIDR must not be replaced, got %v", got)
	}
	if got := route["rule_set"].([]any)[0].(map[string]any)["url"]; got != "https://example.com/a.srs" {
		t.Errorf("empty mirror should strip prefix, got %v", got)
	}
}

func TestRenderPlaceholdersDefaultsAndUnknown(t *testing.T) {
	cfg := map[string]any{
		"a": "{dns_local}",
		"b": "{client_subnet}",
		"c": "{external_controller}",
		"f": "{external_controller_host}",
		"d": []any{"{all}"},
		"e": "{typo_placeholder}",
	}
	var buf bytes.Buffer
	renderPlaceholders(cfg, config.TemplateOptions{
		MirrorURL: "https://m.example",
		Logger:    slog.New(slog.NewTextHandler(&buf, nil)),
	})

	want := map[string]string{"a": defaultDNSLocal, "b": defaultClientSubnet, "c": defaultExternalController, "f": "127.0.0.1"}
	for k, v := range want {
		if cfg[k] != v {
			t.Errorf("%s = %v, want %v", k, cfg[k], v)
		}
	}
	if cfg["d"].([]any)[0] != "{all}" {
		t.Error("{all} must be left for TemplateProcessor")
	}
	if !strings.Contains(buf.String(), "{typo_placeholder}") || strings.Contains(buf.String(), "{all}") {
		t.Errorf("expected warning only for unknown placeholder, got: %s", buf.String())
	}
}
