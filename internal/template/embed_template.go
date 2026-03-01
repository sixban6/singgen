package template

import (
	"fmt"
	"strings"

	"github.com/sixban6/singgen/internal/config"
	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/platform"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/internal/util"
	"go.uber.org/zap"
)

type EmbedTemplate struct {
	rawData   []byte
	rawConfig map[string]any
	processor *TemplateProcessor
}

func NewEmbedTemplate(data []byte) (*EmbedTemplate, error) {
	var config map[string]any
	if err := util.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return &EmbedTemplate{
		rawData:   data,
		rawConfig: config,
		processor: NewTemplateProcessor(),
	}, nil
}

func (t *EmbedTemplate) Inject(outbounds []transformer.Outbound, mirrorURL string) *config.Config {
	return t.InjectWithOptions(outbounds, config.TemplateOptions{MirrorURL: mirrorURL})
}

func (t *EmbedTemplate) InjectWithOptions(outbounds []transformer.Outbound, options config.TemplateOptions) *config.Config {
	// 深拷贝模板配置
	configData := t.deepCopyConfig()

	// 处理emoji移除（如果启用）
	processedOutbounds := outbounds
	if options.RemoveEmoji {
		processedOutbounds = t.processEmojiRemoval(outbounds)
	}

	// 处理镜像URL占位符
	if options.MirrorURL != "" {
		t.replaceMirrorURL(configData, options.MirrorURL)
	} else {
		t.replaceMirrorURL(configData, "")
	}

	// 处理 Tailscale 配置
	t.handleTailscaleConfig(configData, options)

	// 注入外部控制器配置
	if options.ExternalController != "" {
		t.injectExternalController(configData, options.ExternalController)
	}

	// 注入客户端子网配置
	if options.ClientSubnet != "" {
		t.injectClientSubnet(configData, options.ClientSubnet)
	}

	// 注入DNS本地服务器配置
	if options.DNSLocalServer != "" {
		t.injectDNSLocalServer(configData, options.DNSLocalServer)
	}

	// 处理 {all} 占位符和过滤规则
	t.processor.ProcessAllPlaceholders(configData, processedOutbounds)

	// 注入代理节点
	t.injectProxyOutbounds(configData, processedOutbounds)

	// 创建配置对象
	result := &config.Config{
		Log:          t.convertToMap(configData["log"]),
		Experimental: t.convertToMap(configData["experimental"]),
		DNS:          t.convertToMap(configData["dns"]),
		Inbounds:     t.convertToMapArray(configData["inbounds"]),
		Outbounds:    t.convertToMapArray(configData["outbounds"]),
		Route:        t.convertToMap(configData["route"]),
		Certificate:  t.convertToMap(configData["certificate"]),
		Endpoints:    t.convertToMapArray(configData["endpoints"]),
	}

	// 应用平台适配（使用嵌入的平台配置）
	if err := t.applyEmbedPlatformAdaptation(result, options); err != nil {
		if util.L != nil {
			util.L.Warn("Failed to apply platform adaptation", zap.Error(err))
		}
	}

	return result
}

func (t *EmbedTemplate) deepCopyConfig() map[string]any {
	data, _ := util.Marshal(t.rawConfig)
	var copy map[string]any
	util.Unmarshal(data, &copy)
	return copy
}

func (t *EmbedTemplate) replaceMirrorURL(config map[string]any, mirrorURL string) {
	t.walkAndReplace(config, func(s string) string {
		if mirrorURL == "" {
			return strings.Replace(s, constant.MirrorURLPlaceholder+"/", "", -1)
		}
		return strings.Replace(s, constant.MirrorURLPlaceholder, mirrorURL, -1)
	})
}

func (t *EmbedTemplate) walkAndReplace(obj any, replacer func(string) string) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			switch val := value.(type) {
			case string:
				v[key] = replacer(val)
			case map[string]any, []any:
				t.walkAndReplace(val, replacer)
			}
		}
	case []any:
		for i, item := range v {
			switch val := item.(type) {
			case string:
				v[i] = replacer(val)
			case map[string]any, []any:
				t.walkAndReplace(val, replacer)
			}
		}
	}
}

