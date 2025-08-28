package template

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/sixban6/singgen/internal/util"
)

type TemplateFactory struct{}

func NewTemplateFactory() *TemplateFactory {
	return &TemplateFactory{}
}

func (f *TemplateFactory) CreateTemplate(version string) (Template, error) {
	if version == "" {
		version = "v1.12" // 默认版本
	}

	templateFile := fmt.Sprintf("template-%s.json", version)
	templatePath, err := f.getTemplatePath(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get template path: %w", err)
	}

	return NewFileTemplate(templatePath)
}

func (f *TemplateFactory) GetAvailableVersions() ([]string, error) {
	templatesDir, err := f.getTemplatesDir()
	if err != nil {
		return nil, err
	}

	return ListTemplateFiles(templatesDir)
}

func (f *TemplateFactory) getTemplatePath(filename string) (string, error) {
	templatesDir, err := f.getTemplatesDir()
	if err != nil {
		return "", err
	}

	templatePath := filepath.Join(templatesDir, filename)
	if !util.FileExists(templatePath) {
		return "", fmt.Errorf("template file not found: %s", templatePath)
	}

	return templatePath, nil
}

func (f *TemplateFactory) getTemplatesDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get current file path")
	}

	projectRoot := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	templatesDir := filepath.Join(projectRoot, "configs")

	return templatesDir, nil
}
