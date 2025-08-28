package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestSingBoxTemplate(t *testing.T) {
	tmpl := template.NewSingBoxTemplate()
	
	outbounds := []transformer.Outbound{
		{
			Type:       "vmess",
			Tag:        "test-vmess",
			Server:     "example.com",
			ServerPort: 443,
			UUID:       "12345678-abcd-1234-5678-123456789abc",
			TLS: map[string]any{
				"enabled":     true,
				"insecure":    true,
				"server_name": "example.com",
			},
		},
		{
			Type:       "trojan",
			Tag:        "test-trojan",
			Server:     "example.com",
			ServerPort: 443,
			Password:   "password123",
			TLS: map[string]any{
				"enabled":     true,
				"insecure":    false,
				"server_name": "example.com",
			},
		},
	}
	
	config := tmpl.Inject(outbounds, "https://mirror.example.com")
	
	if config == nil {
		t.Error("Config should not be nil")
	}
	
	if config.Log == nil {
		t.Error("Log config should not be nil")
	}
	
	if config.DNS == nil {
		t.Error("DNS config should not be nil")
	}
	
	if config.Experimental == nil {
		t.Error("Experimental config should not be nil")
	}
	
	if len(config.Inbounds) == 0 {
		t.Error("Inbounds should not be empty")
	}
	
	if len(config.Outbounds) == 0 {
		t.Error("Outbounds should not be empty")
	}
	
	proxyOutboundCount := 0
	for _, outbound := range config.Outbounds {
		if outbound["type"] == "vmess" || outbound["type"] == "trojan" {
			proxyOutboundCount++
		}
	}
	
	if proxyOutboundCount != 2 {
		t.Errorf("Expected 2 proxy outbounds, got %d", proxyOutboundCount)
	}
	
	if config.Route == nil {
		t.Error("Route config should not be nil")
	}
	
	if config.Route["rule_set"] == nil {
		t.Error("Rule sets should not be nil")
	}
}