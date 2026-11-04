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
	httpFetcher := fetcher.NewHTTPFetcher()
	httpFetcher.Logger = options.Logger
	if options.HTTPTimeout > 0 {
		httpFetcher.Client.Timeout = options.HTTPTimeout
	}
	return &contextFetcher{
		httpFetcher: httpFetcher,
		fileFetcher: &fetcher.FileFetcher{Logger: options.Logger},
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

	// Prefer native context support so that cancellation actually aborts the request
	if cf, ok := fetcher.(interface {
		FetchContext(context.Context, string) ([]byte, error)
	}); ok {
		data, err := cf.FetchContext(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("fetch failed: %w", err)
		}
		f.logf("Successfully fetched %d bytes", len(data))
		return data, nil
	}

	data, err := fetcher.Fetch(source)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	f.logf("Successfully fetched %d bytes", len(data))
	return data, nil
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
		transformer: &transformer.SingBoxTransformer{Logger: options.Logger},
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

	outbounds, err := t.transformer.Transform(t.withBandwidth(nodes))
	if err != nil {
		return nil, fmt.Errorf("transformation failed: %w", err)
	}
	t.logf("Successfully transformed to %d outbounds", len(outbounds))
	return outbounds, nil
}

// withBandwidth injects configured Hysteria2 bandwidth into nodes that use the
// default (25/300) or no bandwidth. It works on copies so the caller's nodes
// (and their Extra maps) are never mutated.
func (t *contextTransformer) withBandwidth(nodes []model.Node) []model.Node {
	if t.options.UpMbps <= 0 && t.options.DownMbps <= 0 {
		return nodes
	}
	upStr := fmt.Sprintf("%d", t.options.UpMbps)
	downStr := fmt.Sprintf("%d", t.options.DownMbps)

	result := make([]model.Node, len(nodes))
	copy(result, nodes)
	for i, node := range result {
		if node.Type != "hysteria2" {
			continue
		}
		extra := make(map[string]any, len(node.Extra)+2)
		for k, v := range node.Extra {
			extra[k] = v
		}
		// Only override if using default values (25/300) or not set
		if extra["up_mbps"] == "25" || extra["up_mbps"] == nil {
			extra["up_mbps"] = upStr
		}
		if extra["down_mbps"] == "300" || extra["down_mbps"] == nil {
			extra["down_mbps"] = downStr
		}
		result[i].Extra = extra
	}
	return result
}

// logf logs a message if logger is configured
func (t *contextTransformer) logf(format string, args ...any) {
	if t.options.Logger != nil {
		t.options.Logger.Info(fmt.Sprintf(format, args...))
	}
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

	nodes, _, err := parser.ParseSubscription(data, formatHint, p.options.Logger)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}
	p.logf("Successfully parsed %d nodes", len(nodes))
	return nodes, nil
}

// logf logs a message if logger is configured
func (p *contextParser) logf(format string, args ...any) {
	if p.options.Logger != nil {
		p.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}
