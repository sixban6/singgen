package singgen

import (
	"context"
	"fmt"
	"strings"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/renderer"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/pkg/model"
)

// Pipeline provides the low-level API for maximum control over the generation process.
// It allows custom components to be injected and provides fine-grained control over each step.
type Pipeline struct {
	fetcher     ContextFetcher
	parser      ContextParser
	transformer ContextTransformer
	template    template.Template
	renderer    renderer.Renderer
	options     GenerateOptions
}

// PipelineOption configures a Pipeline
type PipelineOption func(*Pipeline)

// PipelineResult contains the complete result of a pipeline execution
type PipelineResult struct {
	Config    *config.Config        `json:"config"`
	Nodes     []model.Node          `json:"nodes"`
	Outbounds []transformer.Outbound `json:"outbounds"`
	Metadata  PipelineMetadata      `json:"metadata"`
}

// PipelineMetadata contains information about the pipeline execution
type PipelineMetadata struct {
	SourceType       string `json:"source_type"`
	DetectedFormat   string `json:"detected_format"`
	NodesCount       int    `json:"nodes_count"`
	OutboundsCount   int    `json:"outbounds_count"`
	TemplateVersion  string `json:"template_version"`
	Platform         string `json:"platform"`
}

// NewPipeline creates a new Pipeline with the specified options
func NewPipeline(opts ...PipelineOption) *Pipeline {
	// Start with default options
	options := buildOptions()
	
	p := &Pipeline{
		options: options,
	}
	
	// Set default components
	p.fetcher = NewContextFetcher(options)
	p.parser = NewContextParser(options) 
	p.transformer = NewContextTransformer(options)
	p.renderer = renderer.NewJSONRenderer()
	
	// Apply custom options
	for _, opt := range opts {
		opt(p)
	}
	
	return p
}

// Execute runs the complete pipeline from source to final output
func (p *Pipeline) Execute(ctx context.Context, source string) (*PipelineResult, error) {
	if source == "" {
		return nil, ErrEmptySource
	}
	
	result := &PipelineResult{
		Metadata: PipelineMetadata{
			TemplateVersion: p.options.TemplateVersion,
			Platform:        p.options.Platform,
		},
	}
	
	p.logf("Starting pipeline execution for source: %s", source)
	
	// Step 1: Fetch data
	data, err := p.fetcher.Fetch(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("fetch step failed: %w", err)
	}
	
	result.Metadata.SourceType = p.getSourceType(source)
	
	// Step 2: Detect format
	format := p.parser.DetectFormat(ctx, data)
	result.Metadata.DetectedFormat = format
	p.logf("Detected format: %s", format)
	
	// Step 3: Parse nodes
	nodes, err := p.parser.Parse(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("parse step failed: %w", err)
	}
	
	if len(nodes) == 0 {
		return nil, ErrNoValidNodes
	}
	
	result.Nodes = nodes
	result.Metadata.NodesCount = len(nodes)
	p.logf("Parsed %d nodes", len(nodes))
	
	// Step 4: Transform to outbounds
	outbounds, err := p.transformer.Transform(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("transform step failed: %w", err)
	}
	
	result.Outbounds = outbounds
	result.Metadata.OutboundsCount = len(outbounds)
	p.logf("Transformed to %d outbounds", len(outbounds))
	
	// Step 5: Generate configuration
	if p.template == nil {
		factory := template.NewTemplateFactory()
		p.template, err = factory.CreateTemplate(p.options.TemplateVersion)
		if err != nil {
			return nil, fmt.Errorf("template creation failed: %w", err)
		}
	}
	
	templateOptions := config.TemplateOptions{
		MirrorURL:          p.options.MirrorURL,
		ExternalController: p.options.ExternalController,
		ClientSubnet:       p.options.ClientSubnet,
		RemoveEmoji:        p.options.RemoveEmoji,
		DNSLocalServer:     p.options.DNSLocalServer,
		Platform:           p.options.Platform,
	}
	
	cfg := p.template.InjectWithOptions(outbounds, templateOptions)
	result.Config = cfg
	
	p.logf("Pipeline execution completed successfully")
	return result, nil
}

// ExecuteBytes runs the pipeline and returns the output as bytes
func (p *Pipeline) ExecuteBytes(ctx context.Context, source string) ([]byte, error) {
	result, err := p.Execute(ctx, source)
	if err != nil {
		return nil, err
	}
	
	return p.renderer.Render(result.Config)
}

