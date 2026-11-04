package singgen

import (
	"context"
	"fmt"
	"github.com/sixban6/singgen/internal/logx"
	"log/slog"

	"github.com/sixban6/singgen/internal/registry"
	"github.com/sixban6/singgen/internal/renderer"
	"github.com/sixban6/singgen/internal/template"
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
				"url", logx.RedactSource(subConfig.URL))
		}

		// Merge global and subscription-specific options
		subOptions := multiConfig.MergeSubscriptionOptions(subConfig)

		// Override with command line options (highest priority)
		mergeBaseOptions(&subOptions, baseOptions)

		// Create generator for this subscription
		generator := newGenerator(subOptions)

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

	// Generate final configuration
	tmpl, err := template.NewTemplateFactory().CreateTemplate(templateOptions.TemplateVersion)
	if err != nil {
		return nil, fmt.Errorf("template creation failed: %w", err)
	}

	cfg := tmpl.InjectWithOptions(allOutbounds, templateOptions.templateOptions())

	if logger != nil {
		logger.Info("Multi-subscription configuration generated successfully")
	}

	return cfg, nil
}

// 用于从config.yaml直接读取配置生成 sb配置文件
func GenerateConfigBytesFromFile(ctx context.Context, configFile string, opts ...Option) ([]byte, error) {
	multiConfig, err := LoadConfigFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	return GenerateConfigBytesFromMulti(ctx, multiConfig, opts...)
}

// 最底层的方法，多订阅源生成配置
func GenerateConfigBytesFromMulti(ctx context.Context, multiConfig *MultiConfig, opts ...Option) ([]byte, error) {
	cfg, err := GenerateConfigFromMulti(ctx, multiConfig, opts...)
	if err != nil {
		return nil, err
	}

	options := buildOptions(opts...)

	// Use format from multiConfig.Global unless explicitly overridden by opts
	format := multiConfig.Global.Format
	if options.isSet(optFormat) {
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

// mergeBaseOptions applies options that were explicitly passed by the caller
// (CLI flags or Option functions) on top of the configuration file.
// Priority: explicit options > subscription-specific settings > global settings.
// Options left at their library defaults never override the configuration file.
func mergeBaseOptions(subOptions *GenerateOptions, baseOptions GenerateOptions) {
	if baseOptions.Logger != nil {
		subOptions.Logger = baseOptions.Logger
	}
	if baseOptions.isSet(optTemplate) {
		subOptions.TemplateVersion = baseOptions.TemplateVersion
	}
	if baseOptions.isSet(optPlatform) {
		subOptions.Platform = baseOptions.Platform
	}
	if baseOptions.isSet(optHTTPTimeout) {
		subOptions.HTTPTimeout = baseOptions.HTTPTimeout
	}
	if baseOptions.isSet(optMirrorURL) {
		subOptions.MirrorURL = baseOptions.MirrorURL
	}
	if baseOptions.isSet(optDNSServer) {
		subOptions.DNSLocalServer = baseOptions.DNSLocalServer
	}
	if baseOptions.isSet(optClientSubnet) {
		subOptions.ClientSubnet = baseOptions.ClientSubnet
	}
	if baseOptions.isSet(optFormat) {
		subOptions.Format = baseOptions.Format
	}
	if baseOptions.isSet(optRemoveEmoji) {
		subOptions.RemoveEmoji = baseOptions.RemoveEmoji
	}
	if baseOptions.isSet(optBandwidth) {
		subOptions.UpMbps = baseOptions.UpMbps
		subOptions.DownMbps = baseOptions.DownMbps
	}
	if baseOptions.isSet(optExternalController) {
		subOptions.ExternalController = baseOptions.ExternalController
	}
	if baseOptions.isSet(optTSAuthKey) {
		subOptions.TSAuthKey = baseOptions.TSAuthKey
	}
	if baseOptions.isSet(optTSLanIPCIDR) {
		subOptions.TSLanIPCIDR = baseOptions.TSLanIPCIDR
	}
}
