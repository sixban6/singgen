package test

import (
	"testing"
	
	"github.com/sixban6/singgen/internal/util"
)

// TestData represents a complex data structure for testing
type TestData struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Count       int               `json:"count"`
	Enabled     bool              `json:"enabled"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Nested      []NestedData      `json:"nested"`
}

type NestedData struct {
	ID       string   `json:"id"`
	Values   []int    `json:"values"`
	Options  []string `json:"options"`
	Settings map[string]interface{} `json:"settings"`
}

// generateTestData creates a large test dataset
func generateTestData() TestData {
	nested := make([]NestedData, 100)
	for i := 0; i < 100; i++ {
		nested[i] = NestedData{
			ID:      "nested-" + string(rune(i)),
			Values:  []int{i, i * 2, i * 3, i * 4, i * 5},
			Options: []string{"opt1", "opt2", "opt3", "opt4", "opt5"},
			Settings: map[string]interface{}{
				"timeout":     30,
				"retry_count": 3,
				"enabled":     true,
				"endpoint":    "https://example.com/api",
			},
		}
	}

	metadata := make(map[string]string)
	for i := 0; i < 50; i++ {
		key := "key" + string(rune(i))
		value := "value" + string(rune(i))
		metadata[key] = value
	}

	return TestData{
		Name:        "Performance Test Dataset",
		Description: "This is a large dataset designed to test JSON marshaling and unmarshaling performance with complex nested structures and arrays",
		Count:       1000,
		Enabled:     true,
		Tags:        []string{"performance", "test", "json", "optimization", "benchmark"},
		Metadata:    metadata,
		Nested:      nested,
	}
}

// BenchmarkJSONMarshal benchmarks the optimized JSON marshaling
func BenchmarkJSONMarshal(b *testing.B) {
	testData := generateTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := util.Marshal(testData)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

// BenchmarkJSONUnmarshal benchmarks the optimized JSON unmarshaling
func BenchmarkJSONUnmarshal(b *testing.B) {
	testData := generateTestData()
	jsonData, err := util.Marshal(testData)
	if err != nil {
		b.Fatalf("Marshal failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestData
		err := util.Unmarshal(jsonData, &result)
		if err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}

// BenchmarkJSONMarshalIndent benchmarks the optimized JSON marshaling with indentation
func BenchmarkJSONMarshalIndent(b *testing.B) {
	testData := generateTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := util.MarshalIndent(testData)
		if err != nil {
			b.Fatalf("MarshalIndent failed: %v", err)
		}
	}
}

// BenchmarkJSONMarshalV1 benchmarks the standard library JSON marshaling for comparison
func BenchmarkJSONMarshalV1(b *testing.B) {
	testData := generateTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := util.MarshalV1(testData)
		if err != nil {
			b.Fatalf("MarshalV1 failed: %v", err)
		}
	}
}

// BenchmarkJSONUnmarshalV1 benchmarks the standard library JSON unmarshaling for comparison
func BenchmarkJSONUnmarshalV1(b *testing.B) {
	testData := generateTestData()
	jsonData, err := util.MarshalV1(testData)
	if err != nil {
		b.Fatalf("MarshalV1 failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result TestData
		err := util.UnmarshalV1(jsonData, &result)
		if err != nil {
			b.Fatalf("UnmarshalV1 failed: %v", err)
		}
	}
}

// BenchmarkJSONMarshalNoEscape benchmarks marshaling without HTML escaping
func BenchmarkJSONMarshalNoEscape(b *testing.B) {
	testData := generateTestData()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := util.MarshalNoEscape(testData)
		if err != nil {
			b.Fatalf("MarshalNoEscape failed: %v", err)
		}
	}
}

// TestJSONOptimizationCorrectness verifies that optimized JSON produces the same results
func TestJSONOptimizationCorrectness(t *testing.T) {
	testData := generateTestData()
	
	// Test Marshal
	optimizedData, err := util.Marshal(testData)
	if err != nil {
		t.Fatalf("Optimized Marshal failed: %v", err)
	}
	
	standardData, err := util.MarshalV1(testData)
	if err != nil {
		t.Fatalf("Standard Marshal failed: %v", err)
	}
	
	// Both should produce valid JSON that unmarshals to the same result
	var optimizedResult TestData
	var standardResult TestData
	
	if err := util.Unmarshal(optimizedData, &optimizedResult); err != nil {
		t.Fatalf("Optimized Unmarshal failed: %v", err)
	}
	
	if err := util.UnmarshalV1(standardData, &standardResult); err != nil {
		t.Fatalf("Standard Unmarshal failed: %v", err)
	}
	
	// Basic verification - both should have same name and count
	if optimizedResult.Name != standardResult.Name {
		t.Errorf("Name mismatch: optimized=%s, standard=%s", optimizedResult.Name, standardResult.Name)
	}
	
	if optimizedResult.Count != standardResult.Count {
		t.Errorf("Count mismatch: optimized=%d, standard=%d", optimizedResult.Count, standardResult.Count)
	}
	
	if len(optimizedResult.Nested) != len(standardResult.Nested) {
		t.Errorf("Nested length mismatch: optimized=%d, standard=%d", len(optimizedResult.Nested), len(standardResult.Nested))
	}
}