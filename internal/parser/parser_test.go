package parser

import (
	"encoding/base64"
	"strings"
	"testing"
)

const (
	linkTrojan = "trojan://pw@1.1.1.1:443?sni=a.com#T1"
	linkSS     = "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@2.2.2.2:8388#S1"
	linkHY2    = "hy2://pw@3.3.3.3:443?sni=h.com#H1"
)

func tags(t *testing.T, raw []byte, only string) ([]string, string) {
	t.Helper()
	nodes, format, err := ParseSubscription(raw, only, nil)
	if err != nil {
		t.Fatalf("ParseSubscription: %v", err)
	}
	var out []string
	for _, n := range nodes {
		out = append(out, n.Tag)
	}
	return out, format
}

func TestParseSubscriptionPlainMixedKeepsOrder(t *testing.T) {
	raw := []byte(strings.Join([]string{linkHY2, "  ", linkTrojan, "# comment", linkSS}, "\n"))
	got, format := tags(t, raw, "")
	if format != FormatMixed {
		t.Errorf("format = %s", format)
	}
	if strings.Join(got, ",") != "H1,T1,S1" {
		t.Errorf("order = %v", got)
	}
}

func TestParseSubscriptionBase64Variants(t *testing.T) {
	plain := linkTrojan + "\n" + linkTrojan[:len(linkTrojan)-2] + "T2"
	std := base64.StdEncoding.EncodeToString([]byte(plain))
	urlSafe := base64.RawURLEncoding.EncodeToString([]byte(plain + "\n" + "trojan://p?w@1.1.1.2:443#T3"))

	// 每 20 字符换行的 base64
	var wrapped strings.Builder
	for i := 0; i < len(std); i += 20 {
		end := min(i+20, len(std))
		wrapped.WriteString(std[i:end] + "\n")
	}

	for name, raw := range map[string]string{"std": std, "wrapped": wrapped.String(), "urlsafe": urlSafe} {
		t.Run(name, func(t *testing.T) {
			got, format := tags(t, []byte(raw), "")
			if format != "trojan" || len(got) < 2 || got[0] != "T1" || got[1] != "T2" {
				t.Errorf("format=%s tags=%v", format, got)
			}
		})
	}
}

func TestParseSubscriptionOnlyProtocol(t *testing.T) {
	raw := []byte(strings.Join([]string{linkHY2, linkTrojan, linkSS}, "\n"))
	got, _ := tags(t, raw, "shadowsocks")
	if strings.Join(got, ",") != "S1" {
		t.Errorf("filtered = %v", got)
	}
	if _, _, err := ParseSubscription(raw, "nope", nil); err == nil {
		t.Error("expected error for unknown protocol filter")
	}
}

func TestParseSubscriptionNoNodes(t *testing.T) {
	if _, format, err := ParseSubscription([]byte("hello\nworld"), "", nil); err == nil || format != FormatUnknown {
		t.Errorf("expected ErrParseFailed and unknown format, got format=%s err=%v", format, err)
	}
}
