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

type VlessParser struct{}

func (p *VlessParser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "vless://")
}

func (p *VlessParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))
	
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "vless://") {
			continue
		}
		
		node, err := p.parseVlessURL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse vless URL", zap.String("url", line), zap.Error(err))
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

func (p *VlessParser) parseVlessURL(vlessURL string) (*model.Node, error) {
	if !strings.HasPrefix(vlessURL, "vless://") {
		return nil, fmt.Errorf("invalid vless URL")
	}
	
	u, err := url.Parse(vlessURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL failed: %w", err)
	}
	
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port failed: %w", err)
	}
	
	query := u.Query()
	
	node := &model.Node{
		ID:   util.MD5String(vlessURL),
		Tag:  u.Fragment,
		Type: constant.ProtocolVLESS,
		Addr: u.Hostname(),
		Port: uint16(port),
		UUID: u.User.Username(),
		Security: model.Security{
			TLS:        query.Get("security") == "tls",
			SkipVerify: query.Get("allowInsecure") == "1",
			ServerName: query.Get("sni"),
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
	
	if encryption := query.Get("encryption"); encryption != "" {
		node.Extra["encryption"] = encryption
	}
	
	if flow := query.Get("flow"); flow != "" {
		node.Extra["flow"] = flow
	}
	
	return node, nil
}

func init() {
	Register(constant.ProtocolVLESS, func() Parser { return &VlessParser{} })
}