// FetchOnly performs only the fetch step
func (p *Pipeline) FetchOnly(ctx context.Context, source string) ([]byte, error) {
	return p.fetcher.Fetch(ctx, source)
}

// ParseOnly performs fetch and parse steps
func (p *Pipeline) ParseOnly(ctx context.Context, source string) ([]model.Node, error) {
	data, err := p.fetcher.Fetch(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	
	return p.parser.Parse(ctx, data)
}

// TransformOnly performs fetch, parse, and transform steps
func (p *Pipeline) TransformOnly(ctx context.Context, source string) ([]transformer.Outbound, error) {
	nodes, err := p.ParseOnly(ctx, source)
	if err != nil {
		return nil, err
	}
	
	return p.transformer.Transform(ctx, nodes)
}

// getSourceType determines the source type for metadata
func (p *Pipeline) getSourceType(source string) string {
	// Try to cast to contextFetcher to use isURL method
	if cf, ok := p.fetcher.(*contextFetcher); ok {
		if cf.isURL(source) {
			return "url"
		}
		return "file"
	}
	
	// Fallback: simple URL detection for non-contextFetcher implementations
	if strings.Contains(source, "://") {
		return "url"
	}
	return "file"
}

// logf logs a message if logger is configured
func (p *Pipeline) logf(format string, args ...any) {
	if p.options.Logger != nil {
		p.options.Logger.Info(fmt.Sprintf(format, args...))
	}
}

// Pipeline configuration options

// WithCustomFetcher sets a custom fetcher for the pipeline
func WithCustomFetcher(fetcher ContextFetcher) PipelineOption {
	return func(p *Pipeline) {
		p.fetcher = fetcher
	}
}

// WithCustomParser sets a custom parser for the pipeline
func WithCustomParser(parser ContextParser) PipelineOption {
	return func(p *Pipeline) {
		p.parser = parser
	}
}

// WithCustomTransformer sets a custom transformer for the pipeline
func WithCustomTransformer(transformer ContextTransformer) PipelineOption {
	return func(p *Pipeline) {
		p.transformer = transformer
	}
}

// WithCustomTemplate sets a custom template for the pipeline
func WithCustomTemplate(template template.Template) PipelineOption {
	return func(p *Pipeline) {
		p.template = template
	}
}

// WithCustomRenderer sets a custom renderer for the pipeline
func WithCustomRenderer(renderer renderer.Renderer) PipelineOption {
	return func(p *Pipeline) {
		p.renderer = renderer
	}
}

// WithPipelineOptions applies GenerateOptions to the pipeline
func WithPipelineOptions(opts ...Option) PipelineOption {
	return func(p *Pipeline) {
		p.options = buildOptions(opts...)
		
		// Update components with new options
		p.fetcher = NewContextFetcher(p.options)
		p.parser = NewContextParser(p.options)
		p.transformer = NewContextTransformer(p.options)
	}
}

// StreamingPipeline provides streaming capabilities for processing large subscription files
type StreamingPipeline struct {
	pipeline *Pipeline
}

// NewStreamingPipeline creates a new streaming pipeline
func NewStreamingPipeline(opts ...PipelineOption) *StreamingPipeline {
	return &StreamingPipeline{
		pipeline: NewPipeline(opts...),
	}
}

// StreamNodes processes nodes in batches and streams results
func (sp *StreamingPipeline) StreamNodes(ctx context.Context, source string, batchSize int) (<-chan []model.Node, <-chan error) {
	nodesCh := make(chan []model.Node, 10)
	errCh := make(chan error, 1)
	
	go func() {
		defer close(nodesCh)
		defer close(errCh)
		
		// For now, this is a simple implementation
		// In a real streaming implementation, we would parse and yield nodes incrementally
		nodes, err := sp.pipeline.ParseOnly(ctx, source)
		if err != nil {
			errCh <- err
			return
		}
		
		// Split nodes into batches
		for i := 0; i < len(nodes); i += batchSize {
			end := i + batchSize
			if end > len(nodes) {
				end = len(nodes)
			}
			
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case nodesCh <- nodes[i:end]:
			}
		}
	}()
	
	return nodesCh, errCh
}