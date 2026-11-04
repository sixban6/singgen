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
