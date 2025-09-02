package template

import (
	"embed"
	"fmt"
	"github.com/sixban6/singgen/internal/util"
	"io/fs"
	"strings"
)

//go:embed configs
var templatesFS embed.FS

type TemplateFactory struct{}

func NewTemplateFactory() *TemplateFactory {
	return &TemplateFactory{}
}

func (f *TemplateFactory) CreateTemplate(version string) (Template, error) {
	if version == "" {
		version = "v1.12" // 默认版本
	}

	templateFile := fmt.Sprintf("template-%s.yaml", version)
	templateData, err := f.getTemplateData(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get template data: %w", err)
	}

	return NewEmbedTemplate(templateData)
}

func (f *TemplateFactory) GetAvailableVersions() ([]string, error) {
	return f.listEmbeddedTemplates()
}

func (f *TemplateFactory) getTemplateData(filename string) ([]byte, error) {
	filePath := "configs/" + filename
	yamlData, err := templatesFS.ReadFile(filePath)
	data, err := util.YamlToJson(yamlData)
	if err != nil {
		return nil, fmt.Errorf("template file not found: %s", filename)
	}
	return data, nil
}

func (f *TemplateFactory) listEmbeddedTemplates() ([]string, error) {
	var versions []string
	err := fs.WalkDir(templatesFS, "configs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasPrefix(d.Name(), "template-") && strings.HasSuffix(d.Name(), ".json") {
			version := strings.TrimPrefix(d.Name(), "template-")
			version = strings.TrimSuffix(version, ".json")
			versions = append(versions, version)
		}
		return nil
	})
	return versions, err
}
