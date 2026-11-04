package test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sixban6/singgen/internal/fetcher"
	"github.com/sixban6/singgen/pkg/singgen"
)

func fixtureBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(singboxCheckFixture)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fastFetcher() *fetcher.HTTPFetcher {
	f := fetcher.NewHTTPFetcher()
	f.RetryBackoff = 10 * time.Millisecond
	return f
}

func TestHTTPFetcherRetriesTransientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	body, err := fastFetcher().FetchContext(context.Background(), srv.URL)
	if err != nil || string(body) != "ok" {
		t.Fatalf("expected success after retries, got body=%q err=%v", body, err)
	}
	if calls.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", calls.Load())
	}
}

func TestHTTPFetcherDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := fastFetcher().FetchContext(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error for 404")
	}
	if calls.Load() != 1 {
		t.Errorf("404 must not be retried, got %d attempts", calls.Load())
	}
}

func TestHTTPFetcherGivesUpAfterMaxAttempts(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	if _, err := fastFetcher().FetchContext(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error")
	}
	if calls.Load() != fetcher.DefaultMaxAttempts {
		t.Errorf("expected %d attempts, got %d", fetcher.DefaultMaxAttempts, calls.Load())
	}
}

func TestHTTPFetcherHandlesGzip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Write([]byte("plain"))
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gz.Write([]byte("compressed-body"))
		gz.Close()
	}))
	defer srv.Close()

	body, err := fastFetcher().FetchContext(context.Background(), srv.URL)
	if err != nil || string(body) != "compressed-body" {
		t.Fatalf("expected transparently decompressed body, got %q err=%v", body, err)
	}
}

func TestSubscriptionCacheFallback(t *testing.T) {
	fixture := fixtureBytes(t)
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(fixture)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	url := srv.URL + "/api/v1/client/subscribe?token=CACHETOKEN42"
	var logs bytes.Buffer
	opts := []singgen.Option{
		singgen.WithCacheDir(cacheDir),
		singgen.WithPlatform("darwin"),
		singgen.WithLogger(slog.New(slog.NewTextHandler(&logs, nil))),
	}

	first, err := singgen.GenerateConfigBytes(context.Background(), url, opts...)
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}

	// 缓存文件：权限 0600，文件名不含令牌
	entries, _ := os.ReadDir(cacheDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 cache file, got %d", len(entries))
	}
	name := entries[0].Name()
	if strings.Contains(name, "CACHETOKEN42") {
		t.Errorf("cache file name leaks token: %s", name)
	}
	if st, _ := os.Stat(filepath.Join(cacheDir, name)); st.Mode().Perm() != 0o600 {
		t.Errorf("cache file mode = %v, want 0600", st.Mode().Perm())
	}

	// 订阅失效后使用缓存，生成结果一致
	fail.Store(true)
	second, err := singgen.GenerateConfigBytes(context.Background(), url, opts...)
	if err != nil {
		t.Fatalf("expected fallback to cache, got %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Error("config generated from cache differs from original")
	}
	if !strings.Contains(logs.String(), "using cached copy") {
		t.Errorf("expected cache fallback warning, logs:\n%s", logs.String())
	}
	if strings.Contains(logs.String(), "CACHETOKEN42") {
		t.Errorf("token leaked into logs:\n%s", logs.String())
	}

	// 未启用缓存时直接失败
	if _, err := singgen.GenerateConfigBytes(context.Background(), url, singgen.WithPlatform("darwin")); err == nil {
		t.Error("expected failure without cache")
	}
}

func TestSubscriptionCacheIgnoresGarbage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>login required</html>"))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	_, _ = singgen.GenerateConfigBytes(context.Background(), srv.URL, singgen.WithCacheDir(cacheDir))
	if entries, _ := os.ReadDir(cacheDir); len(entries) != 0 {
		t.Errorf("content without nodes must not be cached, found %d files", len(entries))
	}
}

func TestParseSubscriptionUserInfo(t *testing.T) {
	info, ok := singgen.ParseSubscriptionUserInfo("upload=1073741824; download=2147483648; total=10737418240; expire=1767196800")
	if !ok {
		t.Fatal("expected header to parse")
	}
	if info.Used() != 3<<30 || info.Remaining() != 7<<30 || info.Expire.Unix() != 1767196800 {
		t.Errorf("unexpected info: %+v", info)
	}
	if _, ok := singgen.ParseSubscriptionUserInfo(""); ok {
		t.Error("empty header must not parse")
	}
}

