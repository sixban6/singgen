package fetcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
	// MaxAttempts 最大尝试次数（含首次），<=0 时使用 DefaultMaxAttempts
	MaxAttempts int
	// RetryBackoff 首次重试前的等待时间，之后逐次翻倍；<=0 时使用 DefaultRetryBackoff
	RetryBackoff time.Duration
}

const (
	// DefaultMaxAttempts 默认最多尝试 3 次（仅网络错误、5xx、429 会重试）
	DefaultMaxAttempts = 3
	// DefaultRetryBackoff 默认首次重试等待时间
	DefaultRetryBackoff = 500 * time.Millisecond
	// maxResponseSize 限制响应大小防止内存耗尽
	maxResponseSize = 10 * 1024 * 1024 // 10MB
)

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
// 单次请求超时由 Client.Timeout 决定；失败可重试时最多尝试 MaxAttempts 次。
func (f *HTTPFetcher) FetchContext(ctx context.Context, urlStr string) ([]byte, error) {
	body, _, err := f.FetchWithHeader(ctx, urlStr)
	return body, err
}

// retryableError 标记可以重试的失败（网络错误、5xx、429）
type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

// FetchWithHeader 拉取订阅并返回响应头（用于读取 subscription-userinfo 等信息）。
// 网络错误、5xx 与 429 会按指数退避重试；ctx 取消时立即返回。
func (f *HTTPFetcher) FetchWithHeader(ctx context.Context, urlStr string) ([]byte, http.Header, error) {
	// 验证URL安全性
	if err := f.validateURL(urlStr); err != nil {
		return nil, nil, err
	}

	attempts := f.MaxAttempts
	if attempts <= 0 {
		attempts = DefaultMaxAttempts
	}
	backoff := f.RetryBackoff
	if backoff <= 0 {
		backoff = DefaultRetryBackoff
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		body, header, err := f.fetchOnce(ctx, urlStr)
		if err == nil {
			f.logger().Info("HTTP fetch successful", "url", logx.RedactSource(urlStr), "size", len(body), "attempt", attempt)
			return body, header, nil
		}
		lastErr = err

		var retryable *retryableError
		if !errors.As(err, &retryable) || ctx.Err() != nil || attempt == attempts {
			break
		}
		f.logger().Warn("HTTP fetch failed, retrying",
			"url", logx.RedactSource(urlStr), "attempt", attempt, "wait", backoff, "error", logx.ErrSource(err))
		select {
		case <-ctx.Done():
			return nil, nil, fmt.Errorf("http request canceled: %w", ctx.Err())
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	return nil, nil, lastErr
}

func (f *HTTPFetcher) fetchOnce(ctx context.Context, urlStr string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request failed: %w", err)
	}

	// 设置更真实的User-Agent来避免被阻止
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/plain,text/html,application/json,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// 不手动设置 Accept-Encoding：由 net/http 自动协商 gzip 并透明解压
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := f.Client.Do(req)
	if err != nil {
		err = fmt.Errorf("http request failed: %w", err)
		if ctx.Err() != nil {
			return nil, nil, err
		}
		return nil, nil, &retryableError{err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("http status %d", resp.StatusCode)
		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			return nil, nil, &retryableError{err}
		}
		return nil, nil, err
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, nil, &retryableError{fmt.Errorf("read response body failed: %w", err)}
	}
	return body, resp.Header, nil
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
