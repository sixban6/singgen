package singgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SubscriptionInfo is the usage information a provider reports through the
// "subscription-userinfo" response header.
type SubscriptionInfo struct {
	Upload   int64     `json:"upload"`   // bytes
	Download int64     `json:"download"` // bytes
	Total    int64     `json:"total"`    // bytes, 0 if unknown/unlimited
	Expire   time.Time `json:"expire"`   // zero if not reported
}

// Used returns the consumed traffic in bytes.
func (i SubscriptionInfo) Used() int64 { return i.Upload + i.Download }

// Remaining returns the remaining traffic in bytes (0 when Total is unknown or exhausted).
func (i SubscriptionInfo) Remaining() int64 {
	if i.Total <= 0 || i.Used() >= i.Total {
		return 0
	}
	return i.Total - i.Used()
}

// Thresholds for subscription warnings.
const (
	lowTrafficRatio  = 0.10
	expireWarnWithin = 7 * 24 * time.Hour
)

// ParseSubscriptionUserInfo parses a header value such as
// "upload=123; download=456; total=1073741824; expire=1767196800".
func ParseSubscriptionUserInfo(header string) (*SubscriptionInfo, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return nil, false
	}
	info := &SubscriptionInfo{}
	found := false
	for _, part := range strings.Split(header, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || n < 0 {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "upload":
			info.Upload, found = int64(n), true
		case "download":
			info.Download, found = int64(n), true
		case "total":
			info.Total, found = int64(n), true
		case "expire":
			if n > 0 {
				info.Expire, found = time.Unix(int64(n), 0), true
			}
		}
	}
	return info, found
}

// report logs the usage and warns when traffic is low or the subscription is about to expire.
func (i SubscriptionInfo) report(logger *slog.Logger, source string, now time.Time) {
	attrs := []any{"url", source, "used", formatBytes(i.Used())}
	if i.Total > 0 {
		attrs = append(attrs, "total", formatBytes(i.Total), "remaining", formatBytes(i.Remaining()))
	}
	if !i.Expire.IsZero() {
		attrs = append(attrs, "expire", i.Expire.Format("2006-01-02"))
	}
	logger.Info("Subscription usage", attrs...)

	if i.Total > 0 && float64(i.Remaining()) <= float64(i.Total)*lowTrafficRatio {
		logger.Warn("Subscription traffic is running low", attrs...)
	}
	if !i.Expire.IsZero() {
		if !i.Expire.After(now) {
			logger.Warn("Subscription has expired", attrs...)
		} else if i.Expire.Sub(now) <= expireWarnWithin {
			logger.Warn("Subscription expires soon", attrs...)
		}
	}
}

func formatBytes(n int64) string {
	const gb = 1 << 30
	const mb = 1 << 20
	if n >= gb {
		return fmt.Sprintf("%.2fGB", float64(n)/gb)
	}
	return fmt.Sprintf("%.2fMB", float64(n)/mb)
}

// DefaultCacheDir returns the default directory for cached subscriptions
// (e.g. ~/Library/Caches/singgen or ~/.cache/singgen), or "" if unavailable.
func DefaultCacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "singgen")
}

// subscriptionCachePath derives a file name from a hash of the URL so that the
// access token embedded in subscription URLs never appears on disk in plain text.
func subscriptionCachePath(dir, source string) string {
	sum := sha256.Sum256([]byte(source))
	return filepath.Join(dir, hex.EncodeToString(sum[:16])+".sub")
}

func writeSubscriptionCache(dir, source string, data []byte) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	path := subscriptionCachePath(dir, source)
	tmp, err := os.CreateTemp(dir, ".sub-*")
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write cache file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cache file: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return fmt.Errorf("chmod cache file: %w", err)
	}
	return os.Rename(tmp.Name(), path)
}

func readSubscriptionCache(dir, source string) ([]byte, time.Time, error) {
	path := subscriptionCachePath(dir, source)
	st, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	return data, st.ModTime(), nil
}
