package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/pkg/model"
)

func TestSingBoxTransformer(t *testing.T) {
	transformer := transformer.NewSingBoxTransformer()

	nodes := []model.Node{
		{
			ID:   "1",
			Tag:  "vmess-test",
			Type: "vmess",
			Addr: "example.com",
			Port: 443,
			UUID: "12345678-abcd-1234-5678-123456789abc",
			Security: model.Security{
				TLS:        true,
				SkipVerify: true,
				ServerName: "example.com",
				ALPN:       []string{"h2", "http/1.1"},
			},
			Transport: model.Transport{
				Net:  "ws",
				Host: "example.com",
				Path: "/ws",
			},
			Extra: map[string]any{
				"alter_id": 0,
			},
		},
		{
			ID:   "2",
			Tag:  "vless-test",
			Type: "vless",
			Addr: "example.com",
			Port: 443,
			UUID: "87654321-dcba-4321-8765-cba987654321",
			Security: model.Security{
				TLS:        true,
				SkipVerify: false,
				ServerName: "example.com",
			},
		},
		{
			ID:       "3",
			Tag:      "trojan-test",
			Type:     "trojan",
			Addr:     "example.com",
			Port:     443,
			Password: "password123",
			Security: model.Security{
				TLS:        true,
				SkipVerify: false,
				ServerName: "example.com",
			},
		},
		{
			ID:       "4",
			Tag:      "hy2-test",
			Type:     "hysteria2",
			Addr:     "example.com",
			Port:     443,
			Password: "password456",
			Security: model.Security{
				TLS:        true,
				SkipVerify: false,
				ServerName: "example.com",
			},
		},
		{
			ID:       "5",
			Tag:      "ss-test",
			Type:     "shadowsocks",
			Addr:     "example.com",
			Port:     8388,
			Password: "password789",
			Extra: map[string]any{
				"method": "aes-256-gcm",
			},
		},
		{
			ID:       "6",
			Tag:      "anytls-test",
			Type:     "anytls",
			Addr:     "example.com",
			Port:     8443,
			Password: "password-anytls",
			Security: model.Security{
				TLS:        true,
				SkipVerify: true,
				ServerName: "real.example.com",
				ALPN:       []string{"h2", "http/1.1"},
			},
			Extra: map[string]any{
				"idle_session_check_interval": "20s",
				"idle_session_timeout":        "40s",
				"min_idle_session":            "2",
			},
		},
	}

	outbounds, err := transformer.Transform(nodes)
	if err != nil {
		t.Errorf("Transform failed: %v", err)
	}

	if len(outbounds) != 6 {
		t.Errorf("Expected 6 outbounds, got %d", len(outbounds))
	}

	vmessOut := outbounds[0]
	if vmessOut.Type != "vmess" {
		t.Errorf("Expected vmess type, got %s", vmessOut.Type)
	}
	if vmessOut.UUID != "12345678-abcd-1234-5678-123456789abc" {
		t.Errorf("Expected UUID match, got %s", vmessOut.UUID)
	}
	if vmessOut.TLS == nil {
		t.Error("Expected TLS config for VMess")
	}

	vlessOut := outbounds[1]
	if vlessOut.Type != "vless" {
		t.Errorf("Expected vless type, got %s", vlessOut.Type)
	}

	trojanOut := outbounds[2]
	if trojanOut.Type != "trojan" {
		t.Errorf("Expected trojan type, got %s", trojanOut.Type)
	}
	if trojanOut.Password != "password123" {
		t.Errorf("Expected password123, got %s", trojanOut.Password)
	}

	hy2Out := outbounds[3]
	if hy2Out.Type != "hysteria2" {
		t.Errorf("Expected hysteria2 type, got %s", hy2Out.Type)
	}

	ssOut := outbounds[4]
	if ssOut.Type != "shadowsocks" {
		t.Errorf("Expected shadowsocks type, got %s", ssOut.Type)
	}
	if ssOut.Method != "aes-256-gcm" {
		t.Errorf("Expected aes-256-gcm method, got %s", ssOut.Method)
	}

	anyTLSOut := outbounds[5]
	if anyTLSOut.Type != "anytls" {
		t.Errorf("Expected anytls type, got %s", anyTLSOut.Type)
	}
	if anyTLSOut.Password != "password-anytls" {
		t.Errorf("Expected password-anytls, got %s", anyTLSOut.Password)
	}
	if anyTLSOut.TLS == nil {
		t.Error("Expected TLS config for AnyTLS")
	}
	if anyTLSOut.IdleSessionCheckInterval != "20s" {
		t.Errorf("Expected idle session check interval 20s, got %s", anyTLSOut.IdleSessionCheckInterval)
	}
	if anyTLSOut.IdleSessionTimeout != "40s" {
		t.Errorf("Expected idle session timeout 40s, got %s", anyTLSOut.IdleSessionTimeout)
	}
	if anyTLSOut.MinIdleSession != 2 {
		t.Errorf("Expected min idle session 2, got %d", anyTLSOut.MinIdleSession)
	}
}
