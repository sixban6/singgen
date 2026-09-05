package platform

import (
	"github.com/sixban6/singgen/internal/config"
)

// stripDesktopAPIServices 剥离桌面专属的 api service。
//
// v1.14 模板的 api service 绑定部署机的局域网 IP(如 listen: 192.168.31.1),
// 供官方 Dashboard / 图形客户端**远程控制部署机上的 sing-box** —— 该语义只对
// 拥有该 IP 的部署机(linux 路由器/服务器)成立。为目标终端设备(macOS/Windows/iOS)
// 生成的配置里, 设备并不拥有模板硬编码的 IP, bind 失败会直接导致客户端无法启动
// ("bind: can't assign requested address")。
//
// 仅 linux 适配器保留该 service。
func stripDesktopAPIServices(config *config.Config) {
	if len(config.Services) == 0 {
		return
	}
	kept := make([]map[string]any, 0, len(config.Services))
	for _, sv := range config.Services {
		if t, _ := sv["type"].(string); t == "api" {
			continue
		}
		kept = append(kept, sv)
	}
	if len(kept) == 0 {
		config.Services = nil // omitempty 移除该 key
	} else {
		config.Services = kept
	}
}
