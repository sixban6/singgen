package parser

import (
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
	"go.uber.org/zap"
)

type MixedParser struct{}

func (p *MixedParser) Accept(mediaTypeHint string, raw []byte) bool {
	format := DetectFormat(raw)
	return format == "mixed"
}

func (p *MixedParser) Parse(raw []byte) ([]model.Node, error) {
	var allNodes []model.Node
	data := strings.TrimSpace(string(raw))
	
	// Try to decode base64 if it looks like base64 data
	if isLikelyBase64(data) {
		if decoded, err := util.DecodeBase64(data); err == nil {
			data = string(decoded)
		}
	}
	
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		var parser Parser
		
		if strings.HasPrefix(line, "vmess://") {
			parser = &VmessParser{}
		} else if strings.HasPrefix(line, "vless://") {
			parser = &VlessParser{}
		} else if strings.HasPrefix(line, "trojan://") {
			parser = &TrojanParser{}
		} else if strings.HasPrefix(line, "hysteria2://") || strings.HasPrefix(line, "hy2://") {
			parser = &Hysteria2Parser{}
		} else if strings.HasPrefix(line, "ss://") {
			parser = &ShadowsocksParser{}
		} else {
			continue
		}
		
		nodes, err := parser.Parse([]byte(line))
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to parse line", zap.String("line", line), zap.Error(err))
			}
			continue
		}
		
		allNodes = append(allNodes, nodes...)
	}
	
	if len(allNodes) == 0 {
		return nil, constant.ErrParseFailed
	}
	
	return allNodes, nil
}

func init() {
	Register("mixed", func() Parser { return &MixedParser{} })
}