func (t *EmbedTemplate) injectProxyOutbounds(config map[string]any, outbounds []transformer.Outbound) {
	// 处理类型转换
	outboundsInterface := config["outbounds"]
	var existingOutbounds []map[string]any

	switch v := outboundsInterface.(type) {
	case []map[string]any:
		existingOutbounds = v
	case []any:
		existingOutbounds = make([]map[string]any, len(v))
		for i, item := range v {
			if m, ok := item.(map[string]any); ok {
				existingOutbounds[i] = m
			}
		}
	default:
		return // 无法处理的类型
	}

	// 找到插入位置 (在 DirectConn 之前)
	insertIndex := len(existingOutbounds) - 1
	for i, outbound := range existingOutbounds {
		if tag, ok := outbound["tag"].(string); ok && tag == "DirectConn" {
			insertIndex = i
			break
		}
	}

	// 转换代理节点为 map[string]any 格式
	var proxyOutbounds []map[string]any
	for _, outbound := range outbounds {
		outboundMap := t.transformerOutboundToMap(outbound)
		proxyOutbounds = append(proxyOutbounds, outboundMap)
	}

	// 插入代理节点
	newOutbounds := make([]map[string]any, 0, len(existingOutbounds)+len(proxyOutbounds))
	newOutbounds = append(newOutbounds, existingOutbounds[:insertIndex]...)
	newOutbounds = append(newOutbounds, proxyOutbounds...)
	newOutbounds = append(newOutbounds, existingOutbounds[insertIndex:]...)

	config["outbounds"] = newOutbounds
}

func (t *EmbedTemplate) transformerOutboundToMap(outbound transformer.Outbound) map[string]any {
	result := map[string]any{
		"type":        outbound.Type,
		"tag":         outbound.Tag,
		"server":      outbound.Server,
		"server_port": outbound.ServerPort,
	}

	if outbound.UUID != "" {
		result["uuid"] = outbound.UUID
	}
	if outbound.Password != "" {
		result["password"] = outbound.Password
	}
	if outbound.Method != "" {
		result["method"] = outbound.Method
	}
	if outbound.Flow != "" {
		result["flow"] = outbound.Flow
	}
	if len(outbound.Transport) > 0 {
		result["transport"] = outbound.Transport
	}
	if len(outbound.TLS) > 0 {
		result["tls"] = outbound.TLS
	}
	if len(outbound.Multiplex) > 0 {
		result["multiplex"] = outbound.Multiplex
	}
	if outbound.UpMbps > 0 {
		result["up_mbps"] = outbound.UpMbps
	}
	if outbound.DownMbps > 0 {
		result["down_mbps"] = outbound.DownMbps
	}

	return result
}

func (t *EmbedTemplate) convertToMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func (t *EmbedTemplate) convertToMapArray(v any) []map[string]any {
	if arr, ok := v.([]map[string]any); ok {
		return arr
	}

	if arr, ok := v.([]any); ok {
		result := make([]map[string]any, len(arr))
		for i, item := range arr {
			if m, ok := item.(map[string]any); ok {
				result[i] = m
			} else {
				result[i] = map[string]any{}
			}
		}
		return result
	}

	return []map[string]any{}
}

func (t *EmbedTemplate) injectExternalController(config map[string]any, externalController string) {
	experimental, ok := config["experimental"].(map[string]any)
	if !ok {
		experimental = make(map[string]any)
		config["experimental"] = experimental
	}

	clashAPI, ok := experimental["clash_api"].(map[string]any)
	if !ok {
		clashAPI = make(map[string]any)
		experimental["clash_api"] = clashAPI
	}

	// 只更新external_controller字段，保留其他字段
	clashAPI["external_controller"] = externalController
}

func (t *EmbedTemplate) injectClientSubnet(config map[string]any, clientSubnet string) {
	// Use walkAndReplace to replace all client_subnet occurrences
	t.walkAndReplace(config, func(s string) string {
		if s == "223.5.5.5/32" { // Replace the default client subnet
			return clientSubnet
		}
		return s
	})

	// Also ensure DNS section has the client_subnet field
	dns, ok := config["dns"].(map[string]any)
	if ok {
		dns["client_subnet"] = clientSubnet
	}
}

