package config

// Config sing-box配置结构
type Config struct {
	Log          map[string]any         `json:"log"`
	Experimental map[string]any         `json:"experimental"`
	DNS          map[string]any         `json:"dns"`
	Inbounds     []map[string]any       `json:"inbounds"`
	Outbounds    []map[string]any       `json:"outbounds"`
	Route        map[string]any         `json:"route"`
}

// TemplateOptions 模板选项
type TemplateOptions struct {
	MirrorURL          string
	ExternalController string
	ClientSubnet       string
	RemoveEmoji        bool
	DNSLocalServer     string
	Platform           string
}