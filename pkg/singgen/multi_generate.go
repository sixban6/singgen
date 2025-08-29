package singgen

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/registry"
	"github.com/sixban6/singgen/internal/renderer"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
)

// GenerateConfigFromFile loads configuration from file and generates sing-box config
// If configFile is empty, it will automatically search for configuration files
func GenerateConfigFromFile(ctx context.Context, configFile string, opts ...Option) (*Config, error) {
	multiConfig, err := LoadConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	
	return GenerateConfigFromMulti(ctx, multiConfig, opts...)
}

// GenerateConfigFromMulti generates configuration from MultiConfig
func GenerateConfigFromMulti(ctx context.Context, multiConfig *MultiConfig, opts ...Option) (*Config, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	
	if err := multiConfig.ValidateConfig(); err != nil {
		return nil, err
	}
	
	// Build base options from command line/function options
	baseOptions := buildOptions(opts...)
	
	// Collect all nodes from all subscriptions
	var allOutbounds []transformer.Outbound
	var logger *slog.Logger = baseOptions.Logger
	
	if logger != nil {
		logger.Info("Starting multi-subscription generation", 
			"subscription_count", len(multiConfig.Subscriptions))
	}
	
	for i, subConfig := range multiConfig.Subscriptions {
		if logger != nil {
			logger.Info("Processing subscription", 
				"index", i+1, 
				"name", subConfig.Name, 
				"url", subConfig.URL)
		}
		
		// Merge global and subscription-specific options
		subOptions := multiConfig.MergeSubscriptionOptions(subConfig)
		
		// Override with command line options (highest priority)
		mergeBaseOptions(&subOptions, baseOptions)
		
		// Create generator for this subscription
		generator := NewGenerator(
			WithTemplate(subOptions.TemplateVersion),
			WithPlatform(subOptions.Platform),
			WithHTTPTimeout(subOptions.HTTPTimeout),
			WithMirrorURL(subOptions.MirrorURL),
			WithDNSServer(subOptions.DNSLocalServer),
			WithOutputFormat(subOptions.Format),
			WithEmojiRemoval(subOptions.RemoveEmoji),
			WithExternalController(subOptions.ExternalController),
			WithLogger(logger),
		)
		
		if subOptions.ClientSubnet != "" {
			generator = NewGenerator(
				WithTemplate(subOptions.TemplateVersion),
				WithPlatform(subOptions.Platform),
				WithHTTPTimeout(subOptions.HTTPTimeout),
				WithMirrorURL(subOptions.MirrorURL),
				WithDNSServer(subOptions.DNSLocalServer),
				WithOutputFormat(subOptions.Format),
				WithEmojiRemoval(subOptions.RemoveEmoji),
				WithExternalController(subOptions.ExternalController),
				WithClientSubnet(subOptions.ClientSubnet),
				WithLogger(logger),
			)
		}
		
		// Parse nodes from this subscription
		nodes, err := generator.ParseNodes(ctx, subConfig.URL)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to parse subscription", 
					"name", subConfig.Name, 
					"error", err)
			}
			continue // Skip failed subscription, don't fail entire operation
		}
		
		if len(nodes) == 0 {
			if logger != nil {
				logger.Warn("No nodes found in subscription", "name", subConfig.Name)
			}
			continue
		}
		
		// Process subscription nodes (name prefix, emoji removal, TLS settings)
		processedNodes := processSubscriptionNodes(nodes, SubscriptionProcessOptions{
			NamePrefix:    subConfig.Name,
			RemoveEmoji:   subOptions.RemoveEmoji,
			SkipTLSVerify: subOptions.SkipTLSVerify,
		})
		
		// Transform to outbounds
		outbounds, err := generator.transformer.Transform(ctx, processedNodes)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to transform nodes", 
					"name", subConfig.Name, 
					"error", err)
			}
			continue // Skip failed subscription
		}
		
		if logger != nil {
			logger.Info("Successfully processed subscription", 
				"name", subConfig.Name, 
				"node_count", len(nodes),
				"outbound_count", len(outbounds))
		}
		
		allOutbounds = append(allOutbounds, outbounds...)
	}
	
	if len(allOutbounds) == 0 {
		return nil, ErrNoValidNodes
	}
	
	if logger != nil {
		logger.Info("Merged all subscriptions", 
			"total_outbounds", len(allOutbounds))
	}
	
	// Use the first subscription's generator for template processing
	// (all should have the same global template settings)
	firstSubConfig := multiConfig.Subscriptions[0]
	templateOptions := multiConfig.MergeSubscriptionOptions(firstSubConfig)
	mergeBaseOptions(&templateOptions, baseOptions)
	
	templateGenerator := NewGenerator(
		WithTemplate(templateOptions.TemplateVersion),
		WithPlatform(templateOptions.Platform),
		WithMirrorURL(templateOptions.MirrorURL),
		WithDNSServer(templateOptions.DNSLocalServer),
		WithExternalController(templateOptions.ExternalController),
		WithLogger(logger),
	)
	
	if templateOptions.ClientSubnet != "" {
		templateGenerator = NewGenerator(
			WithTemplate(templateOptions.TemplateVersion),
			WithPlatform(templateOptions.Platform),
			WithMirrorURL(templateOptions.MirrorURL),
			WithDNSServer(templateOptions.DNSLocalServer),
			WithExternalController(templateOptions.ExternalController),
			WithClientSubnet(templateOptions.ClientSubnet),
			WithLogger(logger),
		)
	}
	
	// Generate final configuration
	tmpl, err := templateGenerator.templateFactory.CreateTemplate(templateOptions.TemplateVersion)
	if err != nil {
		return nil, fmt.Errorf("template creation failed: %w", err)
	}
	
	tmplOptions := config.TemplateOptions{
		MirrorURL:          templateOptions.MirrorURL,
		ExternalController: templateOptions.ExternalController,
		ClientSubnet:       templateOptions.ClientSubnet,
		RemoveEmoji:        templateOptions.RemoveEmoji,
		DNSLocalServer:     templateOptions.DNSLocalServer,
		Platform:           templateOptions.Platform,
	}
	
	cfg := tmpl.InjectWithOptions(allOutbounds, tmplOptions)
	
	if logger != nil {
		logger.Info("Multi-subscription configuration generated successfully")
	}
	
	return cfg, nil
}

