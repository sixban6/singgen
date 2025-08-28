package registry

import (
	"github.com/sixban6/singgen/internal/fetcher"
	"github.com/sixban6/singgen/internal/parser"
	"github.com/sixban6/singgen/internal/renderer"
	"github.com/sixban6/singgen/internal/template"
	"github.com/sixban6/singgen/internal/transformer"
)

var (
	HTTPFetcher     fetcher.Fetcher         = fetcher.NewHTTPFetcher()
	FileFetcher     fetcher.Fetcher         = fetcher.NewFileFetcher()
	Transformer     transformer.Transformer = transformer.NewSingBoxTransformer()
	Template        template.Template       = template.NewSingBoxTemplate()
	TemplateFactory *template.TemplateFactory = template.NewTemplateFactory()
	JSONRenderer    renderer.Renderer       = renderer.NewJSONRenderer()
	YAMLRenderer    renderer.Renderer       = renderer.NewYAMLRenderer()
)

func GetFetcher(protocol string) fetcher.Fetcher {
	switch protocol {
	case "http", "https":
		return HTTPFetcher
	case "file":
		return FileFetcher
	default:
		return HTTPFetcher
	}
}

func GetRenderer(format string) renderer.Renderer {
	switch format {
	case "yaml", "yml":
		return YAMLRenderer
	default:
		return JSONRenderer
	}
}

func GetAllParsers() map[string]func() parser.Parser {
	return parser.Registry
}

func GetTemplate(version string) (template.Template, error) {
	return TemplateFactory.CreateTemplate(version)
}

func GetAvailableTemplateVersions() ([]string, error) {
	return TemplateFactory.GetAvailableVersions()
}