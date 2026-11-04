package singgen

import (
	"log/slog"
	"time"
)

// Option configures the generation process using the functional options pattern
type Option func(*GenerateOptions)

// WithTemplate sets the template version to use (e.g., "v1.14", "v1.12"); defaults to "v1.14"
func WithTemplate(version string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optTemplate)
		o.TemplateVersion = version
	}
}

// WithPlatform sets the target platform ("linux", "darwin", "ios")
func WithPlatform(platform string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optPlatform)
		o.Platform = platform
	}
}

// WithHTTPTimeout sets the timeout for HTTP requests
func WithHTTPTimeout(timeout time.Duration) Option {
	return func(o *GenerateOptions) {
		o.markSet(optHTTPTimeout)
		o.HTTPTimeout = timeout
	}
}

// WithMirrorURL sets the mirror URL for downloading rule sets
func WithMirrorURL(url string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optMirrorURL)
		o.MirrorURL = url
	}
}

// WithDNSServer sets the DNS server address
func WithDNSServer(server string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optDNSServer)
		o.DNSLocalServer = server
	}
}

// WithClientSubnet sets the client subnet for DNS queries
func WithClientSubnet(subnet string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optClientSubnet)
		o.ClientSubnet = subnet
	}
}

// WithOutputFormat sets the output format ("json" or "yaml")
func WithOutputFormat(format string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optFormat)
		o.Format = format
	}
}

// WithEmojiRemoval enables or disables emoji removal from node tags
func WithEmojiRemoval(remove bool) Option {
	return func(o *GenerateOptions) {
		o.markSet(optRemoveEmoji)
		o.RemoveEmoji = remove
	}
}

// WithBandwidthParams sets the bandwidth parameters for Hysteria2 nodes (in Mbps)
// Default values are up=25, down=300 if not specified
func WithBandwidthParams(upMbps, downMbps int) Option {
	return func(o *GenerateOptions) {
		o.markSet(optBandwidth)
		o.UpMbps = upMbps
		o.DownMbps = downMbps
	}
}

// WithExternalController sets the external controller address for Clash API
func WithExternalController(addr string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optExternalController)
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
		o.markSet(optPlatform)
		o.markSet(optExternalController)
		o.Platform = "linux"
		o.ExternalController = "127.0.0.1:9095"
	}
}

// WithDarwinDefaults applies defaults optimized for macOS platform
func WithDarwinDefaults() Option {
	return func(o *GenerateOptions) {
		o.markSet(optPlatform)
		o.markSet(optExternalController)
		o.Platform = "darwin"
		o.ExternalController = "127.0.0.1:9095"
	}
}

// WithIOSDefaults applies defaults optimized for iOS platform
func WithIOSDefaults() Option {
	return func(o *GenerateOptions) {
		o.markSet(optPlatform)
		o.markSet(optExternalController)
		o.Platform = "ios"
		// iOS typically uses different controller settings
		o.ExternalController = ""
	}
}

// WithPerformanceOptimized applies settings optimized for performance
func WithPerformanceOptimized() Option {
	return func(o *GenerateOptions) {
		o.markSet(optHTTPTimeout)
		o.markSet(optRemoveEmoji)
		o.HTTPTimeout = 15 * time.Second // Faster timeout
		o.RemoveEmoji = false            // Skip emoji processing
	}
}

// WithStrictSecurity applies security-focused settings
func WithStrictSecurity() Option {
	return func(o *GenerateOptions) {
		o.markSet(optHTTPTimeout)
		o.HTTPTimeout = 10 * time.Second // Short timeout to reduce attack surface
		// Additional security settings could be added here
	}
}

// WithTSAuthKey sets the Tailscale authentication key
func WithTSAuthKey(authKey string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optTSAuthKey)
		o.TSAuthKey = authKey
	}
}

// WithTSLanIPCIDR sets the Tailscale LAN IP CIDR
func WithTSLanIPCIDR(ipCIDR string) Option {
	return func(o *GenerateOptions) {
		o.markSet(optTSLanIPCIDR)
		o.TSLanIPCIDR = ipCIDR
	}
}

// optionField identifies a GenerateOptions field that was explicitly set
// through an Option, as opposed to holding a library default.
type optionField uint32

const (
	optTemplate optionField = 1 << iota
	optPlatform
	optHTTPTimeout
	optMirrorURL
	optDNSServer
	optClientSubnet
	optFormat
	optRemoveEmoji
	optBandwidth
	optExternalController
	optTSAuthKey
	optTSLanIPCIDR
)

func (o *GenerateOptions) markSet(f optionField) { o.explicit |= f }

func (o GenerateOptions) isSet(f optionField) bool { return o.explicit&f != 0 }
