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

type ShadowsocksParser struct{}

func (p *ShadowsocksParser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "ss://")
}

func (p *ShadowsocksParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))
	
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ss://") {
			continue
		}
		
		node, err := p.parseShadowsocksURL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse shadowsocks URL", zap.String("url", line), zap.Error(err))
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

func (p *ShadowsocksParser) parseShadowsocksURL(ssURL string) (*model.Node, error) {
	if !strings.HasPrefix(ssURL, "ss://") {
		return nil, fmt.Errorf("invalid shadowsocks URL")
	}
	
	u, err := url.Parse(ssURL)
	if err != nil {
		return nil, fmt.Errorf("parse URL failed: %w", err)
	}
	
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("parse port failed: %w", err)
	}
	
	var method, password string
	
	if u.User != nil {
		if u.User.Username() != "" {
			userInfo, err := util.DecodeBase64(u.User.Username())
			if err != nil {
				method = u.User.Username()
				password, _ = u.User.Password()
			} else {
				parts := strings.SplitN(string(userInfo), ":", 2)
				if len(parts) == 2 {
					method = parts[0]
					password = parts[1]
				}
			}
		}
	}
	
	node := &model.Node{
		ID:       util.MD5String(ssURL),
		Tag:      u.Fragment,
		Type:     constant.ProtocolSS,
		Addr:     u.Hostname(),
		Port:     uint16(port),
		Password: password,
		Security: model.Security{},
		Transport: model.Transport{},
		Extra:     make(map[string]any),
	}
	
	if method != "" {
		node.Extra["method"] = method
	}
	
	query := u.Query()
	if plugin := query.Get("plugin"); plugin != "" {
		node.Extra["plugin"] = plugin
		if pluginOpts := query.Get("plugin-opts"); pluginOpts != "" {
			node.Extra["plugin_opts"] = pluginOpts
		}
	}
	
	return node, nil
}

func init() {
	Register(constant.ProtocolSS, func() Parser { return &ShadowsocksParser{} })
}