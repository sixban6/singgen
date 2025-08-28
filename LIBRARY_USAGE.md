# Singgen Library Usage Guide

Singgen has been transformed into a powerful, reusable Go library for generating sing-box configurations from subscription URLs and various proxy formats.

## Features

- **Multiple API Levels**: High-level, mid-level, and low-level APIs for different use cases
- **Context-Aware**: Full support for `context.Context` with cancellation and timeouts
- **High Performance**: Concurrent processing and optimized algorithms
- **Extensible**: Plugin architecture allows custom components
- **Type-Safe**: Comprehensive type definitions and interfaces
- **Well-Tested**: Extensive test suite with benchmarks

## Quick Start

### High-Level API (Recommended for most users)

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/sixban6/singgen/pkg/singgen"
)

func main() {
    ctx := context.Background()
    
    // Simple usage with defaults
    config, err := singgen.GenerateConfig(ctx, "https://example.com/subscription")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated config with %d outbounds\n", len(config.Outbounds))
    
    // Get configuration as bytes
    data, err := singgen.GenerateConfigBytes(ctx, 
        "https://example.com/subscription",
        singgen.WithTemplate("v1.12"),
        singgen.WithPlatform("linux"),
        singgen.WithOutputFormat("json"))
    if err != nil {
        log.Fatal(err)
    }
    
    // Write to file or use data
    fmt.Printf("Generated %d bytes of configuration\n", len(data))
}
```

### Mid-Level API (More Control)

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/sixban6/singgen/pkg/singgen"
)

func main() {
    // Create logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    
    // Create generator with custom options
    generator := singgen.NewGenerator(
        singgen.WithTemplate("v1.12"),
        singgen.WithPlatform("darwin"),
        singgen.WithHTTPTimeout(30*time.Second),
        singgen.WithLogger(logger),
        singgen.WithMirrorURL("https://custom-mirror.com"),
    )
    
    ctx := context.Background()
    
    // Generate configuration
    config, err := generator.Generate(ctx, "subscription.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    // Just parse nodes without generating config
    nodes, err := generator.ParseNodes(ctx, "subscription.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    // List available templates
    templates, err := generator.GetAvailableTemplates()
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Available templates: %v\n", templates)
}
```

### Low-Level API (Maximum Control)

```go
package main

import (
    "context"
    
    "github.com/sixban6/singgen/pkg/singgen"
)

func main() {
    // Create pipeline with custom components
    pipeline := singgen.NewPipeline(
        singgen.WithPipelineOptions(
            singgen.WithTemplate("v1.12"),
            singgen.WithDNSServer("1.1.1.1"),
        ),
    )
    
    ctx := context.Background()
    
    // Execute full pipeline with detailed results
    result, err := pipeline.Execute(ctx, "https://example.com/sub")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Pipeline Results:\n")
    fmt.Printf("- Source Type: %s\n", result.Metadata.SourceType)
    fmt.Printf("- Detected Format: %s\n", result.Metadata.DetectedFormat)
    fmt.Printf("- Nodes Count: %d\n", result.Metadata.NodesCount)
    fmt.Printf("- Outbounds Count: %d\n", result.Metadata.OutboundsCount)
    
    // Execute individual steps
    data, err := pipeline.FetchOnly(ctx, "subscription.txt")
    if err != nil {
        log.Fatal(err)
    }
    
    nodes, err := pipeline.ParseOnly(ctx, "subscription.txt") 
    if err != nil {
        log.Fatal(err)
    }
    
    outbounds, err := pipeline.TransformOnly(ctx, "subscription.txt")
    if err != nil {
        log.Fatal(err)
    }
}
```

## Configuration Options

### Platform-Specific Defaults

```go
// Linux defaults
singgen.WithLinuxDefaults()

// macOS defaults  
singgen.WithDarwinDefaults()

// iOS defaults
singgen.WithIOSDefaults()
```

### Performance Tuning

```go
// Optimized for performance
singgen.WithPerformanceOptimized()

// Security-focused settings
singgen.WithStrictSecurity()
```

### Custom Components

```go
// Create pipeline with custom components
pipeline := singgen.NewPipeline(
    singgen.WithCustomFetcher(myFetcher),
    singgen.WithCustomParser(myParser),
    singgen.WithCustomTransformer(myTransformer),
    singgen.WithCustomTemplate(myTemplate),
    singgen.WithCustomRenderer(myRenderer),
)
```

## Streaming API

For processing large subscription files:

```go
streaming := singgen.NewStreamingPipeline()

nodesCh, errCh := streaming.StreamNodes(ctx, "large-subscription.txt", 100)

for {
    select {
    case nodes, ok := <-nodesCh:
        if !ok {
            return // Done
        }
        // Process batch of nodes
        fmt.Printf("Processing batch of %d nodes\n", len(nodes))
        
    case err := <-errCh:
        if err != nil {
            log.Fatal(err)
        }
    }
}
```

## Error Handling

The library provides structured error types:

```go
config, err := singgen.GenerateConfig(ctx, source)
if err != nil {
    switch {
    case errors.Is(err, singgen.ErrEmptySource):
        fmt.Println("Please provide a valid source URL or file path")
    case errors.Is(err, singgen.ErrNoValidNodes):
        fmt.Println("No valid proxy nodes found in the source")
    case errors.Is(err, singgen.ErrUnsupportedFormat):
        fmt.Println("The source format is not supported")
    default:
        fmt.Printf("Generation failed: %v\n", err)
    }
}
```

## Performance

Benchmark results on Apple M1:

```
BenchmarkGenerateConfig-10    	    1248	    820790 ns/op  (~821µs)
BenchmarkParseNodes-10        	   32541	     36437 ns/op  (~36µs)
```

The library is highly optimized with:
- Concurrent processing for large subscription files
- Smart format detection
- Memory-efficient parsing
- Context-aware cancellation

## Migration from CLI

The CLI has been refactored to use the library, reducing the main.go from ~244 lines to ~172 lines while maintaining full backward compatibility.

## Thread Safety

All library components are thread-safe and can be used concurrently. The `Generator` and `Pipeline` types can be reused across multiple goroutines.

## Examples

See the `test/` directory for comprehensive examples and the CLI implementation in `cmd/singgen/main.go` for real-world usage.