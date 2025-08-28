package test

import (
	"testing"

	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

func TestTemplateFactory(t *testing.T) {
	factory := template.NewTemplateFactory()
	
	// 测试获取可用版本
	versions, err := factory.GetAvailableVersions()
	if err != nil {
		t.Errorf("GetAvailableVersions failed: %v", err)
	}
	
	if len(versions) == 0 {
		t.Error("Expected at least one template version")
	}
	
	// 验证包含预期版本
	hasV112 := false
	hasV113 := false
	for _, version := range versions {
		if version == "v1.12" {
			hasV112 = true
		}
		if version == "v1.13" {
			hasV113 = true
		}
	}
	
	if !hasV112 {
		t.Error("Expected to find v1.12 template")
	}
	if !hasV113 {
		t.Error("Expected to find v1.13 template")
	}
}

func TestCreateTemplate(t *testing.T) {
	factory := template.NewTemplateFactory()
	
	// 测试创建 v1.12 模板
	tmpl112, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Errorf("Failed to create v1.12 template: %v", err)
	}
	
	if tmpl112 == nil {
		t.Error("Expected non-nil template")
	}
	
	// 测试创建 v1.13 模板
	tmpl113, err := factory.CreateTemplate("v1.13")
	if err != nil {
		t.Errorf("Failed to create v1.13 template: %v", err)
	}
	
	if tmpl113 == nil {
		t.Error("Expected non-nil template")
	}
	
	// 测试默认版本
	tmplDefault, err := factory.CreateTemplate("")
	if err != nil {
		t.Errorf("Failed to create default template: %v", err)
	}
	
	if tmplDefault == nil {
		t.Error("Expected non-nil default template")
	}
}

func TestFileTemplate(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}
	
	// 测试生成配置
	outbounds := []transformer.Outbound{
		{
			Type:       "vmess",
			Tag:        "test-node",
			Server:     "example.com",
			ServerPort: 443,
			UUID:       "12345678-abcd-1234-5678-123456789abc",
		},
	}
	
	config := tmpl.Inject(outbounds, "")
	
	if config == nil {
		t.Error("Expected non-nil config")
	}
	
	if config.Log == nil {
		t.Error("Expected log config")
	}
	
	if config.DNS == nil {
		t.Error("Expected DNS config")
	}
	
	if len(config.Outbounds) == 0 {
		t.Error("Expected outbounds")
	}
	
	// 验证代理节点被正确注入
	hasProxyNode := false
	for _, outbound := range config.Outbounds {
		if outbound["tag"] == "test-node" {
			hasProxyNode = true
			if outbound["type"] != "vmess" {
				t.Errorf("Expected vmess type, got %v", outbound["type"])
			}
			break
		}
	}
	
	if !hasProxyNode {
		t.Error("Expected to find injected proxy node")
	}
}

func TestMirrorURLReplacement(t *testing.T) {
	factory := template.NewTemplateFactory()
	tmpl, err := factory.CreateTemplate("v1.12")
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}
	
	mirrorURL := "https://mirror.example.com"
	config := tmpl.Inject([]transformer.Outbound{}, mirrorURL)
	
	// 检查 experimental.clash_api.external_ui_download_url 是否正确替换
	experimental := config.Experimental
	if clashAPI, ok := experimental["clash_api"].(map[string]any); ok {
		if downloadURL, ok := clashAPI["external_ui_download_url"].(string); ok {
			expected := mirrorURL + "/https://github.com/MetaCubeX/Yacd-meta/archive/gh-pages.zip"
			if downloadURL != expected {
				t.Errorf("Mirror URL replacement failed. Expected %s, got %s", expected, downloadURL)
			}
		} else {
			t.Error("external_ui_download_url not found or wrong type")
		}
	} else {
		t.Error("clash_api not found or wrong type")
	}
}