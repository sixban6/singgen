// Package singgen provides a high-performance, context-aware library for generating
// sing-box configurations from subscription URLs and various proxy formats.
//
// The library provides multiple API levels:
//
// High-level API (recommended for most users):
//
//	config, err := singgen.GenerateConfig(ctx, "https://example.com/sub",
//	  singgen.WithTemplate("v1.14"),
//	  singgen.WithPlatform("linux"))
//
// Multi-subscription API:
//
//	config, err := singgen.GenerateConfigFromFile(ctx, "config.yaml")
//
//	multiConfig := &singgen.MultiConfig{...}
//	config, err := singgen.GenerateConfigFromMulti(ctx, multiConfig)
//
// Mid-level API (for more control):
//
//	generator := singgen.NewGenerator(
//	  singgen.WithLogger(logger),
//	  singgen.WithHTTPTimeout(30*time.Second))
//	config, err := generator.Generate(ctx, source)
//
// Low-level API (maximum control):
//
//	pipeline := singgen.NewPipeline(
//	  singgen.WithCustomFetcher(myFetcher),
//	  singgen.WithCustomParser(myParser))
//	result, err := pipeline.Execute(ctx, source)
package singgen

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/registry"
	"github.com/sixban6/singgen/internal/renderer"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/pkg/model"
)

// GenerateOptions configures the generation process
type GenerateOptions struct {
	// Template configuration
	TemplateVersion string
	Platform        string

	// Network configuration
	HTTPTimeout    time.Duration
	MirrorURL      string
	ClientSubnet   string
	DNSLocalServer string

	// Output configuration
	Format      string
	RemoveEmoji bool

	// Hysteria2 bandwidth parameters (Mbps)
	UpMbps   int
	DownMbps int

	// TLS configuration
	SkipTLSVerify bool

	// External services
	ExternalController string

	// Tailscale configuration
	TSAuthKey   string
	TSLanIPCIDR string

	// Logging
	Logger *slog.Logger
}

// Config represents a generated sing-box configuration
type Config = config.Config

// Node represents a parsed proxy node
type Node = model.Node

// GenerateConfig is the high-level API for generating configurations.
// It provides a simple interface with sensible defaults while allowing customization.
func GenerateConfig(ctx context.Context, source string, opts ...Option) (*Config, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	generator := NewGenerator(opts...)
	return generator.Generate(ctx, source)
}

// GenerateConfigBytes generates configuration and returns it as bytes in the specified format
func GenerateConfigBytes(ctx context.Context, source string, opts ...Option) ([]byte, error) {
	cfg, err := GenerateConfig(ctx, source, opts...)
	if err != nil {
		return nil, err
	}

	options := buildOptions(opts...)

	var renderer renderer.Renderer
	switch options.Format {
	case "yaml", "yml":
		renderer = registry.YAMLRenderer
	default:
		renderer = registry.JSONRenderer
	}

	return renderer.Render(cfg)
}

// ParseNodes extracts and parses nodes from a source without generating full configuration
func ParseNodes(ctx context.Context, source string, opts ...Option) ([]Node, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	generator := NewGenerator(opts...)
	return generator.ParseNodes(ctx, source)
}

// Generator provides mid-level API with more control over the generation process
type Generator struct {
	options GenerateOptions

	// Context-aware component wrappers
	fetcher     ContextFetcher
	transformer ContextTransformer

	// Internal components (not context-aware but wrapped)
	templateFactory *template.TemplateFactory
}

// NewGenerator creates a new Generator with the specified options
func NewGenerator(opts ...Option) *Generator {
	return newGenerator(buildOptions(opts...))
}

// newGenerator creates a Generator from fully resolved options
func newGenerator(options GenerateOptions) *Generator {
	return &Generator{
		options:         options,
		fetcher:         NewContextFetcher(options),
		transformer:     NewContextTransformer(options),
		templateFactory: template.NewTemplateFactory(),
	}
}

// Generate performs the complete generation process from source to configuration
func (g *Generator) Generate(ctx context.Context, source string) (*Config, error) {
	if source == "" {
		return nil, ErrEmptySource
	}

	// 1. Fetch data
	data, err := g.fetcher.Fetch(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// 2. Parse nodes
	nodes, err := g.parseData(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	if len(nodes) == 0 {
		return nil, ErrNoValidNodes
	}

	// 3. Transform to outbounds
	outbounds, err := g.transformer.Transform(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("transform failed: %w", err)
	}

	// 4. Generate configuration
	tmpl, err := g.templateFactory.CreateTemplate(g.options.TemplateVersion)
	if err != nil {
		return nil, fmt.Errorf("template creation failed: %w", err)
	}

	cfg := tmpl.InjectWithOptions(outbounds, g.options.templateOptions())
	return cfg, nil
}

// ParseNodes extracts and parses nodes from a source
func (g *Generator) ParseNodes(ctx context.Context, source string) ([]Node, error) {
	if source == "" {
		return nil, ErrEmptySource
	}

	data, err := g.fetcher.Fetch(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	return g.parseData(ctx, data)
}

// parseData decodes the subscription once and dispatches each link to its protocol parser
func (g *Generator) parseData(ctx context.Context, data []byte) ([]Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	nodes, format, err := parser.ParseSubscription(data, "", g.options.Logger)
	g.logf("Detected format: %s", format)
	if err != nil || len(nodes) == 0 {
		return nil, ErrNoValidNodes
	}
	g.logf("Parsed %d nodes", len(nodes))
	return nodes, nil
}

// logf logs a message if logger is configured
func (g *Generator) logf(format string, args ...any) {
	if g.options.Logger != nil {
		g.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}

// GetAvailableTemplates returns list of available template versions
func (g *Generator) GetAvailableTemplates() ([]string, error) {
	return g.templateFactory.GetAvailableVersions()
}

// buildOptions creates GenerateOptions from Option functions with defaults
func buildOptions(opts ...Option) GenerateOptions {
	options := GenerateOptions{
		TemplateVersion:    "v1.14",
		Platform:           "linux",
		HTTPTimeout:        30 * time.Second,
		MirrorURL:          "https://ghfast.top",
		DNSLocalServer:     "114.114.114.114",
		Format:             "json",
		RemoveEmoji:        true,
		ExternalController: "127.0.0.1:9095",
	}

	for _, opt := range opts {
		opt(&options)
	}

	return options
}

// templateOptions maps generation options to template injection options.
// This is the single place where GenerateOptions -> TemplateOptions is defined.
func (o GenerateOptions) templateOptions() config.TemplateOptions {
	return config.TemplateOptions{
		MirrorURL:          o.MirrorURL,
		ExternalController: o.ExternalController,
		ClientSubnet:       o.ClientSubnet,
		RemoveEmoji:        o.RemoveEmoji,
		DNSLocalServer:     o.DNSLocalServer,
		Platform:           o.Platform,
		TSAuthKey:          o.TSAuthKey,
		TSLanIPCIDR:        o.TSLanIPCIDR,
		Logger:             o.Logger,
	}
}
