package model

type Node struct {
	ID       string            `json:"id"`
	Tag      string            `json:"tag"`
	Type     string            `json:"type"`
	Addr     string            `json:"addr"`
	Port     uint16            `json:"port"`
	UUID     string            `json:"uuid,omitempty"`
	Password string            `json:"password,omitempty"`
	Security Security          `json:"security"`
	Transport                  `json:"transport"`
	Extra    map[string]any    `json:"extra,omitempty"`
}

type Security struct {
	TLS        bool     `json:"tls"`
	SkipVerify bool     `json:"skip_verify"`
	ServerName string   `json:"server_name,omitempty"`
	ALPN       []string `json:"alpn,omitempty"`
}

type Transport struct {
	Net     string            `json:"net,omitempty"`
	Host    string            `json:"host,omitempty"`
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}