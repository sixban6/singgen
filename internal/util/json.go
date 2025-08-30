package util

import (
	jsonv1 "encoding/json"
	json "github.com/goccy/go-json"
)

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

// MarshalV1 provides fallback to standard library JSON for compatibility if needed
func MarshalV1(v any) ([]byte, error) {
	return jsonv1.Marshal(v)
}

// UnmarshalV1 provides fallback to standard library JSON for compatibility if needed  
func UnmarshalV1(data []byte, v any) error {
	return jsonv1.Unmarshal(data, v)
}