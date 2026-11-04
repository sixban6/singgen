package parser

import "github.com/sixban6/singgen/pkg/model"

// MixedParser 解析包含多种协议链接的订阅
type MixedParser struct{ base }

func (p *MixedParser) Accept(mediaTypeHint string, raw []byte) bool {
	return DetectFormat(raw) == FormatMixed
}

func (p *MixedParser) Parse(raw []byte) ([]model.Node, error) {
	nodes, _, err := ParseSubscription(raw, "", p.log)
	return nodes, err
}

func init() {
	Register(FormatMixed, func() Parser { return &MixedParser{} })
}
