package singgen

import (
	"context"
	"fmt"
	"github.com/sixban6/singgen/internal/logx"
	"log/slog"
	"sync"

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

	// Fetch and parse all subscriptions concurrently (bounded), then process
	// the results in configuration order so the output stays deterministic.
	fetched := fetchSubscriptions(ctx, multiConfig, baseOptions)

	for i, subConfig := range multiConfig.Subscriptions {
		if logger != nil {
			logger.Info("Processing subscription",
				"index", i+1,
				"name", subConfig.Name,
				"url", logx.RedactSource(subConfig.URL))
		}

		subOptions := fetched[i].options
		generator := fetched[i].generator
		nodes, err := fetched[i].nodes, fetched[i].err
		if err != nil {
			if logger != nil {
				logger.Error("Failed to parse subscription",
					"name", subConfig.Name,
					"error", logx.ErrSource(err))
			}
			continue // Skip failed subscription, don't fail entire operation
		}

		if len(nodes) == 0 {
			if logger != nil {
				logger.Warn("No nodes found in subscription", "name", subConfig.Name)
			}
			continue
		}

		// Apply include/exclude/rename rules (patterns validated in ValidateConfig)
		rules, _ := multiConfig.nodeRulesFor(subConfig)
		nodes, dropped := rules.apply(nodes)
		if dropped > 0 && logger != nil {
			logger.Info("Node rules applied", "name", subConfig.Name, "dropped", dropped, "kept", len(nodes))
		}
		if len(nodes) == 0 {
			if logger != nil {
				logger.Warn("All nodes were filtered out by node rules", "name", subConfig.Name)
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

	if multiConfig.Global.DedupeByServer {
		var removed int
		allOutbounds, removed = dedupeOutboundsByServer(allOutbounds)
		if logger != nil && removed > 0 {
			logger.Info("Removed duplicate nodes across subscriptions", "removed", removed)
		}
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
	if baseOptions.isSet(optCacheDir) {
		subOptions.CacheDir = baseOptions.CacheDir
	}
}

// maxConcurrentFetches bounds how many subscriptions are fetched at the same time
const maxConcurrentFetches = 4

type fetchedSubscription struct {
	options   GenerateOptions
	generator *Generator
	nodes     []model.Node
	err       error
}

// fetchSubscriptions fetches and parses every subscription with bounded concurrency.
// Results are indexed like multiConfig.Subscriptions.
func fetchSubscriptions(ctx context.Context, multiConfig *MultiConfig, baseOptions GenerateOptions) []fetchedSubscription {
	results := make([]fetchedSubscription, len(multiConfig.Subscriptions))
	sem := make(chan struct{}, maxConcurrentFetches)
	var wg sync.WaitGroup

	for i, subConfig := range multiConfig.Subscriptions {
		// Merge global and subscription-specific options, then explicit caller options
		subOptions := multiConfig.MergeSubscriptionOptions(subConfig)
		mergeBaseOptions(&subOptions, baseOptions)
		results[i] = fetchedSubscription{options: subOptions, generator: newGenerator(subOptions)}

		wg.Add(1)
		go func(i int, url string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i].nodes, results[i].err = results[i].generator.ParseNodes(ctx, url)
		}(i, subConfig.URL)
	}
	wg.Wait()
	return results
}
