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

type TuicParser struct{}

func (p *TuicParser) Accept(mediaTypeHint string, raw []byte) bool {
	if strings.EqualFold(mediaTypeHint, "tuic") {
		return true
	}
	data := string(raw)
	return strings.Contains(data, "tuic://")
}

func (p *TuicParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))

	// If data is base64 encoded, decode it
	if isLikelyBase64(data) {
		if decoded, err := util.DecodeBase64(data); err == nil {
			data = string(decoded)
		}
	}

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "tuic://") {
			continue
		}

		node, err := p.parseTuicURL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse tuic URL", zap.String("url", line), zap.Error(err))
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

func (p *TuicParser) parseTuicURL(tuicURL string) (*model.Node, error) {
	if !strings.HasPrefix(tuicURL, "tuic://") {
		return nil, fmt.Errorf("invalid tuic URL")
	}

	u, err := url.Parse(tuicURL)
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

	if u.Hostname() == "" {
		return nil, fmt.Errorf("missing tuic server")
	}

	query := u.Query()
	uuid := ""
	password := ""

	if u.User != nil {
		uuid = u.User.Username()
		if pwd, ok := u.User.Password(); ok {
			password = pwd
		}
	}
	if qUUID := query.Get("uuid"); qUUID != "" {
		uuid = qUUID
	}
	if qPassword := firstQuery(query, "password", "pass"); qPassword != "" {
		password = qPassword
	}
	if password == "" && query.Get("token") != "" {
		password = query.Get("token")
	}

	// Fallback for single-token / single-credential configurations
	if uuid == "" && password != "" {
		uuid = password
	} else if uuid != "" && password == "" {
		password = uuid
	}

	if uuid == "" && password == "" {
		return nil, fmt.Errorf("missing tuic credentials")
	}

	serverName := firstQuery(query, "sni", "peer", "server_name")
	skipVerify := parseBoolQuery(query, "insecure", "allow_insecure", "allowInsecure", "skip-cert-verify")

	node := &model.Node{
		ID:       util.MD5String(tuicURL),
		Tag:      u.Fragment,
		Type:     constant.ProtocolTuic,
		Addr:     u.Hostname(),
		Port:     uint16(port),
		UUID:     uuid,
		Password: password,
		Security: model.Security{
			TLS:        true,
			SkipVerify: skipVerify,
			ServerName: serverName,
		},
		Transport: model.Transport{},
		Extra:     make(map[string]any),
	}

	if alpn := query.Get("alpn"); alpn != "" {
		node.Security.ALPN = strings.Split(alpn, ",")
	} else {
		node.Security.ALPN = []string{"h3"}
	}

	if parseBoolQuery(query, "disable_sni", "disable-sni") {
		node.Extra["disable_sni"] = true
	}

	if cc := firstQuery(query, "congestion_control", "congestion-control", "cc"); cc != "" {
		node.Extra["congestion_control"] = cc
	}

	if udpMode := firstQuery(query, "udp_relay_mode", "udp-relay-mode", "udp_mode"); udpMode != "" {
		node.Extra["udp_relay_mode"] = udpMode
	}

	if parseBoolQuery(query, "udp_over_stream", "udp-over-stream") {
		node.Extra["udp_over_stream"] = true
	}

	if parseBoolQuery(query, "zero_rtt_handshake", "zero-rtt-handshake", "reduce_rtt", "0rtt") {
		node.Extra["zero_rtt_handshake"] = true
	}

	if heartbeat := firstQuery(query, "heartbeat", "heartbeat_interval", "heartbeat-interval"); heartbeat != "" {
		node.Extra["heartbeat"] = normalizeHeartbeat(heartbeat)
	}

	if net := firstQuery(query, "network", "net"); net != "" {
		node.Extra["network"] = net
	}

	return node, nil
}

func normalizeHeartbeat(raw string) string {
	raw = strings.TrimSpace(raw)
	if num, err := strconv.Atoi(raw); err == nil {
		if num >= 1000 {
			if num%1000 == 0 {
				return fmt.Sprintf("%ds", num/1000)
			}
			return fmt.Sprintf("%dms", num)
		}
		return fmt.Sprintf("%ds", num)
	}
	return raw
}

func init() {
	Register(constant.ProtocolTuic, func() Parser { return &TuicParser{} })
}
