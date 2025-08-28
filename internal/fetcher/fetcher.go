package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"go.uber.org/zap"
)

type Fetcher interface {
	Fetch(url string) ([]byte, error)
}

type HTTPFetcher struct {
	Client *http.Client
}

func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		Client: &http.Client{
			Timeout: constant.DefaultHTTPTimeout,
		},
	}
}

func (f *HTTPFetcher) Fetch(urlStr string) ([]byte, error) {
	// 验证URL安全性
	if err := f.validateURL(urlStr); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), constant.DefaultHTTPTimeout)
	defer cancel()

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

	if util.L != nil {
		util.L.Info("HTTP fetch successful", zap.String("url", urlStr), zap.Int("size", len(body)))
	}

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

	// 防止访问本地地址（可选的安全措施）
	hostname := strings.ToLower(parsedURL.Hostname())
	if hostname == "localhost" || 
	   hostname == "127.0.0.1" || 
	   strings.HasPrefix(hostname, "10.") ||
	   strings.HasPrefix(hostname, "192.168.") ||
	   strings.HasPrefix(hostname, "172.") {
		if util.L != nil {
			util.L.Warn("Accessing local/private network address", zap.String("url", urlStr))
		}
	}

	return nil
}

type FileFetcher struct{}

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

	if util.L != nil {
		util.L.Info("File fetch successful", zap.String("path", path), zap.Int("size", len(data)))
	}

	return data, nil
}