package test

import (
	"os"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/platform"
)

// TestIOSAdapterStripsDesktopAPIService iOS 沙盒必须剥离桌面专属的 api service:
// 模板 api service 绑定桌面局域网 IP(如 192.168.31.1), iPhone 上 bind 失败导致 SFI 无法启动
func TestIOSAdapterStripsDesktopAPIService(t *testing.T) {
	adapter := platform.NewIOSAdapter("")

	t.Run("剥离api service并保留其他service", func(t *testing.T) {
		cfg := &config.Config{
			Services: []map[string]any{
				{"type": "api", "tag": "api-in", "listen": "192.168.31.1", "listen_port": 9091.0},
				{"type": "ss", "tag": "ss-in", "listen": "127.0.0.1", "listen_port": 8388.0},
			},
		}
		if err := adapter.AdaptConfig(cfg, config.TemplateOptions{Platform: "ios"}); err != nil {
			t.Fatal(err)
		}
		if len(cfg.Services) != 1 || cfg.Services[0]["tag"] != "ss-in" {
			t.Fatalf("api service should be stripped, ss kept, got: %v", cfg.Services)
		}
	})

	t.Run("全部为api时services置空", func(t *testing.T) {
		cfg := &config.Config{
			Services: []map[string]any{
				{"type": "api", "tag": "api-in", "listen": "192.168.31.1", "listen_port": 9091.0},
			},
		}
		if err := adapter.AdaptConfig(cfg, config.TemplateOptions{Platform: "ios"}); err != nil {
			t.Fatal(err)
		}
		if cfg.Services != nil {
			t.Fatalf("services should be nil when empty, got: %v", cfg.Services)
		}
	})

	t.Run("无services时不受影响", func(t *testing.T) {
		cfg := &config.Config{}
		if err := adapter.AdaptConfig(cfg, config.TemplateOptions{Platform: "ios"}); err != nil {
			t.Fatal(err)
		}
		if cfg.Services != nil {
			t.Fatalf("services should stay nil, got: %v", cfg.Services)
		}
	})
}

// TestIOSAdapterKeepsDefaultHTTPClient 确认 iOS 不破坏 1.14 的规则集下载通道
// (route.default_http_client -> hc-direct, 与桌面共享同一语义)
func TestIOSAdapterKeepsRouteDownloadChannel(t *testing.T) {
	adapter := platform.NewIOSAdapter("")
	cfg := &config.Config{
		Route: map[string]any{
			"default_http_client": "hc-direct",
			"default_mark":        1,
		},
	}
	if err := adapter.AdaptConfig(cfg, config.TemplateOptions{Platform: "ios"}); err != nil {
		t.Fatal(err)
	}
	if cfg.Route["default_http_client"] != "hc-direct" {
		t.Fatalf("default_http_client should be kept, got: %v", cfg.Route["default_http_client"])
	}
	if _, has := cfg.Route["default_mark"]; has {
		t.Fatal("default_mark should be removed on iOS")
	}
}

// TestIOSPlatformTemplateNoHTTPProxy ios-tun.json 模板不得启用 platform.http_proxy:
// SFI 会把它应用到 iOS 系统 HTTP 代理, 而模板没有本地入站监听 7890,
// 系统代理指向死端口会导致走系统代理的应用 HTTP 请求全部失败
func TestIOSPlatformTemplateNoHTTPProxy(t *testing.T) {
	data, err := os.ReadFile("../internal/template/configs/platform/ios-tun.json")
	if err != nil {
		t.Fatalf("read ios-tun.json: %v", err)
	}
	if strings.Contains(string(data), "http_proxy") {
		t.Fatalf("ios-tun.json should not enable platform.http_proxy (no local inbound backs port 7890):\n%s", data)
	}
}
