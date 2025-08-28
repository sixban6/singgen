package renderer

import (
	"github.com/sixban6/singgen/internal/util"
	"gopkg.in/yaml.v3"
)

type Renderer interface {
	Render(v any) ([]byte, error)
}

type JSONRenderer struct{}

func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

func (r *JSONRenderer) Render(v any) ([]byte, error) {
	return util.MarshalIndent(v)
}

type YAMLRenderer struct{}

func NewYAMLRenderer() *YAMLRenderer {
	return &YAMLRenderer{}
}

func (r *YAMLRenderer) Render(v any) ([]byte, error) {
	return yaml.Marshal(v)
}