// processEmojiRemoval 处理emoji移除逻辑
func (t *EmbedTemplate) processEmojiRemoval(outbounds []transformer.Outbound) []transformer.Outbound {
	processedOutbounds := make([]transformer.Outbound, len(outbounds))
	for i, outbound := range outbounds {
		processedOutbounds[i] = outbound
		processedOutbounds[i].Tag = util.CleanNodeTag(outbound.Tag, true)
	}
	return processedOutbounds
}

// injectDNSLocalServer 注入DNS本地服务器配置
func (t *EmbedTemplate) injectDNSLocalServer(config map[string]any, dnsLocalServer string) {
	// 使用walkAndReplace替换所有DNS服务器地址
	t.walkAndReplace(config, func(s string) string {
		if s == "114.114.114.114" { // 替换默认DNS服务器
			return dnsLocalServer
		}
		return s
	})

	// 确保DNS配置中的servers字段包含本地DNS服务器
	dns, ok := config["dns"].(map[string]any)
	if !ok {
		return
	}

	servers, ok := dns["servers"].([]any)
	if !ok {
		return
	}

	// 更新DNS服务器配置
	for _, serverAny := range servers {
		if server, ok := serverAny.(map[string]any); ok {
			if tag, tagOk := server["tag"].(string); tagOk && tag == "dns_local" {
				server["server"] = dnsLocalServer
				break
			}
		}
	}
}

// applyEmbedPlatformAdaptation 为嵌入模板应用平台适配
func (t *EmbedTemplate) applyEmbedPlatformAdaptation(config *config.Config, options config.TemplateOptions) error {
	// 如果没有指定平台，默认使用linux
	platformType := options.Platform
	if platformType == "" {
		platformType = "linux"
	}

	// 通过工厂获取平台适配器
	adapter, err := platform.CreateAdapterByString(platformType, "")
	if err != nil {
		return fmt.Errorf("unsupported platform: %s", platformType)
	}

	// 使用适配器获取配置文件名
	platformConfigFile := fmt.Sprintf("configs/platform/%s", adapter.GetConfigFileName())

	platformData, err := templatesFS.ReadFile(platformConfigFile)
	if err != nil {
		return fmt.Errorf("platform config file not found: %s", platformConfigFile)
	}

	var platformConfig []map[string]any
	if err := util.Unmarshal(platformData, &platformConfig); err != nil {
		return fmt.Errorf("failed to parse platform config: %w", err)
	}

	// 替换inbound配置
	config.Inbounds = platformConfig

	// 直接使用平台适配器进行配置适配
	return adapter.AdaptConfig(config, options)
}

// handleTailscaleConfig 处理 Tailscale 配置
func (t *EmbedTemplate) handleTailscaleConfig(config map[string]any, options config.TemplateOptions) {
	// 如果 TSAuthKey 为空，删除所有 Tailscale 相关配置
	if options.TSAuthKey == "" {
		t.removeTailscaleConfig(config)
		return
	}

	// 如果 TSAuthKey 不为空，更新或生成 endpoints 配置
	t.updateTailscaleEndpoints(config, options)
}

// updateTailscaleEndpoints 更新或生成 Tailscale endpoints 配置
func (t *EmbedTemplate) updateTailscaleEndpoints(config map[string]any, options config.TemplateOptions) {
	// 检查 endpoints 是否存在
	endpoints, ok := config["endpoints"]
	if !ok {
		// 如果不存在，创建新的 endpoints 配置
		config["endpoints"] = []map[string]any{
			{
				"type":                  "tailscale",
				"tag":                   "ts-node",
				"auth_key":              options.TSAuthKey,
				"system_interface":      true,
				"system_interface_name": "tailscale0",
				"accept_routes":         true,
				"advertise_routes":      []string{options.TSLanIPCIDR},
			},
		}
		return
	}

	// 如果存在，更新配置
	endpointsArray, ok := endpoints.([]any)
	if !ok {
		return
	}

	// 找到 ts-node endpoint 并更新
	for _, endpoint := range endpointsArray {
		if endpointMap, ok := endpoint.(map[string]any); ok {
			if tag, tagOk := endpointMap["tag"].(string); tagOk && tag == "ts-node" {
				endpointMap["auth_key"] = options.TSAuthKey
				if options.TSLanIPCIDR != "" {
					endpointMap["advertise_routes"] = []string{options.TSLanIPCIDR}
				}
				break
			}
		}
	}
}

