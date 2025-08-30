package singgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sixban6/singgen/internal/util"
	"gopkg.in/yaml.v3"
)

// ConfigPaths defines the search order for configuration files
var ConfigPaths = []string{
	"./singgen.yaml",                    // Current directory
	"./singgen.json",
	"~/.config/singgen/config.yaml",     // User config directory
	"~/.config/singgen/config.json",
	"/etc/singgen/config.yaml",          // System config directory
	"/etc/singgen/config.json",
}

// LoadConfigFile loads configuration from a specific file path
func LoadConfigFile(configPath string) (*MultiConfig, error) {
	if configPath == "" {
		return LoadConfigAuto()
	}
	
	// Expand home directory if needed
	expandedPath := expandPath(configPath)
	
	if !fileExists(expandedPath) {
		return nil, fmt.Errorf("%w: %s", ErrConfigFileNotFound, expandedPath)
	}
	
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	return parseConfigData(data, expandedPath)
}

// LoadConfigAuto automatically searches for configuration files in predefined paths
func LoadConfigAuto() (*MultiConfig, error) {
	for _, configPath := range ConfigPaths {
		expandedPath := expandPath(configPath)
		if fileExists(expandedPath) {
			data, err := os.ReadFile(expandedPath)
			if err != nil {
				continue // Try next path
			}
			
			config, err := parseConfigData(data, expandedPath)
			if err != nil {
				continue // Try next path
			}
			
			return config, nil
		}
	}
	
	return nil, ErrConfigFileNotFound
}

// parseConfigData parses configuration data based on file extension
func parseConfigData(data []byte, filePath string) (*MultiConfig, error) {
	config := GetDefaultMultiConfig()
	
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidConfigFormat, err)
		}
	case ".json":
		if err := util.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidConfigFormat, err)
		}
	default:
		// Try to auto-detect format
		// First try YAML
		if err := yaml.Unmarshal(data, config); err == nil {
			return config, config.ValidateConfig()
		}
		
		// Reset config and try JSON
		config = GetDefaultMultiConfig()
		if err := util.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("%w: unable to parse as YAML or JSON", ErrInvalidConfigFormat)
		}
	}
	
	// Validate the loaded configuration
	if err := config.ValidateConfig(); err != nil {
		return nil, err
	}
	
	return config, nil
}

// SaveConfigFile saves configuration to a file
func SaveConfigFile(config *MultiConfig, configPath string, format string) error {
	var data []byte
	var err error
	
	switch strings.ToLower(format) {
	case "yaml", "yml":
		data, err = yaml.Marshal(config)
	case "json":
		data, err = util.MarshalIndent(config)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return original if can't get home dir
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GenerateExampleConfig creates an example configuration file
func GenerateExampleConfig() *MultiConfig {
	example := GetDefaultMultiConfig()
	
	example.Subscriptions = []SubscriptionConfig{
		{
			Name: "provider1",
			URL:  "https://example1.com/subscription",
			RemoveEmoji: &[]bool{false}[0],
			SkipTLSVerify: &[]bool{true}[0],
		},
		{
			Name: "provider2", 
			URL:  "https://example2.com/subscription",
			// Uses global defaults for RemoveEmoji and SkipTLSVerify
		},
	}
	
	return example
}