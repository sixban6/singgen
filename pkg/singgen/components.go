package singgen

import (
	"context"
	"fmt"
	"net/url"

	"github.com/sixban6/singgen/internal/fetcher"
	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/pkg/model"
)

// ContextFetcher provides context-aware data fetching with automatic source type detection
type ContextFetcher interface {
	Fetch(ctx context.Context, source string) ([]byte, error)
}

// contextFetcher wraps the internal fetchers with context support
type contextFetcher struct {
	httpFetcher fetcher.Fetcher
	fileFetcher fetcher.Fetcher
	options     GenerateOptions
}

// NewContextFetcher creates a new context-aware fetcher
func NewContextFetcher(options GenerateOptions) ContextFetcher {
	return &contextFetcher{
		httpFetcher: fetcher.NewHTTPFetcher(),
		fileFetcher: fetcher.NewFileFetcher(),
		options:     options,
	}
}

// Fetch retrieves data from the source with context support and automatic type detection
func (f *contextFetcher) Fetch(ctx context.Context, source string) ([]byte, error) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Determine source type and select appropriate fetcher
	var fetcher fetcher.Fetcher
	if f.isURL(source) {
		fetcher = f.httpFetcher
		f.logf("Fetching from URL: %s", source)
	} else {
		fetcher = f.fileFetcher
		f.logf("Reading from file: %s", source)
	}
	
	// Create a channel to handle the result
	resultCh := make(chan fetchResult, 1)
	
	// Start fetching in a goroutine
	go func() {
		data, err := fetcher.Fetch(source)
		resultCh <- fetchResult{data: data, err: err}
	}()
	
	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("fetch operation canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("fetch failed: %w", result.err)
		}
		f.logf("Successfully fetched %d bytes", len(result.data))
		return result.data, nil
	}
}

// isURL determines if the source is a URL or file path
func (f *contextFetcher) isURL(source string) bool {
	u, err := url.Parse(source)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// logf logs a message if logger is configured
func (f *contextFetcher) logf(format string, args ...any) {
	if f.options.Logger != nil {
		f.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}

// fetchResult holds the result of a fetch operation
type fetchResult struct {
	data []byte
	err  error
}

// ContextTransformer provides context-aware node transformation
type ContextTransformer interface {
	Transform(ctx context.Context, nodes []model.Node) ([]transformer.Outbound, error)
}

// contextTransformer wraps the internal transformer with context support
type contextTransformer struct {
	transformer transformer.Transformer
	options     GenerateOptions
}

// NewContextTransformer creates a new context-aware transformer
func NewContextTransformer(options GenerateOptions) ContextTransformer {
	return &contextTransformer{
		transformer: transformer.NewSingBoxTransformer(),
		options:     options,
	}
}

// Transform converts nodes to outbounds with context support
func (t *contextTransformer) Transform(ctx context.Context, nodes []model.Node) ([]transformer.Outbound, error) {
	// Check context cancellation before starting
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("transform operation canceled: %w", ctx.Err())
	default:
	}
	
	if len(nodes) == 0 {
		return []transformer.Outbound{}, nil
	}
	
	t.logf("Transforming %d nodes to outbounds", len(nodes))
	
	// Create a channel for the result
	resultCh := make(chan transformResult, 1)
	
	// Start transformation in a goroutine
	go func() {
		outbounds, err := t.transformer.Transform(nodes)
		resultCh <- transformResult{outbounds: outbounds, err: err}
	}()
	
	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("transform operation canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("transformation failed: %w", result.err)
		}
		t.logf("Successfully transformed to %d outbounds", len(result.outbounds))
		return result.outbounds, nil
	}
}

// logf logs a message if logger is configured
func (t *contextTransformer) logf(format string, args ...any) {
	if t.options.Logger != nil {
		t.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}

// transformResult holds the result of a transform operation
type transformResult struct {
	outbounds []transformer.Outbound
	err       error
}

// ContextParser provides context-aware parsing with smart format detection
type ContextParser interface {
	DetectFormat(ctx context.Context, data []byte) string
	Parse(ctx context.Context, data []byte) ([]model.Node, error)
	ParseWithHint(ctx context.Context, data []byte, formatHint string) ([]model.Node, error)
}

// contextParser wraps parser functionality with context support
type contextParser struct {
	options GenerateOptions
}

// NewContextParser creates a new context-aware parser
func NewContextParser(options GenerateOptions) ContextParser {
	return &contextParser{
		options: options,
	}
}

// DetectFormat detects the format of the input data with context support
func (p *contextParser) DetectFormat(ctx context.Context, data []byte) string {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "unknown"
	default:
	}
	
	// Use the internal parser's detection logic
	// This is fast and doesn't need special context handling
	return parser.DetectFormat(data)
}

// Parse parses the data with automatic format detection
func (p *contextParser) Parse(ctx context.Context, data []byte) ([]model.Node, error) {
	return p.ParseWithHint(ctx, data, "")
}

// ParseWithHint parses the data with an optional format hint
func (p *contextParser) ParseWithHint(ctx context.Context, data []byte, formatHint string) ([]model.Node, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("parse operation canceled: %w", ctx.Err())
	default:
	}
	
	format := formatHint
	if format == "" {
		format = p.DetectFormat(ctx, data)
	}
	
	p.logf("Parsing data with format: %s", format)
	
	// The actual parsing is typically fast and doesn't need context handling,
	// but we wrap it for consistency and future extensibility
	resultCh := make(chan parseResult, 1)
	
	go func() {
		var nodes []model.Node
		var err error
		
		if factory, exists := parser.Registry[format]; exists {
			p := factory()
			if p.Accept("", data) {
				nodes, err = p.Parse(data)
			} else {
				err = fmt.Errorf("parser %s rejected the data", format)
			}
		} else {
			err = fmt.Errorf("no parser available for format: %s", format)
		}
		
		resultCh <- parseResult{nodes: nodes, err: err}
	}()
	
	// Wait for either completion or context cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("parse operation canceled: %w", ctx.Err())
	case result := <-resultCh:
		if result.err != nil {
			return nil, fmt.Errorf("parsing failed: %w", result.err)
		}
		p.logf("Successfully parsed %d nodes", len(result.nodes))
		return result.nodes, nil
	}
}

// logf logs a message if logger is configured
func (p *contextParser) logf(format string, args ...any) {
	if p.options.Logger != nil {
		p.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}

// parseResult holds the result of a parse operation
type parseResult struct {
	nodes []model.Node
	err   error
}