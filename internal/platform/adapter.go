package platform

import (
	"github.com/sixban6/singgen/internal/config"
)

// PlatformAdapter 平台适配器接口
type PlatformAdapter interface {
	// AdaptConfig 适配配置到目标平台
	AdaptConfig(config *config.Config, options config.TemplateOptions) error
	// GetInboundConfig 获取平台特定的inbound配置
	GetInboundConfig() ([]map[string]any, error)
	// RequiresExternalController 是否需要外部控制器
	RequiresExternalController() bool
	// GetPlatformName 获取平台名称
	GetPlatformName() string
	// GetConfigFileName 获取平台配置文件名
	GetConfigFileName() string
}

// PlatformType 平台类型
type PlatformType string

const (
	Linux   PlatformType = "linux"
	Darwin  PlatformType = "darwin"
	IOS     PlatformType = "ios"
	Windows PlatformType = "windows"
)

// ValidPlatforms 有效的平台列表
var ValidPlatforms = []PlatformType{Linux, Darwin, IOS, Windows}

// IsValidPlatform 检查是否为有效平台
func IsValidPlatform(platform string) bool {
	for _, p := range ValidPlatforms {
		if string(p) == platform {
			return true
		}
	}
	return false
}