func TestSubscriptionUserInfoWarnings(t *testing.T) {
	fixture := fixtureBytes(t)
	cases := map[string]struct {
		header string
		want   string
	}{
		"low traffic":  {"upload=0; download=9663676416; total=10737418240", "running low"},
		"expires soon": {"upload=0; download=0; total=10737418240; expire=" + itoa(time.Now().Add(48*time.Hour).Unix()), "expires soon"},
		"expired":      {"upload=0; download=0; total=10737418240; expire=" + itoa(time.Now().Add(-time.Hour).Unix()), "has expired"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("subscription-userinfo", tc.header)
				w.Write(fixture)
			}))
			defer srv.Close()

			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))
			p := singgen.NewPipeline(singgen.WithPipelineOptions(singgen.WithLogger(logger), singgen.WithPlatform("darwin")))
			res, err := p.Execute(context.Background(), srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(logs.String(), tc.want) {
				t.Errorf("expected warning %q, logs:\n%s", tc.want, logs.String())
			}
			if res.Metadata.Subscription == nil || res.Metadata.Subscription.Total != 10737418240 {
				t.Errorf("pipeline metadata missing subscription info: %+v", res.Metadata.Subscription)
			}
		})
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// TestMultiSubscriptionFetchesConcurrently 多订阅并发拉取，且输出顺序与配置一致
func TestMultiSubscriptionFetchesConcurrently(t *testing.T) {
	const delay = 300 * time.Millisecond
	var mu sync.Mutex
	inFlight, maxInFlight := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		inFlight++
		if inFlight > maxInFlight {
			maxInFlight = inFlight
		}
		mu.Unlock()
		time.Sleep(delay)
		mu.Lock()
		inFlight--
		mu.Unlock()
		name := strings.TrimPrefix(r.URL.Path, "/")
		w.Write([]byte("trojan://pw@1.2.3.4:443?sni=a.com#" + name))
	}))
	defer srv.Close()

	mc := singgen.GetDefaultMultiConfig()
	mc.Global.Platform = "darwin"
	for _, n := range []string{"c", "a", "b"} {
		mc.Subscriptions = append(mc.Subscriptions, singgen.SubscriptionConfig{Name: n, URL: srv.URL + "/node-" + n})
	}

	start := time.Now()
	cfg, err := singgen.GenerateConfigFromMulti(context.Background(), mc)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed >= 3*delay {
		t.Errorf("subscriptions were fetched sequentially (%v)", elapsed)
	}
	if maxInFlight < 2 {
		t.Errorf("expected concurrent requests, max in flight = %d", maxInFlight)
	}

	var order []string
	for _, ob := range cfg.Outbounds {
		if tag, _ := ob["tag"].(string); strings.Contains(tag, "node-") {
			order = append(order, tag)
		}
	}
	if strings.Join(order, ",") != "[c] node-c,[a] node-a,[b] node-b" {
		t.Errorf("output order must follow config order, got %v", order)
	}
}

// TestNodeRulesFromConfigFile 过滤 / 重命名 / 跨订阅去重
func TestNodeRulesFromConfigFile(t *testing.T) {
	dir := t.TempDir()
	nodes := strings.Join([]string{
		"trojan://pw@1.1.1.1:443?sni=a.com#HK-01",
		"trojan://pw@1.1.1.2:443?sni=a.com#JP-01",
		"trojan://pw@1.1.1.3:443?sni=a.com#剩余流量 10GB",
		"trojan://pw@1.1.1.4:443?sni=a.com#HK-02 Premium",
	}, "\n")
	other := "trojan://pw@1.1.1.1:443?sni=a.com#Dup-of-HK-01\ntrojan://pw@9.9.9.9:443?sni=a.com#US-01"
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte(nodes), 0o600)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte(other), 0o600)

	cfgYAML := `
global:
  platform: darwin
  exclude: "剩余|到期|官网"
  rename:
    - {match: " Premium$", replace: ""}
    - {match: "^HK-", replace: "香港-"}
  dedupe_by_server: true
subscriptions:
  - name: a
    url: ` + filepath.Join(dir, "a.txt") + `
  - name: b
    url: ` + filepath.Join(dir, "b.txt") + `
    exclude: ""
    include: "US|Dup"
`
	cfgPath := filepath.Join(dir, "singgen.yaml")
	os.WriteFile(cfgPath, []byte(cfgYAML), 0o600)

	cfg, err := singgen.GenerateConfigFromFile(context.Background(), cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	var tags []string
	for _, ob := range cfg.Outbounds {
		if typ, _ := ob["type"].(string); typ == "trojan" {
			tags = append(tags, ob["tag"].(string))
		}
	}
	want := "[a] 香港-01,[a] JP-01,[a] 香港-02,[b] US-01"
	if strings.Join(tags, ",") != want {
		t.Errorf("tags = %v\nwant  %s", tags, want)
	}
}

func TestNodeRulesInvalidRegexRejected(t *testing.T) {
	mc := singgen.GetDefaultMultiConfig()
	bad := "(["
	mc.Subscriptions = []singgen.SubscriptionConfig{{Name: "a", URL: singboxCheckFixture, Exclude: &bad}}
	err := mc.ValidateConfig()
	if !errors.Is(err, singgen.ErrInvalidNodeRule) || !strings.Contains(err.Error(), `"a"`) {
		t.Errorf("expected ErrInvalidNodeRule mentioning the subscription, got %v", err)
	}
}

func TestNodeRulesFilterEverything(t *testing.T) {
	mc := singgen.GetDefaultMultiConfig()
	mc.Global.Include = "^nothing-matches$"
	mc.Subscriptions = []singgen.SubscriptionConfig{{Name: "a", URL: singboxCheckFixture}}
	if _, err := singgen.GenerateConfigFromMulti(context.Background(), mc); !errors.Is(err, singgen.ErrNoValidNodes) {
		t.Errorf("expected ErrNoValidNodes when every node is filtered, got %v", err)
	}
}
