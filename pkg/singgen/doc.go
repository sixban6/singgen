// Package singgen provides a high-performance, context-aware library for generating
// sing-box configurations from subscription URLs and various proxy formats.
//
// # Overview
//
// Singgen transforms proxy subscription URLs and files into sing-box configurations
// with support for multiple protocols (VMess, VLESS, Trojan, Hysteria2, Shadowsocks)
// and platforms (Linux, macOS, iOS).
//
// # API Levels
//
// The library provides three API levels for different use cases:
//
//   - High-level API: Simple functions with sensible defaults
//   - Mid-level API: Generator with configurable options
//   - Low-level API: Pipeline with custom components
//
// # High-Level API
//
// The high-level API is recommended for most users. It provides simple functions
// that handle the complete generation process:
//
//	config, err := singgen.GenerateConfig(ctx, "https://example.com/sub",
//		singgen.WithTemplate("v1.12"),
//		singgen.WithPlatform("linux"))
//
//	// Get configuration as bytes
//	data, err := singgen.GenerateConfigBytes(ctx, "subscription.txt",
//		singgen.WithOutputFormat("yaml"))
//
//	// Just parse nodes without generating configuration
//	nodes, err := singgen.ParseNodes(ctx, "https://example.com/sub")
//
// # Mid-Level API
//
// The mid-level API provides more control over the generation process through
// the Generator type:
//
//	generator := singgen.NewGenerator(
//		singgen.WithTemplate("v1.12"),
//		singgen.WithPlatform("darwin"),
//		singgen.WithHTTPTimeout(30*time.Second),
//		singgen.WithLogger(logger))
//
//	config, err := generator.Generate(ctx, "subscription.txt")
//	nodes, err := generator.ParseNodes(ctx, "subscription.txt")
//	templates, err := generator.GetAvailableTemplates()
//
// # Low-Level API
//
// The low-level API provides maximum control through the Pipeline type,
// allowing custom components to be injected:
//
//	pipeline := singgen.NewPipeline(
//		singgen.WithCustomFetcher(myFetcher),
//		singgen.WithCustomParser(myParser),
//		singgen.WithCustomTransformer(myTransformer))
//
//	result, err := pipeline.Execute(ctx, "https://example.com/sub")
//	// result contains detailed metadata about the generation process
//
//	// Execute individual steps
//	data, err := pipeline.FetchOnly(ctx, "subscription.txt")
//	nodes, err := pipeline.ParseOnly(ctx, "subscription.txt")
//	outbounds, err := pipeline.TransformOnly(ctx, "subscription.txt")
//
// # Configuration Options
//
// The library supports extensive configuration through functional options:
//
//	// Template and platform
//	singgen.WithTemplate("v1.12")
//	singgen.WithPlatform("linux")
//
//	// Network settings
//	singgen.WithHTTPTimeout(30 * time.Second)
//	singgen.WithMirrorURL("https://custom-mirror.com")
//	singgen.WithDNSServer("1.1.1.1")
//	singgen.WithClientSubnet("202.101.170.1/24")
//
//	// Output settings
//	singgen.WithOutputFormat("yaml")
//	singgen.WithEmojiRemoval(true)
//	singgen.WithExternalController("127.0.0.1:9095")
//
//	// Logging
//	singgen.WithLogger(logger)
//
//	// Platform-specific defaults
//	singgen.WithLinuxDefaults()
//	singgen.WithDarwinDefaults()
//	singgen.WithIOSDefaults()
//
//	// Performance tuning
//	singgen.WithPerformanceOptimized()
//	singgen.WithStrictSecurity()
//
// # Context Support
//
// All operations support context.Context for cancellation and timeouts:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	config, err := singgen.GenerateConfig(ctx, url)
//	if errors.Is(err, context.DeadlineExceeded) {
//		// Handle timeout
//	}
//
// # Error Handling
//
// The library provides structured error types for different failure scenarios:
//
//	config, err := singgen.GenerateConfig(ctx, source)
//	if err != nil {
//		switch {
//		case errors.Is(err, singgen.ErrEmptySource):
//			// Handle empty source
//		case errors.Is(err, singgen.ErrNoValidNodes):
//			// Handle no valid nodes
//		case errors.Is(err, singgen.ErrUnsupportedFormat):
//			// Handle unsupported format
//		default:
//			// Handle other errors
//		}
//	}
//
// # Supported Protocols
//
// The library supports parsing and generating configurations for:
//
//   - VMess (vmess://)
//   - VLESS (vless://)
//   - Trojan (trojan://)
//   - Hysteria2 (hysteria2://, hy2://)
//   - Shadowsocks (ss://)
//   - Mixed subscriptions with multiple protocols
//
// # Performance
//
// The library is optimized for performance with:
//
//   - Concurrent processing for large subscription files
//   - Smart format detection to avoid unnecessary parsing attempts
//   - Memory-efficient parsing and processing
//   - Context-aware cancellation to prevent resource leaks
//
// Benchmark results on Apple M1:
//
//	BenchmarkGenerateConfig-10    1248    820790 ns/op  (~821µs)
//	BenchmarkParseNodes-10       32541     36437 ns/op   (~36µs)
//
// # Thread Safety
//
// All library components are thread-safe and can be used concurrently.
// Generator and Pipeline instances can be reused across multiple goroutines.
//
// # Streaming API
//
// For processing large subscription files, use the streaming API:
//
//	streaming := singgen.NewStreamingPipeline()
//	nodesCh, errCh := streaming.StreamNodes(ctx, "large-subscription.txt", 100)
//
//	for {
//		select {
//		case nodes, ok := <-nodesCh:
//			if !ok { return } // Done
//			// Process batch of nodes
//		case err := <-errCh:
//			// Handle error
//		}
//	}
package singgen