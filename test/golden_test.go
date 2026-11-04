package test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sixban6/singgen/pkg/singgen"
)

// 运行 `go test ./test -run TestGolden -update` 以重新生成快照。
// 快照用于保证重构不改变生成结果：任何输出变化都会以 diff 形式出现在提交中。
var updateGolden = flag.Bool("update", false, "update golden snapshot files")

const goldenDir = "testdata/golden"

type goldenCase struct {
	name string
	gen  func() ([]byte, error)
}

func goldenCases() []goldenCase {
	ctx := context.Background()
	var cases []goldenCase

	for _, tmpl := range []string{"v1.14", "v1.12"} {
		for _, platform := range []string{"linux", "darwin", "ios", "windows"} {
			for _, ts := range []bool{false, true} {
				tmpl, platform, ts := tmpl, platform, ts
				name := fmt.Sprintf("single_%s_%s", tmpl, platform)
				if ts {
					name += "_tailscale"
				}
				cases = append(cases, goldenCase{name: name, gen: func() ([]byte, error) {
					opts := []singgen.Option{singgen.WithTemplate(tmpl), singgen.WithPlatform(platform)}
					if ts {
						opts = append(opts, singgen.WithTSAuthKey("tskey-auth-test"), singgen.WithTSLanIPCIDR("192.168.1.0/24"))
					}
					return singgen.GenerateConfigBytes(ctx, singboxCheckFixture, opts...)
				}})
			}
		}

		// 自定义参数：覆盖所有会被注入模板的选项
		for _, platform := range []string{"linux", "darwin"} {
			tmpl, platform := tmpl, platform
			cases = append(cases, goldenCase{
				name: fmt.Sprintf("custom_%s_%s", tmpl, platform),
				gen: func() ([]byte, error) {
					return singgen.GenerateConfigBytes(ctx, singboxCheckFixture,
						singgen.WithTemplate(tmpl),
						singgen.WithPlatform(platform),
						singgen.WithDNSServer("223.6.6.6"),
						singgen.WithClientSubnet("1.2.3.0/24"),
						singgen.WithMirrorURL(""),
						singgen.WithExternalController("127.0.0.1:7777"),
						singgen.WithEmojiRemoval(false),
					)
				},
			})
		}
	}

	// emoji 节点（默认移除 emoji）
	cases = append(cases, goldenCase{name: "emoji_nodes_default", gen: func() ([]byte, error) {
		return singgen.GenerateConfigBytes(ctx, "../test-nodes.txt")
	}})

	// YAML 输出
	cases = append(cases, goldenCase{name: "yaml_output", gen: func() ([]byte, error) {
		return singgen.GenerateConfigBytes(ctx, singboxCheckFixture, singgen.WithOutputFormat("yaml"), singgen.WithPlatform("darwin"))
	}})

	// 多订阅
	for _, platform := range []string{"linux", "ios"} {
		platform := platform
		cases = append(cases, goldenCase{name: "multi_" + platform, gen: func() ([]byte, error) {
			mc := singgen.GetDefaultMultiConfig()
			mc.Global.Platform = platform
			mc.Global.TSAuthKey = "tskey-auth-test"
			mc.Global.TSLanIPCIDR = "192.168.1.0/24"
			keep := false
			skip := true
			mc.Subscriptions = []singgen.SubscriptionConfig{
				{Name: "primary", URL: singboxCheckFixture},
				{Name: "backup", URL: "../test-nodes.txt", RemoveEmoji: &keep, SkipTLSVerify: &skip},
			}
			return singgen.GenerateConfigBytesFromMulti(ctx, mc)
		}})
	}

	return cases
}

func TestGolden(t *testing.T) {
	for _, tc := range goldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.gen()
			if err != nil {
				t.Fatalf("generate failed: %v", err)
			}

			ext := ".json"
			if strings.HasPrefix(tc.name, "yaml") {
				ext = ".yaml"
			}
			path := filepath.Join(goldenDir, tc.name+ext)

			if *updateGolden {
				if err := os.MkdirAll(goldenDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("missing golden file %s (run with -update): %v", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("output differs from %s (run with -update if intended)\n%s", path, firstDiff(want, got))
			}
		})
	}
}

// firstDiff 返回首个不同行附近的上下文，便于定位
func firstDiff(want, got []byte) string {
	wl := strings.Split(string(want), "\n")
	gl := strings.Split(string(got), "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return fmt.Sprintf("line %d:\n  want: %s\n  got:  %s", i+1, w, g)
		}
	}
	return "(no line difference; trailing bytes differ)"
}
