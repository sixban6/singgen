package parser

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/logx"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
)

type TrojanParser struct{ base }

func (p *TrojanParser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "trojan://")
}

func (p *TrojanParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "trojan://") {
			continue
		}

		node, err := p.parseTrojanURL(line)
		if err != nil {
			p.logger().Warn("Failed to parse trojan URL", "node", logx.RedactURL(line), "error", logx.Err(err))
			continue
		}

		nodes = append(nodes, *node)
	}

	if len(nodes) == 0 {
		return nil, constant.ErrParseFailed
	}

	return nodes, nil
}

func (p *TrojanParser) parseTrojanURL(trojanURL string) (*model.Node, error) {
	if !strings.HasPrefix(trojanURL, "trojan://") {
		return nil, fmt.Errorf("invalid trojan URL")
	}

	u, err := url.Parse(trojanURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL failed: %w", err)
	}

	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port failed: %w", err)
	}

	query := u.Query()

	node := &model.Node{
		ID:       util.MD5String(trojanURL),
		Tag:      u.Fragment,
		Type:     constant.ProtocolTrojan,
		Addr:     u.Hostname(),
		Port:     uint16(port),
		Password: u.User.Username(),
		Security: model.Security{
			TLS:        true,
			SkipVerify: parseBoolQuery(query, "allowInsecure", "insecure", "skip-cert-verify"),
			ServerName: firstQuery(query, "sni", "peer"),
		},
		Transport: model.Transport{
			Net:  query.Get("type"),
			Host: query.Get("host"),
			Path: query.Get("path"),
		},
		Extra: make(map[string]any),
	}

	if alpn := query.Get("alpn"); alpn != "" {
		node.Security.ALPN = strings.Split(alpn, ",")
	}

	if security := query.Get("security"); security != "" {
		node.Extra["security"] = security
	}

	if fp := query.Get("fp"); fp != "" {
		node.Extra["fingerprint"] = fp
	}

	if serviceName := query.Get("serviceName"); serviceName != "" {
		node.Extra["service_name"] = serviceName
	}

	return node, nil
}

func init() {
	Register(constant.ProtocolTrojan, func() Parser { return &TrojanParser{} })
}
