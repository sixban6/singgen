package singgen_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/sixban6/singgen/pkg/singgen"
	"github.com/sixban6/singgen/internal/renderer"
)

// Helper function to create test files for pipeline tests
func createPipelineTestFile(t testing.TB, content string) string {
	tmpFile, err := os.CreateTemp("", "singgen-pipeline-test-*.txt")
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

func TestPipeline(t *testing.T) {
	tmpFile := createPipelineTestFile(t, testSubscription)
	defer os.Remove(tmpFile)
	
	pipeline := singgen.NewPipeline()
	
	t.Run("Execute", func(t *testing.T) {
		ctx := context.Background()
		
		result, err := pipeline.Execute(ctx, tmpFile)
		if err != nil {
			t.Errorf("Pipeline.Execute() error = %v", err)
			return
		}
		
		if result == nil {
			t.Errorf("Pipeline.Execute() returned nil result")
			return
		}
		
		if result.Config == nil {
			t.Errorf("Pipeline.Execute() returned nil config")
		}
		
		if len(result.Nodes) == 0 {
			t.Errorf("Pipeline.Execute() returned no nodes")
		}
		
		if len(result.Outbounds) == 0 {
			t.Errorf("Pipeline.Execute() returned no outbounds")
		}
		
		// Verify metadata
		if result.Metadata.SourceType != "file" {
			t.Errorf("Pipeline.Execute() metadata.SourceType = %s, expected file", result.Metadata.SourceType)
		}
		
		if result.Metadata.NodesCount == 0 {
			t.Errorf("Pipeline.Execute() metadata.NodesCount = 0")
		}
		
		if result.Metadata.DetectedFormat == "" {
			t.Errorf("Pipeline.Execute() metadata.DetectedFormat is empty")
		}
	})
	
	t.Run("ExecuteBytes", func(t *testing.T) {
		ctx := context.Background()
		
		data, err := pipeline.ExecuteBytes(ctx, tmpFile)
		if err != nil {
			t.Errorf("Pipeline.ExecuteBytes() error = %v", err)
			return
		}
		
		if len(data) == 0 {
			t.Errorf("Pipeline.ExecuteBytes() returned empty data")
		}
	})
	
	t.Run("FetchOnly", func(t *testing.T) {
		ctx := context.Background()
		
		data, err := pipeline.FetchOnly(ctx, tmpFile)
		if err != nil {
			t.Errorf("Pipeline.FetchOnly() error = %v", err)
			return
		}
		
		if len(data) == 0 {
			t.Errorf("Pipeline.FetchOnly() returned empty data")
		}
	})
	
	t.Run("ParseOnly", func(t *testing.T) {
		ctx := context.Background()
		
		nodes, err := pipeline.ParseOnly(ctx, tmpFile)
		if err != nil {
			t.Errorf("Pipeline.ParseOnly() error = %v", err)
			return
		}
		
		if len(nodes) == 0 {
			t.Errorf("Pipeline.ParseOnly() returned no nodes")
		}
	})
	
	t.Run("TransformOnly", func(t *testing.T) {
		ctx := context.Background()
		
		outbounds, err := pipeline.TransformOnly(ctx, tmpFile)
		if err != nil {
			t.Errorf("Pipeline.TransformOnly() error = %v", err)
			return
		}
		
		if len(outbounds) == 0 {
			t.Errorf("Pipeline.TransformOnly() returned no outbounds")
		}
	})
}

func TestPipelineWithCustomComponents(t *testing.T) {
	tmpFile := createPipelineTestFile(t, testVmessNode)
	defer os.Remove(tmpFile)
	
	// Create pipeline with custom components
	pipeline := singgen.NewPipeline(
		singgen.WithCustomRenderer(renderer.NewYAMLRenderer()),
		singgen.WithPipelineOptions(
			singgen.WithTemplate("v1.12"),
			singgen.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))),
		),
	)
	
	ctx := context.Background()
	
	result, err := pipeline.Execute(ctx, tmpFile)
	if err != nil {
		t.Errorf("Pipeline with custom components error = %v", err)
		return
	}
	
	if result == nil {
		t.Errorf("Pipeline with custom components returned nil result")
	}
}

