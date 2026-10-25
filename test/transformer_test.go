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
		{
			ID:       "7",
			Tag:      "tuic-test",
			Type:     "tuic",
			Addr:     "example.com",
			Port:     8443,
			UUID:     "2dd61d93-75d8-4da4-ac0e-6aece7eac365",
			Password: "password-tuic",
			Security: model.Security{
				TLS:        true,
				SkipVerify: true,
				ServerName: "real.example.com",
				ALPN:       []string{"h3"},
			},
			Extra: map[string]any{
				"congestion_control": "bbr",
				"udp_relay_mode":     "native",
				"zero_rtt_handshake": true,
				"heartbeat":          "10s",
				"disable_sni":        true,
			},
		},
	}

	outbounds, err := transformer.Transform(nodes)
	if err != nil {
		t.Errorf("Transform failed: %v", err)
	}

	if len(outbounds) != 7 {
		t.Errorf("Expected 7 outbounds, got %d", len(outbounds))
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

	tuicOut := outbounds[6]
	if tuicOut.Type != "tuic" {
		t.Errorf("Expected tuic type, got %s", tuicOut.Type)
	}
	if tuicOut.UUID != "2dd61d93-75d8-4da4-ac0e-6aece7eac365" {
		t.Errorf("Expected UUID 2dd61d93-75d8-4da4-ac0e-6aece7eac365, got %s", tuicOut.UUID)
	}
	if tuicOut.Password != "password-tuic" {
		t.Errorf("Expected password-tuic, got %s", tuicOut.Password)
	}
	if tuicOut.TLS == nil {
		t.Error("Expected TLS config for TUIC")
	}
	if tuicOut.TLS["disable_sni"] != true {
		t.Errorf("Expected disable_sni true, got %v", tuicOut.TLS["disable_sni"])
	}
	if tuicOut.CongestionControl != "bbr" {
		t.Errorf("Expected congestion control bbr, got %s", tuicOut.CongestionControl)
	}
	if tuicOut.UDPRelayMode != "native" {
		t.Errorf("Expected udp relay mode native, got %s", tuicOut.UDPRelayMode)
	}
	if !tuicOut.ZeroRttHandshake {
		t.Error("Expected zero rtt handshake to be true")
	}
	if tuicOut.Heartbeat != "10s" {
		t.Errorf("Expected heartbeat 10s, got %s", tuicOut.Heartbeat)
	}
}
