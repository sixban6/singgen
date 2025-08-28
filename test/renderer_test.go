package test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/renderer"
	"gopkg.in/yaml.v3"
)

func TestJSONRenderer(t *testing.T) {
	r := renderer.NewJSONRenderer()
	
	data := map[string]any{
		"name":    "test",
		"value":   123,
		"enabled": true,
		"tags":    []string{"tag1", "tag2"},
	}
	
	result, err := r.Render(data)
	if err != nil {
		t.Errorf("JSON render failed: %v", err)
	}
	
	if len(result) == 0 {
		t.Error("JSON result should not be empty")
	}
	
	var parsed map[string]any
	err = json.Unmarshal(result, &parsed)
	if err != nil {
		t.Errorf("JSON result is not valid JSON: %v", err)
	}
	
	if parsed["name"] != "test" {
		t.Errorf("Expected name=test, got %v", parsed["name"])
	}
}

func TestYAMLRenderer(t *testing.T) {
	r := renderer.NewYAMLRenderer()
	
	data := map[string]any{
		"name":    "test",
		"value":   123,
		"enabled": true,
		"tags":    []string{"tag1", "tag2"},
	}
	
	result, err := r.Render(data)
	if err != nil {
		t.Errorf("YAML render failed: %v", err)
	}
	
	if len(result) == 0 {
		t.Error("YAML result should not be empty")
	}
	
	var parsed map[string]any
	err = yaml.Unmarshal(result, &parsed)
	if err != nil {
		t.Errorf("YAML result is not valid YAML: %v", err)
	}
	
	if parsed["name"] != "test" {
		t.Errorf("Expected name=test, got %v", parsed["name"])
	}
	
	if !strings.Contains(string(result), "name: test") {
		t.Error("YAML should contain 'name: test'")
	}
}