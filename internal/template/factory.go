package template

import (
	"embed"
	"fmt"
	"strings"

	"github.com/sixban6/singgen/internal/util"
)

//go:embed configs
var templatesFS embed.FS

type TemplateFactory struct{}

func NewTemplateFactory() *TemplateFactory {
	return &TemplateFactory{}
}

func (f *TemplateFactory) CreateTemplate(version string) (Template, error) {
	if version == "" {
		version = "v1.14" // 默认版本
	}

	templateData, err := f.getAdaptedTemplateData(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get template data for %s: %w", version, err)
	}

	return NewEmbedTemplate(templateData)
}

func (f *TemplateFactory) GetAvailableVersions() ([]string, error) {
	return f.listEmbeddedTemplates()
}

func (f *TemplateFactory) getTemplateData(filename string) ([]byte, error) {
	version := "v1.14"
	if strings.HasPrefix(filename, "template-") && strings.HasSuffix(filename, ".yaml") {
		version = strings.TrimSuffix(strings.TrimPrefix(filename, "template-"), ".yaml")
	}
	return f.getAdaptedTemplateData(version)
}

func (f *TemplateFactory) getAdaptedTemplateData(version string) ([]byte, error) {
	// 读取唯一的基准配置模板 template.yaml
	yamlData, err := templatesFS.ReadFile("configs/template.yaml")
	if err != nil {
		// 容错回退
		yamlData, err = templatesFS.ReadFile("configs/template-v1.14.yaml")
		if err != nil {
			return nil, fmt.Errorf("baseline template not found: %w", err)
		}
	}

	jsonData, err := util.YamlToJson(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert template YAML to JSON: %w", err)
	}

	var configMap map[string]any
	if err := util.Unmarshal(jsonData, &configMap); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	// 在内存中根据目标版本进行 AST/字段转换
	adaptedMap := adaptTemplateForVersion(configMap, version)
	return util.Marshal(adaptedMap)
}

func (f *TemplateFactory) listEmbeddedTemplates() ([]string, error) {
	// 支持的模板版本列表
	return []string{"v1.12", "v1.14"}, nil
}
