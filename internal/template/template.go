package template

import (
	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/transformer"
)

type Template interface {
	Inject(outbounds []transformer.Outbound, mirrorURL string) *config.Config
	InjectWithOptions(outbounds []transformer.Outbound, options config.TemplateOptions) *config.Config
}
