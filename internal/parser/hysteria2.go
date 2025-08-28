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

type Hysteria2Parser struct{}

func (p *Hysteria2Parser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "hysteria2://") || strings.Contains(data, "hy2://")
}

func (p *Hysteria2Parser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "hysteria2://") && !strings.HasPrefix(line, "hy2://") {
			continue
		}

		node, err := p.parseHysteria2URL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse hysteria2 URL", zap.String("url", line), zap.Error(err))
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

func (p *Hysteria2Parser) parseHysteria2URL(hysteria2URL string) (*model.Node, error) {
	if !strings.HasPrefix(hysteria2URL, "hysteria2://") && !strings.HasPrefix(hysteria2URL, "hy2://") {
		return nil, fmt.Errorf("invalid hysteria2 URL")
	}

	u, err := url.Parse(hysteria2URL)
	if err != nil {
		return nil, fmt.Errorf("parse URL failed: %w", err)
	}

	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port failed: %w", err)
	}

	query := u.Query()

	node := &model.Node{
		ID:       util.MD5String(hysteria2URL),
		Tag:      u.Fragment,
		Type:     constant.ProtocolHysteria2,
		Addr:     u.Hostname(),
		Port:     uint16(port),
		Password: u.User.Username(),
		Security: model.Security{
			TLS:        true,
			SkipVerify: query.Get("insecure") == "1",
			ServerName: query.Get("sni"),
		},
		Transport: model.Transport{},
		Extra:     make(map[string]any),
	}

	if alpn := query.Get("alpn"); alpn != "" {
		node.Security.ALPN = strings.Split(alpn, ",")
	} else {
		node.Security.ALPN = []string{"h3"}
	}

	if obfs := query.Get("obfs"); obfs != "" {
		node.Extra["obfs"] = map[string]string{
			"type":     obfs,
			"password": query.Get("obfs-password"),
		}
	}

	return node, nil
}

func init() {
	Register(constant.ProtocolHysteria2, func() Parser { return &Hysteria2Parser{} })
}
