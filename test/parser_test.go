package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/parser"
)

func TestVmessParser(t *testing.T) {
	vmessURL := `vmess://eyJ2IjoiMiIsInBzIjoidGVzdCBub2RlIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0Ijo0NDMsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LTU2NzgtMTIzNDU2Nzg5YWJjIiwiYWlkIjowLCJuZXQiOiJ3cyIsImhvc3QiOiJleGFtcGxlLmNvbSIsInBhdGgiOiIvd3MiLCJ0bHMiOiJ0bHMifQ==`

	parser := &parser.VmessParser{}

	if !parser.Accept("", []byte(vmessURL)) {
		t.Error("VMess parser should accept vmess URL")
	}

	nodes, err := parser.Parse([]byte(vmessURL))
	if err != nil {
		t.Errorf("VMess parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "vmess" {
		t.Errorf("Expected type vmess, got %s", node.Type)
	}
	if node.Addr != "example.com" {
		t.Errorf("Expected addr example.com, got %s", node.Addr)
	}
	if node.Port != 443 {
		t.Errorf("Expected port 443, got %d", node.Port)
	}
	if !node.Security.TLS {
		t.Error("Expected TLS to be true")
	}
}

func TestVlessParser(t *testing.T) {
	vlessURL := `vless://12345678-abcd-1234-5678-123456789abc@example.com:443?type=ws&host=example.com&path=/ws&security=tls#test%20node`

	parser := &parser.VlessParser{}

	if !parser.Accept("", []byte(vlessURL)) {
		t.Error("VLESS parser should accept vless URL")
	}

	nodes, err := parser.Parse([]byte(vlessURL))
	if err != nil {
		t.Errorf("VLESS parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "vless" {
		t.Errorf("Expected type vless, got %s", node.Type)
	}
	if node.Addr != "example.com" {
		t.Errorf("Expected addr example.com, got %s", node.Addr)
	}
	if node.Port != 443 {
		t.Errorf("Expected port 443, got %d", node.Port)
	}
	if node.UUID != "12345678-abcd-1234-5678-123456789abc" {
		t.Errorf("Expected UUID 12345678-abcd-1234-5678-123456789abc, got %s", node.UUID)
	}
}

func TestTrojanParser(t *testing.T) {
	trojanURL := `trojan://password123@example.com:443?type=ws&host=example.com&path=/ws#test%20node`

	parser := &parser.TrojanParser{}

	if !parser.Accept("", []byte(trojanURL)) {
		t.Error("Trojan parser should accept trojan URL")
	}

	nodes, err := parser.Parse([]byte(trojanURL))
	if err != nil {
		t.Errorf("Trojan parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "trojan" {
		t.Errorf("Expected type trojan, got %s", node.Type)
	}
	if node.Password != "password123" {
		t.Errorf("Expected password password123, got %s", node.Password)
	}
	if !node.Security.TLS {
		t.Error("Expected TLS to be true for Trojan")
	}
}

func TestHysteria2Parser(t *testing.T) {
	hysteria2URL := `hysteria2://password123@example.com:443#test%20node`

	parser := &parser.Hysteria2Parser{}

	if !parser.Accept("", []byte(hysteria2URL)) {
		t.Error("Hysteria2 parser should accept hysteria2 URL")
	}

	nodes, err := parser.Parse([]byte(hysteria2URL))
	if err != nil {
		t.Errorf("Hysteria2 parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "hysteria2" {
		t.Errorf("Expected type hysteria2, got %s", node.Type)
	}
	if node.Password != "password123" {
		t.Errorf("Expected password password123, got %s", node.Password)
	}
}

func TestAnyTLSParser(t *testing.T) {
	anyTLSURL := `anytls://password123@example.com:8443/?sni=real.example.com&insecure=1&alpn=h2,http/1.1&idle_session_check_interval=20s&idle_session_timeout=40s&min_idle_session=2#test%20node`

	parser := &parser.AnyTLSParser{}

	if !parser.Accept("", []byte(anyTLSURL)) {
		t.Error("AnyTLS parser should accept anytls URL")
	}

	nodes, err := parser.Parse([]byte(anyTLSURL))
	if err != nil {
		t.Errorf("AnyTLS parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "anytls" {
		t.Errorf("Expected type anytls, got %s", node.Type)
	}
	if node.Password != "password123" {
		t.Errorf("Expected password password123, got %s", node.Password)
	}
	if node.Port != 8443 {
		t.Errorf("Expected port 8443, got %d", node.Port)
	}
	if !node.Security.TLS {
		t.Error("Expected TLS to be true for AnyTLS")
	}
	if !node.Security.SkipVerify {
		t.Error("Expected SkipVerify to be true")
	}
	if node.Security.ServerName != "real.example.com" {
		t.Errorf("Expected SNI real.example.com, got %s", node.Security.ServerName)
	}
	if minIdleSession := node.Extra["min_idle_session"]; minIdleSession != "2" {
		t.Errorf("Expected min_idle_session 2, got %v", minIdleSession)
	}
}

func TestAnyTLSParserDefaultPort(t *testing.T) {
	anyTLSURL := `anytls://password123@example.com/?peer=real.example.com#test%20node`

	parser := &parser.AnyTLSParser{}
	nodes, err := parser.Parse([]byte(anyTLSURL))
	if err != nil {
		t.Errorf("AnyTLS parser failed: %v", err)
	}

	if nodes[0].Port != 443 {
		t.Errorf("Expected default port 443, got %d", nodes[0].Port)
	}
	if nodes[0].Security.ServerName != "real.example.com" {
		t.Errorf("Expected peer to map to server name, got %s", nodes[0].Security.ServerName)
	}
}

func TestShadowsocksParser(t *testing.T) {
	ssURL := `ss://YWVzLTI1Ni1nY206cGFzc3dvcmQxMjM@example.com:8388#test%20node`

	parser := &parser.ShadowsocksParser{}

	if !parser.Accept("", []byte(ssURL)) {
		t.Error("Shadowsocks parser should accept ss URL")
	}

	nodes, err := parser.Parse([]byte(ssURL))
	if err != nil {
		t.Errorf("Shadowsocks parser failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	node := nodes[0]
	if node.Type != "shadowsocks" {
		t.Errorf("Expected type shadowsocks, got %s", node.Type)
	}
	if node.Password != "password123" {
		t.Errorf("Expected password password123, got %s", node.Password)
	}
	if method, ok := node.Extra["method"].(string); !ok || method != "aes-256-gcm" {
		t.Errorf("Expected method aes-256-gcm, got %v", node.Extra["method"])
	}
}

func TestMixedParser(t *testing.T) {
	mixedURLs := `vmess://eyJ2IjoiMiIsInBzIjoidm1lc3MgdGVzdCIsImFkZCI6ImV4YW1wbGUuY29tIiwicG9ydCI6NDQzLCJpZCI6IjEyMzQ1Njc4LWFiY2QtMTIzNC01Njc4LTEyMzQ1Njc4OWFiYyIsImFpZCI6MCwibmV0Ijoid3MiLCJob3N0IjoiZXhhbXBsZS5jb20iLCJwYXRoIjoiL3dzIiwidGxzIjoidGxzIn0=
vless://12345678-abcd-1234-5678-123456789abc@example.com:443?type=ws&host=example.com&path=/ws&security=tls#vless%20test
trojan://password123@example.com:443?type=ws&host=example.com&path=/ws#trojan%20test
anytls://password123@example.com:443/?sni=example.com#anytls%20test`

	parser := &parser.MixedParser{}

	if !parser.Accept("", []byte(mixedURLs)) {
		t.Error("Mixed parser should accept mixed URLs")
	}

	nodes, err := parser.Parse([]byte(mixedURLs))
	if err != nil {
		t.Errorf("Mixed parser failed: %v", err)
	}

	if len(nodes) != 4 {
		t.Errorf("Expected 4 nodes, got %d", len(nodes))
	}

	types := []string{"vmess", "vless", "trojan", "anytls"}
	for i, expectedType := range types {
		if nodes[i].Type != expectedType {
			t.Errorf("Expected node %d type %s, got %s", i, expectedType, nodes[i].Type)
		}
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"vmess://eyJ2IjoiMiJ9", "vmess"},
		{"vless://uuid@host:port", "vless"},
		{"trojan://password@host:port", "trojan"},
		{"hysteria2://password@host:port", "hysteria2"},
		{"hy2://password@host:port", "hysteria2"},
		{"anytls://password@host:port", "anytls"},
		{"ss://method:password@host:port", "shadowsocks"},
		{"vmess://xxx\nvless://yyy", "mixed"},
		{"invalid data", "unknown"},
	}

	for _, test := range tests {
		result := parser.DetectFormat([]byte(test.input))
		if result != test.expected {
			t.Errorf("DetectFormat(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}
