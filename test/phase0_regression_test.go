package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/fetcher"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

// TestHTTPFetcherHonorsContext 验证调用方 ctx 的取消能真正中断 HTTP 请求
func TestHTTPFetcherHonorsContext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := fetcher.NewHTTPFetcher().FetchContext(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error when context times out")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("fetch was not aborted by context, took %v", elapsed)
	}
}

// TestEmptyTagFallsBackToServerPort 验证空 tag 节点会回退为 server:port 且不重名
func TestEmptyTagFallsBackToServerPort(t *testing.T) {
	tmpl, err := template.NewTemplateFactory().CreateTemplate("v1.14")
	if err != nil {
		t.Fatal(err)
	}
	outbounds := []transformer.Outbound{
		{Type: "shadowsocks", Tag: "", Server: "1.2.3.4", ServerPort: 8388, Method: "aes-256-gcm", Password: "x"},
		{Type: "shadowsocks", Tag: "  ", Server: "1.2.3.4", ServerPort: 8388, Method: "aes-256-gcm", Password: "x"},
	}
	cfg := tmpl.InjectWithOptions(outbounds, config.TemplateOptions{Platform: "darwin"})

	found := map[string]bool{}
	for _, ob := range cfg.Outbounds {
		if tag, _ := ob["tag"].(string); tag != "" {
			found[tag] = true
		}
	}
	for _, want := range []string{"1.2.3.4:8388", "1.2.3.4:8388 2"} {
		if !found[want] {
			t.Errorf("expected outbound tag %q in generated config", want)
		}
	}
}