// GenerateConfigBytesFromFile loads config file and generates configuration bytes
func GenerateConfigBytesFromFile(ctx context.Context, configFile string, opts ...Option) ([]byte, error) {
	multiConfig, err := LoadConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}
	
	return GenerateConfigBytesFromMulti(ctx, multiConfig, opts...)
}

// GenerateConfigBytesFromMulti generates configuration bytes from MultiConfig
func GenerateConfigBytesFromMulti(ctx context.Context, multiConfig *MultiConfig, opts ...Option) ([]byte, error) {
	cfg, err := GenerateConfigFromMulti(ctx, multiConfig, opts...)
	if err != nil {
		return nil, err
	}
	
	options := buildOptions(opts...)
	
	// Use format from multiConfig.Global if not overridden by opts
	format := multiConfig.Global.Format
	if options.Format != "json" { // "json" is the default, so check if it was explicitly set
		format = options.Format
	}
	
	var renderer renderer.Renderer
	switch format {
	case "yaml", "yml":
		renderer = registry.YAMLRenderer
	default:
		renderer = registry.JSONRenderer
	}
	
	return renderer.Render(cfg)
}

// SubscriptionProcessOptions holds options for processing subscription nodes
type SubscriptionProcessOptions struct {
	NamePrefix    string
	RemoveEmoji   bool
	SkipTLSVerify bool
}

// processSubscriptionNodes processes nodes with subscription-specific settings
func processSubscriptionNodes(nodes []model.Node, options SubscriptionProcessOptions) []model.Node {
	if len(nodes) == 0 {
		return nodes
	}
	
	processedNodes := make([]model.Node, len(nodes))
	for i, node := range nodes {
		processedNodes[i] = node
		
		// 1. Process tag name and prefix
		baseTag := getBaseTag(node)
		if options.RemoveEmoji {
			baseTag = removeEmojiFromString(baseTag)
		}
		if options.NamePrefix != "" {
			processedNodes[i].Tag = fmt.Sprintf("[%s] %s", options.NamePrefix, baseTag)
		} else {
			processedNodes[i].Tag = baseTag
		}
		
		// 2. Process TLS verification settings
		if processedNodes[i].Security.TLS && options.SkipTLSVerify {
			processedNodes[i].Security.SkipVerify = true
		}
	}
	
	return processedNodes
}

// getBaseTag returns the base tag for a node
func getBaseTag(node model.Node) string {
	if node.Tag != "" {
		return node.Tag
	}
	return node.Addr
}

// addNamePrefixWithOptions adds name prefix to all nodes and optionally removes emojis
// Deprecated: Use processSubscriptionNodes instead
func addNamePrefixWithOptions(nodes []model.Node, prefix string, removeEmoji bool) []model.Node {
	return processSubscriptionNodes(nodes, SubscriptionProcessOptions{
		NamePrefix:    prefix,
		RemoveEmoji:   removeEmoji,
		SkipTLSVerify: false,
	})
}

// addNamePrefix adds name prefix to all nodes (backward compatibility)
func addNamePrefix(nodes []model.Node, prefix string) []model.Node {
	return addNamePrefixWithOptions(nodes, prefix, false)
}

// removeEmojiFromString removes emoji characters from a string
func removeEmojiFromString(text string) string {
	return util.RemoveEmoji(text)
}

// mergeBaseOptions merges command line options into subscription options
// Command line options have the highest priority
func mergeBaseOptions(subOptions *GenerateOptions, baseOptions GenerateOptions) {
	// Only override if base options were explicitly set (not defaults)
	// This is a simple approach - in practice you might want more sophisticated
	// detection of which options were explicitly set vs defaults
	
	// For template, platform etc., we assume global config takes precedence
	// unless there's a specific command line override mechanism
	
	// Logger always comes from base options
	if baseOptions.Logger != nil {
		subOptions.Logger = baseOptions.Logger
	}
}