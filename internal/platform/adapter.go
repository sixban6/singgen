package platform

import (
	"fmt"
	"net"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/logx"
)

// PlatformAdapter 平台适配器接口
type PlatformAdapter interface {
	// AdaptConfig 适配配置到目标平台
	AdaptConfig(config *config.Config, options config.TemplateOptions) error
	// GetPlatformName 获取平台名称
	GetPlatformName() string
	// GetConfigFileName 获取平台 inbound 配置文件名（位于 configs/platform/）
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

// Profile 声明式地描述一个平台与基准模板之间的差异。
// 新增平台只需在 profiles 中增加一项并提供对应的 inbound 配置文件。
type Profile struct {
	// Name 平台名称
	Name PlatformType
	// InboundFile 替换模板 inbounds 的配置文件（configs/platform/ 下）
	InboundFile string
	// Router 为 true 表示部署在路由器/服务器上（linux tproxy）：
	// 保留 default_mark、api service、http_clients 与 cache 文件路径。
	// 为 false 表示终端客户端（macOS/iOS/Windows）。
	Router bool
}

var profiles = map[PlatformType]Profile{
	Linux:   {Name: Linux, InboundFile: "linux-tproxy.json", Router: true},
	Darwin:  {Name: Darwin, InboundFile: "mac-tun.json"},
	IOS:     {Name: IOS, InboundFile: "ios-tun.json"},
	Windows: {Name: Windows, InboundFile: "win-tun.json"},
}

// routerCachePath linux 部署机上 cache 文件的位置
const routerCachePath = "/etc/sing-box/cache.db"

// Adapter 基于 Profile 的通用平台适配器
type Adapter struct {
	profile Profile
}

// IsValidPlatform 检查是否为有效平台
func IsValidPlatform(platform string) bool {
	_, ok := profiles[PlatformType(platform)]
	return ok
}

// GetSupportedPlatforms 获取支持的平台列表
func GetSupportedPlatforms() []string {
	platforms := make([]string, len(ValidPlatforms))
	for i, p := range ValidPlatforms {
		platforms[i] = string(p)
	}
	return platforms
}

// CreateAdapterByString 根据平台名称创建适配器（支持 mac/macos 别名）。
// configDir 参数保留以兼容旧调用，已不再使用。
func CreateAdapterByString(platform, configDir string) (PlatformAdapter, error) {
	switch platform {
	case "mac", "macos":
		platform = string(Darwin)
	}
	p, ok := profiles[PlatformType(platform)]
	if !ok {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
	return &Adapter{profile: p}, nil
}

// NewLinuxAdapter 创建 Linux 适配器
func NewLinuxAdapter(string) PlatformAdapter { return &Adapter{profile: profiles[Linux]} }

// NewDarwinAdapter 创建 macOS 适配器
func NewDarwinAdapter(string) PlatformAdapter { return &Adapter{profile: profiles[Darwin]} }

// NewIOSAdapter 创建 iOS 适配器
func NewIOSAdapter(string) PlatformAdapter { return &Adapter{profile: profiles[IOS]} }

// NewWindowsAdapter 创建 Windows 适配器
func NewWindowsAdapter(string) PlatformAdapter { return &Adapter{profile: profiles[Windows]} }

// GetPlatformName 获取平台名称
func (a *Adapter) GetPlatformName() string { return string(a.profile.Name) }

// GetConfigFileName 获取 inbound 配置文件名
func (a *Adapter) GetConfigFileName() string { return a.profile.InboundFile }

// AdaptConfig 按 Profile 适配配置
func (a *Adapter) AdaptConfig(cfg *config.Config, options config.TemplateOptions) error {
	if cfg.Experimental == nil {
		cfg.Experimental = make(map[string]any)
	}

	if a.profile.Router {
		// tproxy 需要 default_mark 防止回环
		if v, ok := cfg.Route["auto_detect_interface"]; ok && v == true {
			cfg.Route["default_mark"] = 1
		}
		if options.ExternalController != "" {
			ensureClashAPI(cfg)["external_controller"] = options.ExternalController
		}
		if cacheFile, ok := cfg.Experimental["cache_file"].(map[string]any); ok {
			cacheFile["path"] = routerCachePath
		}
		return nil
	}

	// 终端客户端
	// default_mark 仅 linux 支持
	delete(cfg.Route, "default_mark")
	// 剥离桌面专属的 api service(详见 stripDesktopAPIServices)
	stripDesktopAPIServices(cfg)
	// 规则集下载切换为 legacy download_detour(详见 useLegacyRuleSetDownload)
	useLegacyRuleSetDownload(cfg)
	clashAPI := ensureClashAPI(cfg)
	if options.ExternalController != "" {
		// 终端设备只能绑定本机地址：为路由器准备的局域网地址（如 192.168.31.1:9095）
		// 在客户端上会绑定失败导致 sing-box 无法启动，因此改写为 127.0.0.1 并保留端口
		controller, rewritten := clientController(options.ExternalController)
		if rewritten {
			logx.Or(options.Logger).Warn("external controller host is not a loopback address; using 127.0.0.1 on client platform",
				"platform", a.profile.Name, "requested", options.ExternalController, "used", controller)
		}
		clashAPI["external_controller"] = controller
	}
	// 客户端(沙盒)使用默认 cache 位置
	if cacheFile, ok := cfg.Experimental["cache_file"].(map[string]any); ok {
		delete(cacheFile, "path")
	}
	return nil
}

// clientController 将 external controller 地址限制为本机回环地址，返回结果与是否发生改写
func clientController(addr string) (string, bool) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// 无端口等格式问题：保持原样，由 sing-box 校验
		return addr, false
	}
	if host == "localhost" {
		return addr, false
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return addr, false
	}
	return net.JoinHostPort("127.0.0.1", port), true
}

// ensureClashAPI 获取或创建 experimental.clash_api，并补齐 default_mode / secret 默认值
func ensureClashAPI(cfg *config.Config) map[string]any {
	clashAPI, ok := cfg.Experimental["clash_api"].(map[string]any)
	if !ok {
		clashAPI = make(map[string]any)
		cfg.Experimental["clash_api"] = clashAPI
	}
	if _, ok := clashAPI["default_mode"]; !ok {
		clashAPI["default_mode"] = "rule"
	}
	if _, ok := clashAPI["secret"]; !ok {
		clashAPI["secret"] = ""
	}
	return clashAPI
}
