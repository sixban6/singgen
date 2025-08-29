package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/internal/version"
	"github.com/sixban6/singgen/pkg/singgen"
)

func main() {
	var (
		// Single subscription mode (legacy)
		subURL             = flag.String("url", "", "subscription URL or file path")
		
		// Multi-subscription mode (new)
		configFile         = flag.String("config", "", "multi-subscription config file path (yaml/json)")
		generateExample    = flag.Bool("generate-example", false, "generate example configuration file")
		
		// Output options
		outFile            = flag.String("out", "config.json", "output file path")
		format             = flag.String("format", "json", "output format (json, yaml)")
		
		// Global options (can override config file)
		mirrorURL          = flag.String("mirror", "https://ghfast.top", "mirror URL for downloading rule sets")
		logLevel           = flag.String("log", "warn", "log level (debug, info, warn, error)")
		templateVer        = flag.String("template", "v1.12", "sing-box template version (v1.12, v1.13, etc.)")
		externalController = flag.String("external-controller", "127.0.0.1:9095", "external controller address for Clash API")
		clientSubnet       = flag.String("subnet", "", "client subnet for DNS queries (e.g., 202.101.170.1/24)")
		removeEmoji        = flag.Bool("emoji", true, "remove emoji characters from node tags")
		dnsLocalServer     = flag.String("dns", "114.114.114.114", "DNS local server address")
		platform           = flag.String("platform", "linux", "target platform (linux, darwin, ios)")
		
		// Utility options
		listTemplate       = flag.Bool("list-templates", false, "list available template versions")
		showVersion        = flag.Bool("version", false, "show version information")
		showFullVersion    = flag.Bool("version-full", false, "show detailed version information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetVersion())
		return
	}

	if *showFullVersion {
		fmt.Println(version.GetFullVersion())
		return
	}

	// Initialize logger for legacy compatibility
	if err := util.InitLogger(*logLevel); err != nil {
		log.Fatal("Failed to init logger:", err)
	}
	defer util.Sync()

	// Create structured logger for the library
	var logHandler slog.Handler
	switch *logLevel {
	case "debug":
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	case "info":
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	case "warn":
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})
	case "error":
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError})
	default:
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})
	}
	logger := slog.New(logHandler)

	// Start timing
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		fmt.Printf("Total execution time: %v\n", duration)
		logger.Info("Configuration generation completed", "duration", duration)
	}()

	if *listTemplate {
		generator := singgen.NewGenerator()
		versions, err := generator.GetAvailableTemplates()
		if err != nil {
			log.Fatal("Failed to list templates:", err)
		}
		fmt.Println("Available template versions:")
		for _, version := range versions {
			fmt.Printf("  - %s\n", version)
		}
		return
	}

	if *generateExample {
		example := singgen.GenerateExampleConfig()
		format := "yaml"
		if strings.HasSuffix(*outFile, ".json") {
			format = "json"
		}
		
		if err := singgen.SaveConfigFile(example, *outFile, format); err != nil {
			log.Fatal("Failed to generate example config:", err)
		}
		fmt.Printf("Example configuration generated: %s\n", *outFile)
		return
	}

	// Determine operation mode
	if *configFile != "" {
		// Multi-subscription mode - config file driven
		if err := runMultiSubscriptionMode(logger, *configFile, *outFile); err != nil {
			logger.Error("Multi-subscription mode failed", "error", err)
			os.Exit(1)
		}
	} else if *subURL != "" {
		// Single subscription mode (legacy)
		if err := runSingleSubscriptionMode(logger, *subURL, *outFile, *templateVer, *platform, *mirrorURL, *dnsLocalServer, *externalController, *clientSubnet, *format, *removeEmoji); err != nil {
			logger.Error("Single subscription mode failed", "error", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Error: Either -url (single subscription) or -config (multi subscription) must be specified")
		fmt.Println()
		flag.Usage()
		os.Exit(1)
	}

	logger.Info("Configuration generation completed successfully", "output", *outFile)
	fmt.Printf("Configuration generated: %s\n", *outFile)
}

// runMultiSubscriptionMode handles multi-subscription configuration generation
func runMultiSubscriptionMode(logger *slog.Logger, configFile string, outFile string) error {
	logger.Info("Running in multi-subscription mode", "config_file", configFile)
	
	// Load and generate - everything comes from config file
	ctx := context.Background()
	data, err := singgen.GenerateConfigBytesFromFile(ctx, configFile, singgen.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}
	
	// Write output
	if err := writeOutput(outFile, data); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	
	return nil
}

// runSingleSubscriptionMode handles single subscription configuration generation (legacy)
func runSingleSubscriptionMode(logger *slog.Logger, subURL, outFile, templateVer, platform, mirrorURL, dnsLocalServer, externalController, clientSubnet, format string, removeEmoji bool) error {
	logger.Info("Running in single subscription mode", "url", subURL)
	
	// Validate inputs
	if err := validateInputs(dnsLocalServer, platform, logger); err != nil {
		return err
	}
	
	// Build options for the library
	opts := []singgen.Option{
		singgen.WithTemplate(templateVer),
		singgen.WithPlatform(platform),
		singgen.WithMirrorURL(mirrorURL),
		singgen.WithDNSServer(dnsLocalServer),
		singgen.WithOutputFormat(format),
		singgen.WithEmojiRemoval(removeEmoji),
		singgen.WithExternalController(externalController),
		singgen.WithLogger(logger),
	}
	
	if clientSubnet != "" {
		opts = append(opts, singgen.WithClientSubnet(clientSubnet))
	}
	
	// Generate configuration using the library
	ctx := context.Background()
	data, err := singgen.GenerateConfigBytes(ctx, subURL, opts...)
	if err != nil {
		return fmt.Errorf("failed to generate configuration: %w", err)
	}
	
	// Write output
	if err := writeOutput(outFile, data); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	
	return nil
}

// validateInputs validates common input parameters
func validateInputs(dnsLocalServer, platform string, logger *slog.Logger) error {
	// Validate DNS server address
	if dnsLocalServer != "" && !util.ValidateDNSServer(dnsLocalServer) {
		return fmt.Errorf("invalid DNS server address: %s", dnsLocalServer)
	}
	
	// Validate platform
	validPlatforms := []string{"linux", "darwin", "ios"}
	isValidPlatform := false
	for _, p := range validPlatforms {
		if platform == p {
			isValidPlatform = true
			break
		}
	}
	if !isValidPlatform {
		return fmt.Errorf("invalid platform: %s, valid platforms: %v", platform, validPlatforms)
	}
	
	return nil
}

func writeOutput(outFile string, data []byte) error {
	dir := filepath.Dir(outFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory failed: %w", err)
		}
	}

	if err := util.WriteFile(outFile, data); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}
