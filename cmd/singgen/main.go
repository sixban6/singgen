package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/internal/version"
	"github.com/sixban6/singgen/pkg/singgen"
)

func main() {
	var (
		subURL             = flag.String("url", "", "subscription URL or file path")
		outFile            = flag.String("out", "config.json", "output file path")
		format             = flag.String("format", "json", "output format (json, yaml)")
		mirrorURL          = flag.String("mirror", "https://ghfast.top", "mirror URL for downloading rule sets")
		logLevel           = flag.String("log", "warn", "log level (debug, info, warn, error)")
		templateVer        = flag.String("template", "v1.12", "sing-box template version (v1.12, v1.13, etc.)")
		listTemplate       = flag.Bool("list-templates", false, "list available template versions")
		externalController = flag.String("external-controller", "127.0.0.1:9095", "external controller address for Clash API")
		clientSubnet       = flag.String("subnet", "", "client subnet for DNS queries (e.g., 202.101.170.1/24)")
		removeEmoji        = flag.Bool("emoji", true, "remove emoji characters from node tags")
		dnsLocalServer     = flag.String("dns", "114.114.114.114", "DNS local server address")
		platform           = flag.String("platform", "linux", "target platform (linux, darwin, ios)")
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

	if *subURL == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Validate DNS server address
	if *dnsLocalServer != "" && !util.ValidateDNSServer(*dnsLocalServer) {
		logger.Error("Invalid DNS server address", "dns_server", *dnsLocalServer)
		os.Exit(1)
	}

	// Validate platform
	validPlatforms := []string{"linux", "darwin", "ios"}
	isValidPlatform := false
	for _, p := range validPlatforms {
		if *platform == p {
			isValidPlatform = true
			break
		}
	}
	if !isValidPlatform {
		logger.Error("Invalid platform", "platform", *platform, "valid_platforms", validPlatforms)
		os.Exit(1)
	}

	logger.Info("Starting singgen",
		"url", *subURL,
		"output", *outFile,
		"format", *format,
		"template", *templateVer)

	// Build options for the library
	opts := []singgen.Option{
		singgen.WithTemplate(*templateVer),
		singgen.WithPlatform(*platform),
		singgen.WithMirrorURL(*mirrorURL),
		singgen.WithDNSServer(*dnsLocalServer),
		singgen.WithOutputFormat(*format),
		singgen.WithEmojiRemoval(*removeEmoji),
		singgen.WithExternalController(*externalController),
		singgen.WithLogger(logger),
	}

	// Set client subnet if provided
	if *clientSubnet != "" {
		opts = append(opts, singgen.WithClientSubnet(*clientSubnet))
	}

	// For Linux platform, use user provided external_controller
	// For macOS and iOS, platform adapters will set defaults
	if *platform == "linux" {
		opts = append(opts, singgen.WithExternalController(*externalController))
	}

	// Generate configuration using the library
	ctx := context.Background()
	data, err := singgen.GenerateConfigBytes(ctx, *subURL, opts...)
	if err != nil {
		logger.Error("Failed to generate configuration", "error", err)
		os.Exit(1)
	}

	// Write output
	if err := writeOutput(*outFile, data); err != nil {
		logger.Error("Failed to write output", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration generated successfully", "output", *outFile)
	fmt.Printf("Configuration generated: %s\n", *outFile)
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
