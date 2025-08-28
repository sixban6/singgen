package singgen

import (
	"log/slog"
	"time"
)

// Option configures the generation process using the functional options pattern
type Option func(*GenerateOptions)

// WithTemplate sets the template version to use (e.g., "v1.12", "v1.13")
func WithTemplate(version string) Option {
	return func(o *GenerateOptions) {
		o.TemplateVersion = version
	}
}

// WithPlatform sets the target platform ("linux", "darwin", "ios")
func WithPlatform(platform string) Option {
	return func(o *GenerateOptions) {
		o.Platform = platform
	}
}

// WithHTTPTimeout sets the timeout for HTTP requests
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(o *GenerateOptions) {
		o.HTTPTimeout = timeout
	}
}

// WithMirrorURL sets the mirror URL for downloading rule sets
func WithMirrorURL(url string) Option {
	return func(o *GenerateOptions) {
		o.MirrorURL = url
	}
}

// WithDNSServer sets the DNS server address
func WithDNSServer(server string) Option {
	return func(o *GenerateOptions) {
		o.DNSLocalServer = server
	}
}

// WithClientSubnet sets the client subnet for DNS queries
func WithClientSubnet(subnet string) Option {
	return func(o *GenerateOptions) {
		o.ClientSubnet = subnet
	}
}

// WithOutputFormat sets the output format ("json" or "yaml")
func WithOutputFormat(format string) Option {
	return func(o *GenerateOptions) {
		o.Format = format
	}
}

// WithEmojiRemoval enables or disables emoji removal from node tags
func WithEmojiRemoval(remove bool) Option {
	return func(o *GenerateOptions) {
		o.RemoveEmoji = remove
	}
}

// WithExternalController sets the external controller address for Clash API
func WithExternalController(addr string) Option {
	return func(o *GenerateOptions) {
		o.ExternalController = addr
	}
}

// WithLogger sets the logger for the generation process
func WithLogger(logger *slog.Logger) Option {
	return func(o *GenerateOptions) {
		o.Logger = logger
	}
}

// WithDefaults applies sensible defaults for common use cases
func WithDefaults() Option {
	return func(o *GenerateOptions) {
		// Defaults are already set in buildOptions, this is a no-op
		// but provides a way for users to explicitly request defaults
	}
}

// WithLinuxDefaults applies defaults optimized for Linux platform
func WithLinuxDefaults() Option {
	return func(o *GenerateOptions) {
		o.Platform = "linux"
		o.ExternalController = "127.0.0.1:9095"
	}
}

// WithDarwinDefaults applies defaults optimized for macOS platform
func WithDarwinDefaults() Option {
	return func(o *GenerateOptions) {
		o.Platform = "darwin"
		o.ExternalController = "127.0.0.1:9095"
	}
}

// WithIOSDefaults applies defaults optimized for iOS platform
func WithIOSDefaults() Option {
	return func(o *GenerateOptions) {
		o.Platform = "ios"
		// iOS typically uses different controller settings
		o.ExternalController = ""
	}
}

// WithPerformanceOptimized applies settings optimized for performance
func WithPerformanceOptimized() Option {
	return func(o *GenerateOptions) {
		o.HTTPTimeout = 15 * time.Second // Faster timeout
		o.RemoveEmoji = false // Skip emoji processing
	}
}

// WithStrictSecurity applies security-focused settings
func WithStrictSecurity() Option {
	return func(o *GenerateOptions) {
		o.HTTPTimeout = 10 * time.Second // Short timeout to reduce attack surface
		// Additional security settings could be added here
	}
}