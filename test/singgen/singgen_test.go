package singgen_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sixban6/singgen/pkg/singgen"
)

// Test data
const (
	testVmessNode = `vmess://eyJhZGQiOiJ0ZXN0LmV4YW1wbGUuY29tIiwiYWlkIjowLCJhbHBuIjoiIiwiaG9zdCI6IiIsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LWFiY2QtMTIzNDU2Nzg5YWJjIiwibmV0IjoidGNwIiwicGF0aCI6IiIsInBvcnQiOjEwODA5LCJwcyI6IlRlc3QgTm9kZSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
	
	testSubscription = `vmess://eyJhZGQiOiJ0ZXN0MS5leGFtcGxlLmNvbSIsImFpZCI6MCwiYWxwbiI6IiIsImhvc3QiOiIiLCJpZCI6IjEyMzQ1Njc4LWFiY2QtMTIzNC1hYmNkLTEyMzQ1Njc4OWFiYyIsIm5ldCI6InRjcCIsInBhdGgiOiIiLCJwb3J0IjoxMDgwOSwicHMiOiJUZXN0IE5vZGUgMSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=
vmess://eyJhZGQiOiJ0ZXN0Mi5leGFtcGxlLmNvbSIsImFpZCI6MCwiYWxwbiI6IiIsImhvc3QiOiIiLCJpZCI6Ijg3NjU0MzIxLWRjYmEtNDMyMS1kY2JhLTg3NjU0MzIxOWRjYiIsIm5ldCI6InRjcCIsInBhdGgiOiIiLCJwb3J0IjoyMDgwOSwicHMiOiJUZXN0IE5vZGUgMiIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
)

func TestGenerateConfig(t *testing.T) {
	// Create test subscription file
	tmpFile := createTestFile(t, testSubscription)
	defer os.Remove(tmpFile)
	
	tests := []struct {
		name    string
		source  string
		opts    []singgen.Option
		wantErr bool
	}{
		{
			name:    "basic generation",
			source:  tmpFile,
			opts:    []singgen.Option{},
			wantErr: false,
		},
		{
			name:   "with custom template",
			source: tmpFile,
			opts: []singgen.Option{
				singgen.WithTemplate("v1.12"),
				singgen.WithPlatform("linux"),
			},
			wantErr: false,
		},
		{
			name:   "with logger",
			source: tmpFile,
			opts: []singgen.Option{
				singgen.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))),
			},
			wantErr: false,
		},
		{
			name:    "empty source",
			source:  "",
			opts:    []singgen.Option{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			cfg, err := singgen.GenerateConfig(ctx, tt.source, tt.opts...)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateConfig() expected error, got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("GenerateConfig() error = %v", err)
				return
			}
			
			if cfg == nil {
				t.Errorf("GenerateConfig() returned nil config")
				return
			}
			
			// Verify config structure
			if len(cfg.Outbounds) == 0 {
				t.Errorf("GenerateConfig() returned config with no outbounds")
			}
		})
	}
}

func TestGenerateConfigBytes(t *testing.T) {
	tmpFile := createTestFile(t, testVmessNode)
	defer os.Remove(tmpFile)
	
	tests := []struct {
		name   string
		format string
		opts   []singgen.Option
	}{
		{
			name:   "json format",
			format: "json",
			opts:   []singgen.Option{singgen.WithOutputFormat("json")},
		},
		{
			name:   "yaml format", 
			format: "yaml",
			opts:   []singgen.Option{singgen.WithOutputFormat("yaml")},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			data, err := singgen.GenerateConfigBytes(ctx, tmpFile, tt.opts...)
			if err != nil {
				t.Errorf("GenerateConfigBytes() error = %v", err)
				return
			}
			
			if len(data) == 0 {
				t.Errorf("GenerateConfigBytes() returned empty data")
			}
			
			// Basic format validation
			dataStr := string(data)
			if tt.format == "json" {
				if !strings.Contains(dataStr, "{") {
					t.Errorf("GenerateConfigBytes() JSON output doesn't contain '{'")
				}
			} else if tt.format == "yaml" {
				if !strings.Contains(dataStr, ":") {
					t.Errorf("GenerateConfigBytes() YAML output doesn't contain ':'")
				}
			}
		})
	}
}

func TestParseNodes(t *testing.T) {
	tmpFile := createTestFile(t, testSubscription)
	defer os.Remove(tmpFile)
	
	ctx := context.Background()
	
	nodes, err := singgen.ParseNodes(ctx, tmpFile)
	if err != nil {
		t.Errorf("ParseNodes() error = %v", err)
		return
	}
	
	if len(nodes) != 2 {
		t.Errorf("ParseNodes() expected 2 nodes, got %d", len(nodes))
	}
	
	// Verify node structure
	for i, node := range nodes {
		if node.Tag == "" {
			t.Errorf("ParseNodes() node %d has empty tag", i)
		}
		if node.Type != "vmess" {
			t.Errorf("ParseNodes() node %d has type %s, expected vmess", i, node.Type)
		}
	}
}

