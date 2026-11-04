package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goccy/go-json"
	"github.com/sixban6/singgen/pkg/singgen"
)

const singboxCheckFixture = "testdata/singbox_check_nodes.txt"

// TestSingBoxCheck 端到端校验：生成的配置必须能被真实的 sing-box 加载。
// 本机（或 CI）PATH 中没有 sing-box 时自动跳过；设置 SINGGEN_REQUIRE_SINGBOX=1 时缺失则失败。
func TestSingBoxCheck(t *testing.T) {
	singbox, err := exec.LookPath("sing-box")
	if err != nil {
		if os.Getenv("SINGGEN_REQUIRE_SINGBOX") == "1" {
			t.Fatal("sing-box binary not found in PATH")
		}
		t.Skip("sing-box binary not found in PATH, skipping end-to-end check")
	}

	platforms := []string{"darwin", "ios", "windows", "linux"}
	templates := []string{"v1.14", "v1.12"}

	for _, tmpl := range templates {
		for _, platform := range platforms {
			for _, tailscale := range []bool{false, true} {
				name := tmpl + "/" + platform
				if tailscale {
					name += "/tailscale"
				}
				t.Run(name, func(t *testing.T) {
					// linux 模板使用 default_mark / tproxy，只能在 linux 上校验
					if platform == "linux" && runtime.GOOS != "linux" {
						t.Skip("linux-only options cannot be checked on " + runtime.GOOS)
					}

					opts := []singgen.Option{
						singgen.WithTemplate(tmpl),
						singgen.WithPlatform(platform),
					}
					if tailscale {
						opts = append(opts,
							singgen.WithTSAuthKey("tskey-auth-test"),
							singgen.WithTSLanIPCIDR("192.168.31.0/24"),
						)
					}

					data, err := singgen.GenerateConfigBytes(context.Background(), singboxCheckFixture, opts...)
					if err != nil {
						t.Fatalf("generate failed: %v", err)
					}

					path := filepath.Join(t.TempDir(), "config.json")
					if err := os.WriteFile(path, data, 0o600); err != nil {
						t.Fatal(err)
					}

					out, err := exec.Command(singbox, "check", "-c", path).CombinedOutput()
					if err != nil {
						t.Fatalf("sing-box check failed: %v\n%s", err, out)
					}
				})
			}
		}
	}
}

// TestGeneratedOutboundsRegression 不依赖 sing-box 二进制，直接断言关键字段。
func TestGeneratedOutboundsRegression(t *testing.T) {
	data, err := singgen.GenerateConfigBytes(context.Background(), singboxCheckFixture,
		singgen.WithTemplate("v1.14"), singgen.WithPlatform("darwin"))
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}

	byTag := make(map[string]map[string]any)
	for _, ob := range cfg.Outbounds {
		tag, _ := ob["tag"].(string)
		if _, dup := byTag[tag]; dup {
			t.Errorf("duplicate outbound tag: %q", tag)
		}
		byTag[tag] = ob
	}

	mustGet := func(tag string) map[string]any {
		t.Helper()
		ob, ok := byTag[tag]
		if !ok {
			t.Fatalf("outbound %q not found", tag)
		}
		return ob
	}
	transportOf := func(ob map[string]any) map[string]any {
		tr, _ := ob["transport"].(map[string]any)
		return tr
	}
	tlsOf := func(ob map[string]any) map[string]any {
		tls, _ := ob["tls"].(map[string]any)
		return tls
	}

	t.Run("ws host goes to headers", func(t *testing.T) {
		tr := transportOf(mustGet("VL-WS"))
		if _, bad := tr["host"]; bad {
			t.Errorf("ws transport must not contain host field: %v", tr)
		}
		headers, _ := tr["headers"].(map[string]any)
		if headers["Host"] != "v.example.com" {
			t.Errorf("expected headers.Host=v.example.com, got %v", tr)
		}
	})

	t.Run("ws early data parsed from path", func(t *testing.T) {
		tr := transportOf(mustGet("VM-WS"))
		if tr["path"] != "/p" || tr["max_early_data"] != float64(2048) {
			t.Errorf("unexpected early data handling: %v", tr)
		}
	})

	t.Run("grpc service_name", func(t *testing.T) {
		if got := transportOf(mustGet("VL-GRPC"))["service_name"]; got != "vlgrpc" {
			t.Errorf("vless grpc service_name = %v", got)
		}
		if got := transportOf(mustGet("VM-GRPC"))["service_name"]; got != "grpcsvc" {
			t.Errorf("vmess grpc service_name = %v", got)
		}
	})

	t.Run("httpupgrade keeps host", func(t *testing.T) {
		if got := transportOf(mustGet("VL-HU"))["host"]; got != "u.example.com" {
			t.Errorf("httpupgrade host = %v", got)
		}
	})

	t.Run("unsupported transport is skipped", func(t *testing.T) {
		if _, ok := byTag["VL-XHTTP-UNSUPPORTED"]; ok {
			t.Error("xhttp node should be skipped because sing-box does not support it")
		}
	})

	t.Run("duplicate and reserved tags are renamed", func(t *testing.T) {
		for _, tag := range []string{"HK", "HK 2", "HK 3", "Proxy 2"} {
			mustGet(tag)
		}
		if typ := mustGet("Proxy")["type"]; typ != "selector" {
			t.Errorf("template group Proxy was overwritten by a node: type=%v", typ)
		}
	})

	t.Run("vmess verifies certificates and keeps cipher", func(t *testing.T) {
		ob := mustGet("VM-WS")
		if tlsOf(ob)["insecure"] != false {
			t.Errorf("vmess must not skip certificate verification by default: %v", tlsOf(ob))
		}
		if tlsOf(ob)["server_name"] != "h.example.com" {
			t.Errorf("vmess SNI should fall back to host: %v", tlsOf(ob))
		}
		if _, bad := ob["multiplex"]; bad {
			t.Errorf("vmess must not derive multiplex from alter_id: %v", ob)
		}
		if ob["security"] != "auto" {
			t.Errorf("vmess security = %v", ob["security"])
		}
		if mustGet("VM-GRPC")["security"] != "aes-128-gcm" {
			t.Errorf("vmess cipher lost")
		}
	})

	t.Run("utls fingerprint for plain tls", func(t *testing.T) {
		utls, _ := tlsOf(mustGet("VL-WS"))["utls"].(map[string]any)
		if utls["fingerprint"] != "firefox" {
			t.Errorf("expected utls fingerprint firefox, got %v", tlsOf(mustGet("VL-WS")))
		}
	})
}
