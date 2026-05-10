package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
	"go.uber.org/zap"
)

type AnyTLSParser struct{}

func (p *AnyTLSParser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "anytls://")
}

func (p *AnyTLSParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "anytls://") {
			continue
		}

		node, err := p.parseAnyTLSURL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse anytls URL", zap.String("url", line), zap.Error(err))
			}
			continue
		}

		nodes = append(nodes, *node)
	}

	if len(nodes) == 0 {
		return nil, constant.ErrParseFailed
	}

	return nodes, nil
}

func (p *AnyTLSParser) parseAnyTLSURL(anyTLSURL string) (*model.Node, error) {
	if !strings.HasPrefix(anyTLSURL, "anytls://") {
		return nil, fmt.Errorf("invalid anytls URL")
	}

	u, err := url.Parse(anyTLSURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL failed: %w", err)
	}

	port := uint64(443)
	if rawPort := u.Port(); rawPort != "" {
		port, err = strconv.ParseUint(rawPort, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("parse port failed: %w", err)
		}
	}

	query := u.Query()
	serverName := firstQuery(query, "sni", "peer")
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}

	node := &model.Node{
		ID:       util.MD5String(anyTLSURL),
		Tag:      u.Fragment,
		Type:     constant.ProtocolAnyTLS,
		Addr:     u.Hostname(),
		Port:     uint16(port),
		Password: password,
		Security: model.Security{
			TLS:        true,
			SkipVerify: parseBoolQuery(query, "insecure", "allowInsecure", "skip-cert-verify"),
			ServerName: serverName,
		},
		Transport: model.Transport{},
		Extra:     make(map[string]any),
	}

	if node.Addr == "" {
		return nil, fmt.Errorf("missing anytls server")
	}
	if node.Password == "" {
		return nil, fmt.Errorf("missing anytls password")
	}

	if alpn := query.Get("alpn"); alpn != "" {
		node.Security.ALPN = strings.Split(alpn, ",")
	}

	copyAnyTLSOptionalField(query, node.Extra, "idle_session_check_interval", "idle-session-check-interval")
	copyAnyTLSOptionalField(query, node.Extra, "idle_session_timeout", "idle-session-timeout")
	copyAnyTLSOptionalField(query, node.Extra, "min_idle_session", "min-idle-session")

	return node, nil
}

func firstQuery(query url.Values, keys ...string) string {
	for _, key := range keys {
		if value := query.Get(key); value != "" {
			return value
		}
	}
	return ""
}

func parseBoolQuery(query url.Values, keys ...string) bool {
	for _, key := range keys {
		switch strings.ToLower(query.Get(key)) {
		case "1", "true", "yes":
			return true
		}
	}
	return false
}

func copyAnyTLSOptionalField(query url.Values, extra map[string]any, canonicalKey string, aliases ...string) {
	keys := append([]string{canonicalKey}, aliases...)
	if value := firstQuery(query, keys...); value != "" {
		extra[canonicalKey] = value
	}
}

func init() {
	Register(constant.ProtocolAnyTLS, func() Parser { return &AnyTLSParser{} })
}
