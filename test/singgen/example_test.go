// Example demonstrates how to use the singgen library
package singgen_test

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/sixban6/singgen/pkg/singgen"
)

// Example_highLevelAPI demonstrates the high-level API usage
func Example_highLevelAPI() {
	ctx := context.Background()
	
	// Create test data
	testVmess := `vmess://eyJhZGQiOiJ0ZXN0LmV4YW1wbGUuY29tIiwiYWlkIjowLCJhbHBuIjoiIiwiaG9zdCI6IiIsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LWFiY2QtMTIzNDU2Nzg5YWJjIiwibmV0IjoidGNwIiwicGF0aCI6IiIsInBvcnQiOjEwODA5LCJwcyI6IlRlc3QgTm9kZSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
	
	// Write test data to a temporary file
	tmpFile, err := os.CreateTemp("", "singgen-example-*.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testVmess); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()
	
	// Simple usage with defaults
	config, err := singgen.GenerateConfig(ctx, tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Generated config with %d outbounds\n", len(config.Outbounds))
	
	// Get configuration as JSON bytes
	data, err := singgen.GenerateConfigBytes(ctx, tmpFile.Name(),
		singgen.WithTemplate("v1.12"),
		singgen.WithPlatform("linux"),
		singgen.WithOutputFormat("json"))
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Generated %d bytes of JSON configuration\n", len(data))
	
	// Just parse nodes without generating configuration
	nodes, err := singgen.ParseNodes(ctx, tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Parsed %d nodes\n", len(nodes))
	
	// Output:
	// Generated config with 44 outbounds
	// Generated 23741 bytes of JSON configuration
	// Parsed 1 nodes
}

// Example_midLevelAPI demonstrates the mid-level Generator API
func Example_midLevelAPI() {
	
	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Use WARN to reduce output in example
	}))
	
	// Create generator with options
	generator := singgen.NewGenerator(
		singgen.WithTemplate("v1.12"),
		singgen.WithPlatform("linux"),
		singgen.WithDNSServer("1.1.1.1"),
		singgen.WithLogger(logger),
	)
	
	// List available templates
	templates, err := generator.GetAvailableTemplates()
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Available templates: %v\n", templates)
	
	// Output:
	// Available templates: [v1.11 v1.12]
}

// Example_lowLevelAPI demonstrates the low-level Pipeline API
func Example_lowLevelAPI() {
	ctx := context.Background()
	
	// Create test data
	testVmess := `vmess://eyJhZGQiOiJ0ZXN0LmV4YW1wbGUuY29tIiwiYWlkIjowLCJhbHBuIjoiIiwiaG9zdCI6IiIsImlkIjoiMTIzNDU2NzgtYWJjZC0xMjM0LWFiY2QtMTIzNDU2Nzg5YWJjIiwibmV0IjoidGNwIiwicGF0aCI6IiIsInBvcnQiOjEwODA5LCJwcyI6IlRlc3QgTm9kZSIsInNjeSI6Im5vbmUiLCJzbmkiOiIiLCJ0bHMiOiIiLCJ0eXBlIjoibm9uZSIsInYiOiIyIn0=`
	
	tmpFile, err := os.CreateTemp("", "singgen-pipeline-example-*.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(testVmess); err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()
	
	// Create pipeline with options
	pipeline := singgen.NewPipeline(
		singgen.WithPipelineOptions(
			singgen.WithTemplate("v1.12"),
			singgen.WithPlatform("linux"),
		),
	)
	
	// Execute complete pipeline with detailed results
	result, err := pipeline.Execute(ctx, tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Pipeline Results:\n")
	fmt.Printf("- Source Type: %s\n", result.Metadata.SourceType)
	fmt.Printf("- Detected Format: %s\n", result.Metadata.DetectedFormat)
	fmt.Printf("- Nodes Count: %d\n", result.Metadata.NodesCount)
	fmt.Printf("- Outbounds Count: %d\n", result.Metadata.OutboundsCount)
	
	// Execute individual steps
	data, err := pipeline.FetchOnly(ctx, tmpFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Fetched %d bytes of raw data\n", len(data))
	
	// Output:
	// Pipeline Results:
	// - Source Type: file
	// - Detected Format: vmess
	// - Nodes Count: 1
	// - Outbounds Count: 1
	// Fetched 280 bytes of raw data
}

// Example_errorHandling demonstrates structured error handling
func Example_errorHandling() {
	ctx := context.Background()
	
	// Try to generate config from empty source
	_, err := singgen.GenerateConfig(ctx, "")
	
	fmt.Printf("Error type check: %v\n", err == singgen.ErrEmptySource)
	
	// Output:
	// Error type check: true
}