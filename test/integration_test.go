package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/registry"
	"github.com/sixban6/singgen/internal/util"
)

func TestIntegrationWorkflow(t *testing.T) {
	if err := util.InitLogger("error"); err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}
	defer util.Sync()

	testData := `vmess://eyJ2IjoiMiIsInBzIjoidm1lc3MgdGVzdCIsImFkZCI6ImV4YW1wbGUuY29tIiwicG9ydCI6NDQzLCJpZCI6IjEyMzQ1Njc4LWFiY2QtMTIzNC01Njc4LTEyMzQ1Njc4OWFiYyIsImFpZCI6MCwibmV0Ijoid3MiLCJob3N0IjoiZXhhbXBsZS5jb20iLCJwYXRoIjoiL3dzIiwidGxzIjoidGxzIn0=
vless://12345678-abcd-1234-5678-123456789abc@example.com:443?type=ws&host=example.com&path=/ws&security=tls#vless%20test
trojan://password123@example.com:443?type=ws&host=example.com&path=/ws#trojan%20test`

	format := parser.DetectFormat([]byte(testData))
	if format != "mixed" {
		t.Errorf("Expected format mixed, got %s", format)
	}

	var nodes []string
	if factory, exists := parser.Registry[format]; exists {
		p := factory()
		parsedNodes, err := p.Parse([]byte(testData))
		if err != nil {
			t.Errorf("Parse failed: %v", err)
		}
		
		if len(parsedNodes) != 3 {
			t.Errorf("Expected 3 nodes, got %d", len(parsedNodes))
		}
		
		for _, node := range parsedNodes {
			nodes = append(nodes, node.Tag)
		}

		outbounds, err := registry.Transformer.Transform(parsedNodes)
		if err != nil {
			t.Errorf("Transform failed: %v", err)
		}
		
		if len(outbounds) != 3 {
			t.Errorf("Expected 3 outbounds, got %d", len(outbounds))
		}

		config := registry.Template.Inject(outbounds, "")

		jsonRenderer := registry.GetRenderer("json")
		jsonData, err := jsonRenderer.Render(config)
		if err != nil {
			t.Errorf("JSON render failed: %v", err)
		}
		
		var jsonConfig map[string]any
		if err := json.Unmarshal(jsonData, &jsonConfig); err != nil {
			t.Errorf("Invalid JSON output: %v", err)
		}
		
		if jsonConfig["log"] == nil {
			t.Error("JSON config should have log section")
		}
		
		yamlRenderer := registry.GetRenderer("yaml")
		yamlData, err := yamlRenderer.Render(config)
		if err != nil {
			t.Errorf("YAML render failed: %v", err)
		}
		
		if !strings.Contains(string(yamlData), "log:") {
			t.Error("YAML config should contain log section")
		}
	}
}

func TestFileOperations(t *testing.T) {
	if err := util.InitLogger("error"); err != nil {
		t.Fatalf("Failed to init logger: %v", err)
	}
	defer util.Sync()

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_subscription.txt")
	
	testContent := `vmess://eyJ2IjoiMiIsInBzIjoidGVzdCBub2RlIiwiYWRkIjoiZXhhbXBsZS5jb20iLCJwb3J0Ijo0NDMsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LTU2NzgtMTIzNDU2Nzg5YWJjIiwiYWlkIjowLCJuZXQiOiJ3cyIsImhvc3QiOiJleGFtcGxlLmNvbSIsInBhdGgiOiIvd3MiLCJ0bHMiOiJ0bHMifQ==`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	fileFetcher := registry.GetFetcher("file")
	data, err := fileFetcher.Fetch(testFile)
	if err != nil {
		t.Errorf("File fetch failed: %v", err)
	}
	
	if string(data) != testContent {
		t.Error("Fetched data doesn't match original content")
	}
	
	p := &parser.VmessParser{}
	nodes, err := p.Parse(data)
	if err != nil {
		t.Errorf("Parse failed: %v", err)
	}
	
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
	
	outbounds, err := registry.Transformer.Transform(nodes)
	if err != nil {
		t.Errorf("Transform failed: %v", err)
	}
	
	config := registry.Template.Inject(outbounds, "")
	
	renderer := registry.GetRenderer("json")
	configData, err := renderer.Render(config)
	if err != nil {
		t.Errorf("Render failed: %v", err)
	}
	
	outputFile := filepath.Join(tempDir, "config.json")
	if err := util.WriteFile(outputFile, configData); err != nil {
		t.Errorf("Write output file failed: %v", err)
	}
	
	if !util.FileExists(outputFile) {
		t.Error("Output file should exist")
	}
	
	readData, err := util.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Read output file failed: %v", err)
	}
	
	var outputConfig map[string]any
	if err := json.Unmarshal(readData, &outputConfig); err != nil {
		t.Errorf("Output file contains invalid JSON: %v", err)
	}
}