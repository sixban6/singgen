package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/pkg/model"
)

// BenchmarkTransformer 测试Transformer的性能
func BenchmarkTransformer(b *testing.B) {
	// 创建测试数据
	nodes := createTestNodes(100)
	transformer := transformer.NewSingBoxTransformer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(nodes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTransformerLarge 测试大量节点的性能
func BenchmarkTransformerLarge(b *testing.B) {
	nodes := createTestNodes(1000)
	transformer := transformer.NewSingBoxTransformer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(nodes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestTransformerConcurrency 测试并发安全性
func TestTransformerConcurrency(t *testing.T) {
	nodes := createTestNodes(50)
	transformer := transformer.NewSingBoxTransformer()

	start := time.Now()
	result, err := transformer.Transform(nodes)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("No results returned")
	}

	t.Logf("Transformed %d nodes in %v (concurrent)", len(nodes), duration)

	// 验证结果的完整性
	tagSet := make(map[string]bool)
	for _, outbound := range result {
		if tagSet[outbound.Tag] {
			t.Errorf("Duplicate tag found: %s", outbound.Tag)
		}
		tagSet[outbound.Tag] = true
	}
}

// createTestNodes 创建测试节点
func createTestNodes(count int) []model.Node {
	nodes := make([]model.Node, count)
	
	protocols := []string{"vmess", "vless", "trojan", "shadowsocks", "hysteria2"}
	
	for i := 0; i < count; i++ {
		protocol := protocols[i%len(protocols)]
		
		node := model.Node{
			ID:   string(rune('A' + i)),
			Tag:  protocol + "_node_" + fmt.Sprintf("%03d", i), // 确保tag唯一
			Type: protocol,
			Addr: "example.com",
			Port: uint16(8000 + i),
			Security: model.Security{
				TLS:        true,
				ServerName: "example.com",
				ALPN:       []string{"h2", "http/1.1"},
			},
			Transport: model.Transport{
				Net:  "ws",
				Path: "/path" + string(rune('0' + (i%10))),
				Host: "example.com",
			},
			Extra: make(map[string]any),
		}
		
		switch protocol {
		case "vmess", "vless":
			node.UUID = "12345678-abcd-1234-5678-" + string(rune('0' + (i%10))) + "23456789abc"
		case "trojan", "hysteria2":
			node.Password = "password" + string(rune('0' + (i%10)))
		case "shadowsocks":
			node.Password = "password" + string(rune('0' + (i%10)))
			node.Extra["method"] = "aes-256-gcm"
		}
		
		nodes[i] = node
	}
	
	return nodes
}