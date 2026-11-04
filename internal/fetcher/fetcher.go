package fetcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/logx"
	"github.com/sixban6/singgen/internal/util"
)

type Fetcher interface {
	Fetch(url string) ([]byte, error)
}

type HTTPFetcher struct {
	Client *http.Client
	// Logger 可选；为 nil 时不输出日志
	Logger *slog.Logger
}

func (f *HTTPFetcher) logger() *slog.Logger { return logx.Or(f.Logger) }

func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		Client: &http.Client{
			Timeout: constant.DefaultHTTPTimeout,
		},
	}
}

func (f *HTTPFetcher) Fetch(urlStr string) ([]byte, error) {
	return f.FetchContext(context.Background(), urlStr)
}

// FetchContext 与 Fetch 相同，但遵循调用方传入的 ctx（取消 / 超时）。
// 实际超时由 ctx 与 Client.Timeout 中较早者决定。
func (f *HTTPFetcher) FetchContext(ctx context.Context, urlStr string) ([]byte, error) {
	// 验证URL安全性
	if err := f.validateURL(urlStr); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置更真实的User-Agent来避免被阻止
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/plain,text/html,application/json,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "identity") // 避免压缩以简化处理
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	// 限制响应大小防止内存耗尽
	const maxResponseSize = 10 * 1024 * 1024 // 10MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read response body failed: %w", err)
	}

	f.logger().Info("HTTP fetch successful", "url", logx.RedactSource(urlStr), "size", len(body))

	return body, nil
}

// validateURL 验证URL的安全性
func (f *HTTPFetcher) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 只允许HTTP和HTTPS协议
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s, only http and https are allowed", scheme)
	}

	// 检查主机名不为空
	if parsedURL.Host == "" {
		return fmt.Errorf("missing host in URL")
	}

	// 访问本地 / 内网地址时给出警告（可选的安全措施）
	if isLocalHost(parsedURL.Hostname()) {
		f.logger().Warn("Accessing local/private network address", "url", logx.RedactSource(urlStr))
	}

	return nil
}

// isLocalHost 判断主机名是否为 localhost、回环、私有或链路本地地址
func isLocalHost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") {
		return true
	}
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()
}

type FileFetcher struct {
	// Logger 可选；为 nil 时不输出日志
	Logger *slog.Logger
}

func (f *FileFetcher) logger() *slog.Logger { return logx.Or(f.Logger) }

func NewFileFetcher() *FileFetcher {
	return &FileFetcher{}
}

func (f *FileFetcher) Fetch(path string) ([]byte, error) {
	if !util.FileExists(path) {
		return nil, fmt.Errorf("file does not exist: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file failed: %w", err)
	}

	f.logger().Info("File fetch successful", "path", path, "size", len(data))

	return data, nil
}
