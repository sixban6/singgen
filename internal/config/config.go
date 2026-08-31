package config

// Config sing-box配置结构
type Config struct {
	Log          map[string]any   `json:"log"`
	Experimental map[string]any   `json:"experimental"`
	DNS          map[string]any   `json:"dns"`
	Inbounds     []map[string]any `json:"inbounds"`
	Outbounds    []map[string]any `json:"outbounds"`
	Route        map[string]any   `json:"route"`
	Certificate  map[string]any   `json:"certificate,omitempty"`
	Endpoints    []map[string]any `json:"endpoints,omitempty"`
	// sing-box 1.14+: 顶层 services（api 等）与 http_clients
	Services    []map[string]any `json:"services,omitempty"`
	HTTPClients []map[string]any `json:"http_clients,omitempty"`
}

// TemplateOptions 模板选项
type TemplateOptions struct {
	MirrorURL          string
	ExternalController string
	ClientSubnet       string
	RemoveEmoji        bool
	DNSLocalServer     string
	Platform           string
	TSAuthKey          string
	TSLanIPCIDR        string
}
