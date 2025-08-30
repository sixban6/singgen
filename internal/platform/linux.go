package platform

import (
	"path/filepath"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/util"
)

// LinuxAdapter Linux平台适配器
type LinuxAdapter struct {
	configDir string
}

// NewLinuxAdapter 创建Linux适配器
func NewLinuxAdapter(configDir string) *LinuxAdapter {
	return &LinuxAdapter{
		configDir: configDir,
	}
}

// AdaptConfig 适配配置到Linux平台
func (a *LinuxAdapter) AdaptConfig(config *config.Config, options config.TemplateOptions) error {
	// Linux使用tproxy，需要default_mark
	if route, ok := config.Route["auto_detect_interface"]; ok && route == true {
		config.Route["default_mark"] = 1
	}

	// Linux需要external_controller配置
	if a.RequiresExternalController() && options.ExternalController != "" {
		experimental := config.Experimental
		if experimental == nil {
			experimental = make(map[string]any)
			config.Experimental = experimental
		}

		// 获取或创建clash_api配置，保留已有字段
		var clashAPI map[string]any
		if existingClashAPI, ok := experimental["clash_api"].(map[string]any); ok {
			clashAPI = existingClashAPI
		} else {
			clashAPI = make(map[string]any)
			experimental["clash_api"] = clashAPI
		}

		// 只更新必要的字段
		clashAPI["external_controller"] = options.ExternalController
		if _, ok := clashAPI["default_mode"]; !ok {
			clashAPI["default_mode"] = "rule"
		}
		if _, ok := clashAPI["secret"]; !ok {
			clashAPI["secret"] = ""
		}
	}

	// Linux需要cache文件路径
	experimental := config.Experimental
	if experimental == nil {
		experimental = make(map[string]any)
		config.Experimental = experimental
	}

	if cacheFile, ok := experimental["cache_file"].(map[string]any); ok {
		cacheFile["path"] = "/etc/sing-box/cache.db"
	}

	return nil
}

// GetInboundConfig 获取Linux特定的inbound配置
func (a *LinuxAdapter) GetInboundConfig() ([]map[string]any, error) {
	configPath := filepath.Join(a.configDir, "linux-tproxy.json")
	data, err := util.ReadFile(configPath)
	if err != nil {
		// 如果配置文件不存在，返回默认配置
		return []map[string]any{
			{
				"type":        "tproxy",
				"tag":         "tproxy-in",
				"listen":      "::",
				"listen_port": 7895,
				"udp_timeout": "5m",
			},
		}, nil
	}

	var inbounds []map[string]any
	if err := util.Unmarshal(data, &inbounds); err != nil {
		return nil, err
	}

	return inbounds, nil
}

// RequiresExternalController Linux需要外部控制器
func (a *LinuxAdapter) RequiresExternalController() bool {
	return true
}

// GetPlatformName 获取平台名称
func (a *LinuxAdapter) GetPlatformName() string {
	return "linux"
}

// GetConfigFileName 获取配置文件名
func (a *LinuxAdapter) GetConfigFileName() string {
	return "linux-tproxy.json"
}