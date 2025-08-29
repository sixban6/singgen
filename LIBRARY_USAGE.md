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

### Single Subscription API (Recommended for single source)

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

### Multi-Subscription API (Recommended for multiple sources)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/sixban6/singgen/pkg/singgen"
)

func main() {
    ctx := context.Background()
    
    // Method 1: Load from configuration file
    config, err := singgen.GenerateConfigFromFile(ctx, "config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Generated config from file with %d outbounds\n", len(config.Outbounds))
    
    // Method 2: Load from configuration file and get bytes
    data, err := singgen.GenerateConfigBytesFromFile(ctx, "config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Generated %d bytes of configuration\n", len(data))
    
    // Method 3: Programmatically construct multi-config
    multiConfig := &singgen.MultiConfig{
        Global: singgen.GlobalConfig{
            Template:       "v1.12",
            Platform:       "linux", 
            MirrorURL:      "https://ghfast.top",
            DNSLocalServer: "114.114.114.114",
            WebUIAddress:   "127.0.0.1:9095",
            RemoveEmoji:    true,
            SkipTLSVerify:  false,
            HTTPTimeout:    30 * time.Second,
            Format:         "json",
        },
        Subscriptions: []singgen.SubscriptionConfig{
            {
                Name:         "primary",
                URL:          "https://provider1.example.com/sub",
                RemoveEmoji:  &[]bool{false}[0], // Override global setting
                SkipTLSVerify: &[]bool{true}[0],
            },
            {
                Name:         "backup",
                URL:          "https://provider2.example.com/sub", 
                RemoveEmoji:  &[]bool{true}[0],  // Use emoji removal
                SkipTLSVerify: &[]bool{false}[0],
            },
        },
    }
    
    // Generate config from multi-config
    config, err = singgen.GenerateConfigFromMulti(ctx, multiConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get bytes from multi-config
    data, err = singgen.GenerateConfigBytesFromMulti(ctx, multiConfig, 
        singgen.WithLogger(logger))
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Generated multi-subscription config: %d bytes\n", len(data))
}
```

### Configuration File Loading

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/sixban6/singgen/pkg/singgen"
)

func main() {
    // Load from specific file
    config, err := singgen.LoadConfigFile("./my-config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Auto-discovery (searches predefined paths)
    config, err = singgen.LoadConfigAuto()
    if err != nil {
        log.Fatal(err)
    }
    
    // Validate configuration
    if err := config.ValidateConfig(); err != nil {
        log.Fatal(err)
    }
    
    // Generate example configuration
    example := singgen.GenerateExampleConfig()
    if err := singgen.SaveConfigFile(example, "example.yaml", "yaml"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Configuration loaded and validated successfully")
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

## Multi-Subscription Configuration File Format

### YAML Format Example

```yaml
global:
  template: v1.12
  platform: linux
  mirror_url: https://ghfast.top
  dns_server: 114.114.114.114
  webui_address: 127.0.0.1:9095
  remove_emoji: true
  skip_tls_verify: false
  http_timeout: 30s
  format: json
  client_subnet: 202.101.170.1/24  # Optional

subscriptions:
  - name: primary
    url: https://provider1.example.com/subscription
    remove_emoji: false      # Override global setting
    skip_tls_verify: true    # Override global setting
    http_timeout: 10s        # Optional, override global
    
  - name: backup
    url: https://provider2.example.com/subscription
    remove_emoji: true       # Use emoji removal
    skip_tls_verify: false
    
  - name: local-file
    url: ./local-nodes.txt
    # Uses global defaults for other settings
```

### JSON Format Example

```json
{
  "global": {
    "template": "v1.12",
    "platform": "linux",
    "mirror_url": "https://ghfast.top",
    "dns_server": "114.114.114.114",
    "webui_address": "127.0.0.1:9095",
    "remove_emoji": true,
    "skip_tls_verify": false,
    "http_timeout": "30s",
    "format": "json"
  },
  "subscriptions": [
    {
      "name": "primary",
      "url": "https://provider1.example.com/subscription",
      "remove_emoji": false,
      "skip_tls_verify": true
    },
    {
      "name": "backup", 
      "url": "https://provider2.example.com/subscription",
      "remove_emoji": true,
      "skip_tls_verify": false
    }
  ]
}
```

### Configuration Priority

Settings are applied with the following priority (highest to lowest):
1. Subscription-specific settings (only `remove_emoji`, `skip_tls_verify`, `http_timeout`)
2. Global configuration settings
3. Library default settings

### Configuration File Search Paths

When using `LoadConfigAuto()`, the library searches these paths in order:
1. `./singgen.yaml`
2. `./singgen.json`
3. `~/.config/singgen/config.yaml`
4. `~/.config/singgen/config.json`
5. `/etc/singgen/config.yaml`
6. `/etc/singgen/config.json`

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
// Single subscription error handling
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

// Multi-subscription error handling
config, err := singgen.GenerateConfigFromFile(ctx, "config.yaml")
if err != nil {
    switch {
    case errors.Is(err, singgen.ErrConfigFileNotFound):
        fmt.Println("Configuration file not found")
    case errors.Is(err, singgen.ErrInvalidConfigFormat):
        fmt.Println("Invalid configuration file format")
    case errors.Is(err, singgen.ErrNoSubscriptions):
        fmt.Println("No subscriptions configured")
    case errors.Is(err, singgen.ErrEmptySubscriptionName):
        fmt.Println("Subscription name cannot be empty")
    case errors.Is(err, singgen.ErrEmptySubscriptionURL):
        fmt.Println("Subscription URL cannot be empty")
    case errors.Is(err, singgen.ErrDuplicateSubscriptionName):
        fmt.Println("Duplicate subscription name found")
    case errors.Is(err, singgen.ErrNoValidNodes):
        fmt.Println("No valid nodes found in any subscription")
    default:
        fmt.Printf("Multi-subscription generation failed: %v\n", err)
    }
}
```

## Performance

Benchmark results on Apple M1:

### Single Subscription
```
BenchmarkGenerateConfig-10    	    1248	    820790 ns/op  (~821µs)
BenchmarkParseNodes-10        	   32541	     36437 ns/op  (~36µs)
```

### Multi-Subscription  
```
BenchmarkGenerateConfigFromFile-10     	   400	   3115167 ns/op  (~3.1ms)
BenchmarkGenerateConfigFromMulti-10   	   450	   2970708 ns/op  (~3.0ms)
```

The library is highly optimized with:
- Concurrent processing for large subscription files
- Smart format detection  
- Memory-efficient parsing
- Context-aware cancellation
- Per-subscription emoji processing
- Graceful error handling (single subscription failure doesn't affect others)

## Migration from CLI

The CLI has been enhanced to support multi-subscription functionality while maintaining full backward compatibility:

### Before (Single Subscription Only)
```bash
./singgen -url https://example.com/sub -out config.json
```

### After (Both Modes Supported)
```bash
# Single subscription mode (unchanged)
./singgen -url https://example.com/sub -out config.json

# Multi-subscription mode (new)
./singgen -config multi-config.yaml -out config.json

# Generate example config (new)
./singgen -generate-example -out example-config.yaml
```

## New Features

### Multi-Subscription Support
- ✅ **Configuration-driven**: Use YAML/JSON files for complex setups
- ✅ **Per-subscription settings**: Individual emoji removal, TLS verification, timeouts
- ✅ **Name prefixes**: Auto-prefix node names with subscription identifiers (`[primary] Node1`)
- ✅ **Graceful degradation**: Single subscription failure doesn't affect others
- ✅ **Mixed protocols**: Support SS, VMess, VLESS, Trojan, Hysteria2 in same config

### Enhanced APIs
- ✅ `GenerateConfigFromFile()` - Generate from config file
- ✅ `GenerateConfigFromMulti()` - Generate from MultiConfig struct
- ✅ `LoadConfigFile()` / `LoadConfigAuto()` - Configuration loading
- ✅ `GenerateExampleConfig()` - Create example configurations
- ✅ `SaveConfigFile()` - Save configurations

## Thread Safety

All library components are thread-safe and can be used concurrently. The `Generator` and `Pipeline` types can be reused across multiple goroutines.

## Examples

See the following for comprehensive examples:
- `test/` directory for unit tests and examples
- `TEST_COMMANDS.md` for testing commands
- `RUN_TESTS.md` for validation procedures  
- `EMOJI_TEST_RESULT.md` for emoji handling validation
- CLI implementation in `cmd/singgen/main.go` for real-world usage