// removeTailscaleConfig 删除所有 Tailscale 相关配置
func (t *EmbedTemplate) removeTailscaleConfig(config map[string]any) {
	// 删除 endpoints 配置
	delete(config, "endpoints")

	// 删除 route.rules 中 outbound 为 ts-node 的规则
	t.removeRouteRules(config)

	// 删除 dns.rules 中 server 名为 dns_tailscale 的规则
	t.removeDNSRules(config)

	// 删除 dns.servers 中 tag 为 dns_tailscale 的 dns 服务
	t.removeDNSServers(config)
}

// removeRouteRules 删除 route.rules 中 outbound 为 ts-node 的路由规则
func (t *EmbedTemplate) removeRouteRules(config map[string]any) {
	route, ok := config["route"]
	if !ok {
		return
	}

	routeMap, ok := route.(map[string]any)
	if !ok {
		return
	}

	rules, ok := routeMap["rules"]
	if !ok {
		return
	}

	rulesArray, ok := rules.([]any)
	if !ok {
		return
	}

	// 过滤掉 outbound 为 ts-node 的规则
	filteredRules := make([]any, 0)
	for _, rule := range rulesArray {
		if ruleMap, ok := rule.(map[string]any); ok {
			if outbound, outboundOk := ruleMap["outbound"].(string); outboundOk && outbound == "ts-node" {
				continue // 跳过这个规则
			}
			filteredRules = append(filteredRules, rule)
		}
	}

	if len(filteredRules) > 0 {
		routeMap["rules"] = filteredRules
	} else {
		delete(routeMap, "rules")
	}
}

// removeDNSRules 删除 dns.rules 中 server 名为 dns_tailscale 的规则
func (t *EmbedTemplate) removeDNSRules(config map[string]any) {
	dns, ok := config["dns"]
	if !ok {
		return
	}

	dnsMap, ok := dns.(map[string]any)
	if !ok {
		return
	}

	rules, ok := dnsMap["rules"]
	if !ok {
		return
	}

	rulesArray, ok := rules.([]any)
	if !ok {
		return
	}

	// 过滤掉 server 为 dns_tailscale 的规则
	filteredRules := make([]any, 0)
	for _, rule := range rulesArray {
		if ruleMap, ok := rule.(map[string]any); ok {
			if server, serverOk := ruleMap["server"].(string); serverOk && server == "dns_tailscale" {
				continue // 跳过这个规则
			}
			filteredRules = append(filteredRules, rule)
		}
	}

	if len(filteredRules) > 0 {
		dnsMap["rules"] = filteredRules
	} else {
		delete(dnsMap, "rules")
	}
}

// removeDNSServers 删除 dns.servers 中 tag 为 dns_tailscale 的 dns 服务
func (t *EmbedTemplate) removeDNSServers(config map[string]any) {
	dns, ok := config["dns"]
	if !ok {
		return
	}

	dnsMap, ok := dns.(map[string]any)
	if !ok {
		return
	}

	servers, ok := dnsMap["servers"]
	if !ok {
		return
	}

	serversArray, ok := servers.([]any)
	if !ok {
		return
	}

	// 过滤掉 tag 为 dns_tailscale 的服务
	filteredServers := make([]any, 0)
	for _, server := range serversArray {
		if serverMap, ok := server.(map[string]any); ok {
			if tag, tagOk := serverMap["tag"].(string); tagOk && tag == "dns_tailscale" {
				continue // 跳过这个服务
			}
			filteredServers = append(filteredServers, server)
		}
	}

	if len(filteredServers) > 0 {
		dnsMap["servers"] = filteredServers
	} else {
		delete(dnsMap, "servers")
	}
}
