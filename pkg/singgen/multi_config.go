package singgen

import (
	"time"
)

// SubscriptionConfig represents configuration for a single subscription
type SubscriptionConfig struct {
	URL           string         `json:"url" yaml:"url"`
	Name          string         `json:"name" yaml:"name"`                                         // Used as config prefix
	RemoveEmoji   *bool          `json:"remove_emoji,omitempty" yaml:"remove_emoji,omitempty"`
	SkipTLSVerify *bool          `json:"skip_tls_verify,omitempty" yaml:"skip_tls_verify,omitempty"`
	HTTPTimeout   *time.Duration `json:"http_timeout,omitempty" yaml:"http_timeout,omitempty"`
}

// GlobalConfig represents global default configuration
type GlobalConfig struct {
	Template         string `json:"template" yaml:"template"`
	Platform         string `json:"platform" yaml:"platform"`
	MirrorURL        string `json:"mirror_url" yaml:"mirror_url"`
	DNSLocalServer   string `json:"dns_server" yaml:"dns_server"`
	WebUIAddress     string `json:"webui_address" yaml:"webui_address"`
	RemoveEmoji      bool   `json:"remove_emoji" yaml:"remove_emoji"`
	SkipTLSVerify    bool   `json:"skip_tls_verify" yaml:"skip_tls_verify"`
	HTTPTimeout      time.Duration `json:"http_timeout" yaml:"http_timeout"`
	Format           string `json:"format" yaml:"format"`
	ClientSubnet     string `json:"client_subnet,omitempty" yaml:"client_subnet,omitempty"`
}

// MultiConfig represents configuration supporting multiple subscriptions
type MultiConfig struct {
	// Global default configuration
	Global GlobalConfig `json:"global" yaml:"global"`
	
	// Multiple subscription configurations
	Subscriptions []SubscriptionConfig `json:"subscriptions" yaml:"subscriptions"`
}

// GetDefaultMultiConfig returns a MultiConfig with sensible defaults
func GetDefaultMultiConfig() *MultiConfig {
	return &MultiConfig{
		Global: GlobalConfig{
			Template:       "v1.12",
			Platform:       "linux",
			MirrorURL:      "https://ghfast.top",
			DNSLocalServer: "114.114.114.114",
			WebUIAddress:   "127.0.0.1:9095",
			RemoveEmoji:    true,
			SkipTLSVerify:  false,
			HTTPTimeout:    30 * time.Second,
			Format:         "json",
		},
		Subscriptions: []SubscriptionConfig{},
	}
}

// MergeSubscriptionOptions merges global config with subscription-specific config
// Subscription-specific settings override global settings
func (mc *MultiConfig) MergeSubscriptionOptions(subConfig SubscriptionConfig) GenerateOptions {
	opts := GenerateOptions{
		TemplateVersion:    mc.Global.Template,
		Platform:          mc.Global.Platform,
		HTTPTimeout:       mc.Global.HTTPTimeout,
		MirrorURL:         mc.Global.MirrorURL,
		ClientSubnet:      mc.Global.ClientSubnet,
		DNSLocalServer:    mc.Global.DNSLocalServer,
		Format:            mc.Global.Format,
		RemoveEmoji:       mc.Global.RemoveEmoji,
		SkipTLSVerify:     mc.Global.SkipTLSVerify,
		ExternalController: mc.Global.WebUIAddress,
	}
	
	// Override with subscription-specific settings
	if subConfig.RemoveEmoji != nil {
		opts.RemoveEmoji = *subConfig.RemoveEmoji
	}
	
	if subConfig.HTTPTimeout != nil {
		opts.HTTPTimeout = *subConfig.HTTPTimeout
	}
	
	if subConfig.SkipTLSVerify != nil {
		opts.SkipTLSVerify = *subConfig.SkipTLSVerify
	}
	
	return opts
}

// ValidateConfig validates the multi-config structure
func (mc *MultiConfig) ValidateConfig() error {
	if len(mc.Subscriptions) == 0 {
		return ErrNoSubscriptions
	}
	
	// Check for duplicate subscription names
	nameMap := make(map[string]bool)
	for _, sub := range mc.Subscriptions {
		if sub.Name == "" {
			return ErrEmptySubscriptionName
		}
		if sub.URL == "" {
			return ErrEmptySubscriptionURL
		}
		
		if nameMap[sub.Name] {
			return ErrDuplicateSubscriptionName
		}
		nameMap[sub.Name] = true
	}
	
	return nil
}