func TestStreamingPipeline(t *testing.T) {
	tmpFile := createPipelineTestFile(t, testSubscription)
	defer os.Remove(tmpFile)
	
	streaming := singgen.NewStreamingPipeline()
	
	t.Run("StreamNodes", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		nodesCh, errCh := streaming.StreamNodes(ctx, tmpFile, 1)
		
		var totalNodes int
		var batchCount int
		
	loop:
		for {
			select {
			case nodes, ok := <-nodesCh:
				if !ok {
					break loop
				}
				totalNodes += len(nodes)
				batchCount++
				
				if len(nodes) == 0 {
					t.Errorf("StreamNodes() returned empty batch")
				}
				
			case err := <-errCh:
				if err != nil {
					t.Errorf("StreamNodes() error = %v", err)
					return
				}
				
			case <-ctx.Done():
				t.Errorf("StreamNodes() timed out")
				return
			}
		}
		
		if totalNodes == 0 {
			t.Errorf("StreamNodes() returned no nodes total")
		}
		
		if batchCount == 0 {
			t.Errorf("StreamNodes() returned no batches")
		}
	})
}

// Mock components for testing - we need to import internal types since we can't access the interfaces directly

// We'll create a simplified mock test that doesn't rely on internal interfaces
// Instead, focus on testing the public API behavior

func TestPipelinePublicAPI(t *testing.T) {
	// Test that we can create pipeline with public options
	pipeline := singgen.NewPipeline(
		singgen.WithPipelineOptions(
			singgen.WithTemplate("v1.12"),
			singgen.WithPlatform("linux"),
		),
	)
	
	// Create test data file
	testData := `vmess://eyJhZGQiOiJ0ZXN0LmV4YW1wbGUuY29tIiwiYWlkIjowLCJhbHBuIjoiIiwiaG9zdCI6IiIsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LWFiY2QtMTIzNDU2Nzg5YWJjIiwibmV0IjoidGNwIiwicGF0aCI6IiIsInBvcnQiOjEwODA5LCJwcyI6IlRlc3QgTm9kZSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
	tmpFile := createPipelineTestFile(t, testData)
	defer os.Remove(tmpFile)
	
	ctx := context.Background()
	
	result, err := pipeline.Execute(ctx, tmpFile)
	if err != nil {
		t.Errorf("Pipeline public API test error = %v", err)
		return
	}
	
	if result == nil {
		t.Errorf("Pipeline returned nil result")
		return
	}
	
	if len(result.Nodes) == 0 {
		t.Errorf("Pipeline returned no nodes")
	}
	
	if len(result.Outbounds) == 0 {
		t.Errorf("Pipeline returned no outbounds")
	}
	
	if result.Metadata.SourceType != "file" {
		t.Errorf("Expected source type 'file', got '%s'", result.Metadata.SourceType)
	}
}

func TestPipelineErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		expectError bool
	}{
		{
			name:        "non-existent file",
			source:      "/non/existent/file.txt",
			expectError: true,
		},
		{
			name:        "empty source",
			source:      "",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := singgen.NewPipeline()
			ctx := context.Background()
			
			_, err := pipeline.Execute(ctx, tt.source)
			
			if tt.expectError && err == nil {
				t.Errorf("Pipeline.Execute() expected error, got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Pipeline.Execute() unexpected error = %v", err)
			}
		})
	}
	
	// Test invalid data
	t.Run("invalid data", func(t *testing.T) {
		tmpFile := createPipelineTestFile(t, "invalid data")
		defer os.Remove(tmpFile)
		
		pipeline := singgen.NewPipeline()
		ctx := context.Background()
		
		_, err := pipeline.Execute(ctx, tmpFile)
		if err == nil {
			t.Errorf("Pipeline.Execute() with invalid data should return error")
		}
	})
}

func BenchmarkPipeline(b *testing.B) {
	tmpFile := createPipelineTestFile(b, testSubscription)
	defer os.Remove(tmpFile)
	
	pipeline := singgen.NewPipeline()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Execute(ctx, tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPipelineReal(b *testing.B) {
	// Test with real data for more accurate benchmarks
	testData := `vmess://eyJhZGQiOiJ0ZXN0LmV4YW1wbGUuY29tIiwiYWlkIjowLCJhbHBuIjoiIiwiaG9zdCI6IiIsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LWFiY2QtMTIzNDU2Nzg5YWJjIiwibmV0IjoidGNwIiwicGF0aCI6IiIsInBvcnQiOjEwODA5LCJwcyI6IlRlc3QgTm9kZSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
	tmpFile := createPipelineTestFile(b, testData)
	defer os.Remove(tmpFile)
	
	pipeline := singgen.NewPipeline()
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := pipeline.Execute(ctx, tmpFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}