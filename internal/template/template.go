package template

import (
	"os"
	"strings"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/internal/util"
)

type Template interface {
	Inject(outbounds []transformer.Outbound, mirrorURL string) *config.Config
	InjectWithOptions(outbounds []transformer.Outbound, options config.TemplateOptions) *config.Config
}

// SingBoxTemplate 现在直接从文件模板加载，保持向后兼容的 API
type SingBoxTemplate struct {
	fileTemplate Template
}

func NewSingBoxTemplate() *SingBoxTemplate {
	factory := NewTemplateFactory()
	// 默认使用 v1.12 模板
	fileTemplate, err := factory.CreateTemplate("v1.12")
	if err != nil {
		// 如果无法加载文件模板，创建一个空的实现避免崩溃
		// 在生产环境中应该有更好的错误处理
		fileTemplate = &EmptyTemplate{}
	}

	return &SingBoxTemplate{
		fileTemplate: fileTemplate,
	}
}

func (t *SingBoxTemplate) Inject(outbounds []transformer.Outbound, mirrorURL string) *config.Config {
	return t.fileTemplate.Inject(outbounds, mirrorURL)
}

func (t *SingBoxTemplate) InjectWithOptions(outbounds []transformer.Outbound, options config.TemplateOptions) *config.Config {
	return t.fileTemplate.InjectWithOptions(outbounds, options)
}

// EmptyTemplate 提供基本的空配置，作为最后的回退方案
type EmptyTemplate struct{}

func (t *EmptyTemplate) Inject(outbounds []transformer.Outbound, mirrorURL string) *config.Config {
	return t.InjectWithOptions(outbounds, config.TemplateOptions{MirrorURL: mirrorURL})
}

func (t *EmptyTemplate) InjectWithOptions(outbounds []transformer.Outbound, options config.TemplateOptions) *config.Config {
	// 提供最基本的配置结构，避免程序崩溃
	var outboundsMap []map[string]any

	// 转换代理节点
	for _, outbound := range outbounds {
		tag := outbound.Tag
		if options.RemoveEmoji {
			tag = util.CleanNodeTag(tag, true)
		}

		outboundMap := map[string]any{
			"type":        outbound.Type,
			"tag":         tag,
			"server":      outbound.Server,
			"server_port": outbound.ServerPort,
		}

		if outbound.UUID != "" {
			outboundMap["uuid"] = outbound.UUID
		}
		if outbound.Password != "" {
			outboundMap["password"] = outbound.Password
		}
		if outbound.Method != "" {
			outboundMap["method"] = outbound.Method
		}
		if len(outbound.Transport) > 0 {
			outboundMap["transport"] = outbound.Transport
		}
		if len(outbound.TLS) > 0 {
			outboundMap["tls"] = outbound.TLS
		}
		if len(outbound.Multiplex) > 0 {
			outboundMap["multiplex"] = outbound.Multiplex
		}

		outboundsMap = append(outboundsMap, outboundMap)
	}

	// 添加基本的直连出站
	outboundsMap = append(outboundsMap, map[string]any{
		"tag":  "DirectConn",
		"type": "direct",
	})

	// 构建实验性配置
	experimental := map[string]any{
		"cache_file": map[string]any{
			"enabled": true,
			"path":    "/etc/sing-box/cache.db",
		},
	}

	// 如果有外部控制器配置，添加 clash_api
	if options.ExternalController != "" {
		experimental["clash_api"] = map[string]any{
			"external_controller": options.ExternalController,
			"default_mode":        "rule",
			"secret":              "",
		}
	}

	// 构建DNS配置
	dnsServer := "114.114.114.114"
	if options.DNSLocalServer != "" {
		dnsServer = options.DNSLocalServer
	}

	dns := map[string]any{
		"servers": []map[string]any{
			{"tag": "dns_local", "type": "udp", "server": dnsServer},
		},
		"rules": []map[string]any{},
		"final": "dns_local",
	}

	// 如果有客户端子网配置，添加到DNS配置中
	if options.ClientSubnet != "" {
		dns["client_subnet"] = options.ClientSubnet
	}

	return &config.Config{
		Log: map[string]any{
			"disabled":  false,
			"level":     "warn",
			"timestamp": true,
		},
		Experimental: experimental,
		DNS:          dns,
		Inbounds: []map[string]any{
			{
				"type":        "tproxy",
				"tag":         "tproxy-in",
				"listen":      "::",
				"listen_port": 7895,
				"udp_timeout": "5m",
			},
		},
		Outbounds: outboundsMap,
		Route: map[string]any{
			"auto_detect_interface": true,
			"final":                 "DirectConn",
			"rules":                 []map[string]any{},
		},
	}
}

func ListTemplateFiles(dir string) ([]string, error) {
	var versions []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, "template-") && strings.HasSuffix(name, ".json") {
			// 提取版本号: template-v1.12.json -> v1.12
			version := strings.TrimPrefix(name, "template-")
			version = strings.TrimSuffix(version, ".json")
			versions = append(versions, version)
		}
	}

	return versions, nil
}
