package test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixban6/singgen/internal/logx"
	"github.com/sixban6/singgen/pkg/singgen"
)

// TestLibraryLoggerReceivesInternalWarnings 库调用方通过 WithLogger 注入的 logger
// 必须能收到内部组件的警告（例如节点被跳过），且日志中不得出现节点密码。
func TestLibraryLoggerReceivesInternalWarnings(t *testing.T) {
	src := filepath.Join(t.TempDir(), "nodes.txt")
	content := strings.Join([]string{
		"trojan://supersecretpw@1.2.3.4:443?sni=a.com#OK",
		"vless://b831381d-6324-4d53-ad4f-8cda48b30811@2.2.2.5:443?type=xhttp&security=tls#BAD-XHTTP",
		"trojan://anothersecret@1.2.3.4:notaport#BAD-PORT",
	}, "\n")
	if err := os.WriteFile(src, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	if _, err := singgen.GenerateConfigBytes(context.Background(), src,
		singgen.WithLogger(logger), singgen.WithPlatform("darwin")); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(logs, "Failed to transform node") || !strings.Contains(logs, "BAD-XHTTP") {
		t.Errorf("expected transform warning for skipped node, got logs:\n%s", logs)
	}
	for _, secret := range []string{"supersecretpw", "anothersecret", "b831381d-6324"} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs leak credential %q:\n%s", secret, logs)
		}
	}
}

func TestRedactURL(t *testing.T) {
	cases := map[string]string{
		"trojan://pw@1.2.3.4:443?sni=a.com#HK%20Node": "trojan://***@1.2.3.4:443#HK Node",
		"vmess://eyJ2IjoiMiJ9":                        "vmess://***",
		"ss://YWVzOnB3@4.4.4.4:8388#SS":               "ss://***@4.4.4.4:8388#SS",
		"garbage":                                     "<invalid>",
	}
	for in, want := range cases {
		if got := logx.RedactURL(in); got != want {
			t.Errorf("RedactURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestErrRedactsCredentials(t *testing.T) {
	err := errors.New(`parse "trojan://secret@1.2.3.4:x": invalid port`)
	if got := logx.Err(err); strings.Contains(got, "secret") {
		t.Errorf("logx.Err leaked credential: %s", got)
	}
}

// TestTransformDoesNotMutateCallerNodes 带宽注入不得修改调用方传入的节点
func TestTransformDoesNotMutateCallerNodes(t *testing.T) {
	nodes, err := singgen.ParseNodes(context.Background(), singboxCheckFixture)
	if err != nil {
		t.Fatal(err)
	}
	var hy2 *singgen.Node
	for i := range nodes {
		if nodes[i].Type == "hysteria2" {
			hy2 = &nodes[i]
		}
	}
	if hy2 == nil {
		t.Fatal("fixture has no hysteria2 node")
	}
	before := len(hy2.Extra)
	_, beforeUp := hy2.Extra["up_mbps"]

	tr := singgen.NewContextTransformer(singgen.GenerateOptions{UpMbps: 50, DownMbps: 500})
	if _, err := tr.Transform(context.Background(), nodes); err != nil {
		t.Fatal(err)
	}
	if _, afterUp := hy2.Extra["up_mbps"]; len(hy2.Extra) != before || afterUp != beforeUp {
		t.Errorf("caller node Extra was mutated: %v", hy2.Extra)
	}
}

// TestExternalControllerRespectedOnAllPlatforms 修复前 darwin/ios/windows 会无视用户参数写死地址
func TestExternalControllerRespectedOnAllPlatforms(t *testing.T) {
	controllerOf := func(platform, addr string) (string, string) {
		var buf bytes.Buffer
		cfg, err := singgen.GenerateConfig(context.Background(), singboxCheckFixture,
			singgen.WithPlatform(platform), singgen.WithExternalController(addr),
			singgen.WithLogger(slog.New(slog.NewTextHandler(&buf, nil))))
		if err != nil {
			t.Fatal(err)
		}
		clash, _ := cfg.Experimental["clash_api"].(map[string]any)
		got, _ := clash["external_controller"].(string)
		return got, buf.String()
	}

	for _, platform := range []string{"linux", "darwin", "ios", "windows"} {
		if got, _ := controllerOf(platform, "127.0.0.1:7777"); got != "127.0.0.1:7777" {
			t.Errorf("%s: external_controller = %s, want 127.0.0.1:7777", platform, got)
		}
	}

	// 路由器的局域网地址：linux 原样使用；客户端平台改写为 127.0.0.1 并告警
	if got, _ := controllerOf("linux", "192.168.31.1:9095"); got != "192.168.31.1:9095" {
		t.Errorf("linux: external_controller = %s", got)
	}
	for _, platform := range []string{"darwin", "ios", "windows"} {
		got, logs := controllerOf(platform, "192.168.31.1:9095")
		if got != "127.0.0.1:9095" {
			t.Errorf("%s: external_controller = %s, want 127.0.0.1:9095", platform, got)
		}
		if !strings.Contains(logs, "not a loopback address") {
			t.Errorf("%s: expected warning about non-loopback controller, logs: %s", platform, logs)
		}
	}
}
