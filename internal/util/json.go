package util

import (
	jsonv1 "encoding/json"
	json "github.com/goccy/go-json"
	"gopkg.in/yaml.v3"
	"log"
)

func YamlToJson(yamlData []byte) ([]byte, error) {

	var data map[string]interface{}

	// 使用yaml.Unmarshal将YAML数据解析到map中
	err := yaml.Unmarshal(yamlData, &data)
	if err != nil {
		log.Fatalf("yaml to json error: %v", err)
	}

	// 使用json.Marshal将Go对象编码为JSON格式
	jsonData, err := MarshalIndent(data) // 使用json.MarshalIndent可以美化输出
	if err != nil {
		log.Fatalf("yaml to json error: %v", err)
	}

	return jsonData, nil
}

// MarshalIndent marshals the value to JSON with indentation using high-performance JSON library
func MarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// Marshal marshals the value to JSON using high-performance JSON library
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal unmarshals JSON data using high-performance JSON library
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// ValidateJSON validates if the data is valid JSON
func ValidateJSON(data []byte) error {
	if json.Valid(data) {
		return nil
	}
	return jsonv1.Unmarshal(data, &json.RawMessage{})
}

// FormatJSON formats JSON with proper indentation
func FormatJSON(data []byte) ([]byte, error) {
	if err := ValidateJSON(data); err != nil {
		return nil, err
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return MarshalIndent(v)
}

// MarshalNoEscape marshals without HTML escaping for better performance
func MarshalNoEscape(v any) ([]byte, error) {
	return json.MarshalWithOption(v, json.DisableHTMLEscape())
}

// MarshalIndentNoEscape marshals with indentation without HTML escaping
func MarshalIndentNoEscape(v any) ([]byte, error) {
	return json.MarshalIndentWithOption(v, "", "  ", json.DisableHTMLEscape())
}

// UnmarshalFast unmarshals with optimizations for better performance
func UnmarshalFast(data []byte, v any) error {
	// goccy/go-json is already optimized, so just use the regular Unmarshal
	return json.Unmarshal(data, v)
}
