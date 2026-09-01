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

	// 端口跳跃：mport 为通行约定（部分机场用 ports），值形如 20000-30000 或 443,8443
	serverPorts := query.Get("mport")
	if serverPorts == "" {
		serverPorts = query.Get("ports")
	}
	if serverPorts != "" {
		node.Extra["server_ports"] = normalizeHy2ServerPorts(serverPorts)
		// 跳跃间隔，sing-box 默认 30s；hop_interval_max 为 1.14 新增（随机化跳跃间隔，更难被识别）
		if hopInterval := query.Get("hop_interval"); hopInterval != "" {
			node.Extra["hop_interval"] = hopInterval
		}
		if hopIntervalMax := query.Get("hop_interval_max"); hopIntervalMax != "" {
			node.Extra["hop_interval_max"] = hopIntervalMax
		}
	}

	// 解析带宽限制参数
	if upMbps := query.Get("up_mbps"); upMbps != "" {
		node.Extra["up_mbps"] = upMbps
	} else {
		node.Extra["up_mbps"] = "25" // 默认上行 25 Mbps
	}

	if downMbps := query.Get("down_mbps"); downMbps != "" {
		node.Extra["down_mbps"] = downMbps
	} else {
		node.Extra["down_mbps"] = "300" // 默认下行 300 Mbps
	}

	return node, nil
}

func init() {
	Register(constant.ProtocolHysteria2, func() Parser { return &Hysteria2Parser{} })
}

// normalizeHy2ServerPorts 将订阅链接的端口跳跃表示规范化为 sing-box server_ports 格式：
// sing-box 仅接受 "起始:结束" 范围语法，单端口需展开为 N:N：
// "20000-30000" -> ["20000:30000"]，"443" -> ["443:443"]，
// "443,8443,2053" -> ["443:443","8443:8443","2053:2053"]，"443,20000-30000" -> ["443:443","20000:30000"]
func normalizeHy2ServerPorts(raw string) []string {
	tokens := strings.Split(raw, ",")
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		switch {
		case strings.Contains(token, ":"):
			// 已是范围语法，保持原样
		case strings.Contains(token, "-"):
			token = strings.ReplaceAll(token, "-", ":")
		default:
			// 单端口展开为 N:N 范围（sing-box 不接受裸端口）
			token = token + ":" + token
		}
		result = append(result, token)
	}
	return result
}
