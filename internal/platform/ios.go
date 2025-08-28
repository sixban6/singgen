package platform

import (
	"path/filepath"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/util"
)

// IOSAdapter iOS平台适配器
type IOSAdapter struct {
	configDir string
}

// NewIOSAdapter 创建iOS适配器
func NewIOSAdapter(configDir string) *IOSAdapter {
	return &IOSAdapter{
		configDir: configDir,
	}
}

// AdaptConfig 适配配置到iOS平台
func (a *IOSAdapter) AdaptConfig(config *config.Config, options config.TemplateOptions) error {
	// iOS不需要default_mark，删除它
	delete(config.Route, "default_mark")

	// iOS默认使用external_controller，无需用户传入
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

	// 更新必要的字段
	clashAPI["external_controller"] = "127.0.0.1:9095"
	if _, ok := clashAPI["default_mode"]; !ok {
		clashAPI["default_mode"] = "rule"
	}
	if _, ok := clashAPI["secret"]; !ok {
		clashAPI["secret"] = ""
	}

	// iOS不需要cache文件路径，删除path字段
	if cacheFile, ok := experimental["cache_file"].(map[string]any); ok {
		delete(cacheFile, "path")
	}

	return nil
}

// GetInboundConfig 获取iOS特定的inbound配置
func (a *IOSAdapter) GetInboundConfig() ([]map[string]any, error) {
	configPath := filepath.Join(a.configDir, "ios-tun.json")
	data, err := util.ReadFile(configPath)
	if err != nil {
		// 如果配置文件不存在，返回默认配置
		return []map[string]any{
			{
				"type":    "tun",
				"tag":     "tun-in",
				"address": []string{"10.8.8.8/30"},
				"mtu":     9000,
				"auto_route": true,
				"stack":      "system",
				"route_exclude_address_set": []string{
					"geosite-private",
					"geosite-ctm_cn",
					"geoip-cn",
				},
			},
		}, nil
	}

	var inbounds []map[string]any
	if err := util.Unmarshal(data, &inbounds); err != nil {
		return nil, err
	}

	return inbounds, nil
}

// RequiresExternalController iOS不需要用户传入external_controller
func (a *IOSAdapter) RequiresExternalController() bool {
	return false
}

// GetPlatformName 获取平台名称
func (a *IOSAdapter) GetPlatformName() string {
	return "ios"
}