func TestGenerator(t *testing.T) {
	tmpFile := createTestFile(t, testVmessNode)
	defer os.Remove(tmpFile)
	
	generator := singgen.NewGenerator(
		singgen.WithTemplate("v1.12"),
		singgen.WithPlatform("linux"),
		singgen.WithHTTPTimeout(5*time.Second),
	)
	
	t.Run("Generate", func(t *testing.T) {
		ctx := context.Background()
		
		cfg, err := generator.Generate(ctx, tmpFile)
		if err != nil {
			t.Errorf("Generator.Generate() error = %v", err)
			return
		}
		
		if cfg == nil {
			t.Errorf("Generator.Generate() returned nil config")
		}
	})
	
	t.Run("ParseNodes", func(t *testing.T) {
		ctx := context.Background()
		
		nodes, err := generator.ParseNodes(ctx, tmpFile)
		if err != nil {
			t.Errorf("Generator.ParseNodes() error = %v", err)
			return
		}
		
		if len(nodes) == 0 {
			t.Errorf("Generator.ParseNodes() returned no nodes")
		}
	})
	
	t.Run("GetAvailableTemplates", func(t *testing.T) {
		templates, err := generator.GetAvailableTemplates()
		if err != nil {
			t.Errorf("Generator.GetAvailableTemplates() error = %v", err)
			return
		}
		
		if len(templates) == 0 {
			t.Errorf("Generator.GetAvailableTemplates() returned no templates")
		}
	})
}

func TestOptions(t *testing.T) {
	// Test that we can create generators with different options
	// Since we can't access internal options directly, we test behavior indirectly
	
	t.Run("WithTemplate", func(t *testing.T) {
		generator := singgen.NewGenerator(singgen.WithTemplate("v1.12"))
		templates, err := generator.GetAvailableTemplates()
		if err != nil {
			t.Errorf("Failed to get templates: %v", err)
		}
		if len(templates) == 0 {
			t.Errorf("Expected some templates to be available")
		}
	})
	
	t.Run("WithPlatformDefaults", func(t *testing.T) {
		// Test that platform-specific defaults can be created
		linuxGen := singgen.NewGenerator(singgen.WithLinuxDefaults())
		darwinGen := singgen.NewGenerator(singgen.WithDarwinDefaults()) 
		iosGen := singgen.NewGenerator(singgen.WithIOSDefaults())
		
		// All should be creatable without error
		if linuxGen == nil || darwinGen == nil || iosGen == nil {
			t.Errorf("Platform defaults should create valid generators")
		}
	})
	
	t.Run("WithPerformanceOptimized", func(t *testing.T) {
		generator := singgen.NewGenerator(singgen.WithPerformanceOptimized())
		if generator == nil {
			t.Errorf("Performance optimized generator should be created successfully")
		}
	})
}

func TestContextCancellation(t *testing.T) {
	tmpFile := createTestFile(t, testVmessNode)
	defer os.Remove(tmpFile)
	
	t.Run("GenerateConfig with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, err := singgen.GenerateConfig(ctx, tmpFile)
		if err == nil {
			t.Errorf("GenerateConfig() with canceled context should return error")
		}
	})
	
	t.Run("GenerateConfig with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(1 * time.Millisecond) // Ensure timeout
		
		_, err := singgen.GenerateConfig(ctx, tmpFile)
		if err == nil {
			t.Errorf("GenerateConfig() with expired context should return error")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		ctx := context.Background()
		
		_, err := singgen.GenerateConfig(ctx, "/non/existent/file")
		if err == nil {
			t.Errorf("GenerateConfig() with non-existent file should return error")
		}
	})
	
	t.Run("invalid data", func(t *testing.T) {
		tmpFile := createTestFile(t, "invalid data")
		defer os.Remove(tmpFile)
		
		ctx := context.Background()
		
		_, err := singgen.GenerateConfig(ctx, tmpFile)
		if err == nil {
			t.Errorf("GenerateConfig() with invalid data should return error")
		}
	})
}

// Benchmark tests
func BenchmarkGenerateConfig(b *testing.B) {
	tmpFile := createTestFile(b, testSubscription)
	defer os.Remove(tmpFile)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := singgen.GenerateConfig(ctx, tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNodes(b *testing.B) {
	tmpFile := createTestFile(b, testSubscription)
	defer os.Remove(tmpFile)
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := singgen.ParseNodes(ctx, tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper function to create test files
func createTestFile(t testing.TB, content string) string {
	tmpFile, err := os.CreateTemp("", "singgen-test-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatal(err)
	}
	
	tmpFile.Close()
	return tmpFile.Name()
}