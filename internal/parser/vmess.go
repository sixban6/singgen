package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
	"go.uber.org/zap"
)

type vmessDTO struct {
	V    string `json:"v"`
	Add  string `json:"add"`
	Port any    `json:"port"`
	ID   string `json:"id"`
	Aid  any    `json:"aid"`
	Net  string `json:"net"`
	Host string `json:"host"`
	Path string `json:"path"`
	TLS  string `json:"tls"`
	Ps   string `json:"ps"`
	Scy  string `json:"scy"`
	Sni  string `json:"sni"`
	Alpn string `json:"alpn"`
}

type VmessParser struct{}

func (p *VmessParser) Accept(mediaTypeHint string, raw []byte) bool {
	data := string(raw)
	return strings.Contains(data, "vmess://")
}

func (p *VmessParser) Parse(raw []byte) ([]model.Node, error) {
	var nodes []model.Node
	data := strings.TrimSpace(string(raw))
	
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "vmess://") {
			continue
		}
		
		node, err := p.parseVmessURL(line)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse vmess URL", zap.String("url", line), zap.Error(err))
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

func (p *VmessParser) parseVmessURL(vmessURL string) (*model.Node, error) {
	if !strings.HasPrefix(vmessURL, "vmess://") {
		return nil, fmt.Errorf("invalid vmess URL")
	}
	
	encoded := strings.TrimPrefix(vmessURL, "vmess://")
	decoded, err := util.DecodeBase64(encoded)
	if err != nil {
		return nil, fmt.Errorf("base64 decode failed: %w", err)
	}
	
	var dto vmessDTO
	if err := util.Unmarshal(decoded, &dto); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}
	
	port, err := parsePort(dto.Port)
	if err != nil {
		return nil, fmt.Errorf("parse port failed: %w", err)
	}
	
	node := &model.Node{
		ID:       util.MD5String(vmessURL),
		Tag:      dto.Ps,
		Type:     constant.ProtocolVMess,
		Addr:     dto.Add,
		Port:     port,
		UUID:     dto.ID,
		Security: model.Security{
			TLS:        dto.TLS == "tls",
			SkipVerify: true,
			ServerName: dto.Sni,
		},
		Transport: model.Transport{
			Net:  dto.Net,
			Host: dto.Host,
			Path: dto.Path,
		},
		Extra: make(map[string]any),
	}
	
	if dto.Alpn != "" {
		node.Security.ALPN = strings.Split(dto.Alpn, ",")
	}
	
	if dto.Scy != "" {
		node.Extra["security"] = dto.Scy
	}
	
	if aid, err := parseAlterId(dto.Aid); err == nil {
		node.Extra["alter_id"] = aid
	}
	
	return node, nil
}

func parsePort(port any) (uint16, error) {
	switch v := port.(type) {
	case string:
		p, err := strconv.ParseUint(v, 10, 16)
		return uint16(p), err
	case float64:
		return uint16(v), nil
	case int:
		return uint16(v), nil
	default:
		return 0, fmt.Errorf("invalid port type: %T", port)
	}
}

func parseAlterId(aid any) (int, error) {
	switch v := aid.(type) {
	case string:
		return strconv.Atoi(v)
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("invalid alter_id type: %T", aid)
	}
}

func init() {
	Register(constant.ProtocolVMess, func() Parser { return &VmessParser{} })
}