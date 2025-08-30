package template

import (
	"strings"

	"github.com/sixban6/singgen/internal/filter"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/internal/util"
	"go.uber.org/zap"
)

// TemplateProcessor 处理模板中的特殊占位符和过滤逻辑
type TemplateProcessor struct {
	filter filter.NodeFilter
}

// NewTemplateProcessor 创建新的模板处理器
func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		filter: filter.NewNodeFilter(),
	}
}

// ProcessAllPlaceholders 处理模板中所有的 {all} 占位符
func (p *TemplateProcessor) ProcessAllPlaceholders(config map[string]any, outbounds []transformer.Outbound) {
	// 递归处理整个配置
	p.walkAndProcessAll(config, outbounds)
}

// walkAndProcessAll 递归遍历配置并处理 {all} 占位符
func (p *TemplateProcessor) walkAndProcessAll(obj any, outbounds []transformer.Outbound) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			switch val := value.(type) {
			case []any:
				// 检查是否包含 {all} 占位符
				if p.containsAllPlaceholder(val) {
					v[key] = p.processOutboundArray(val, outbounds, v)
				} else {
					p.walkAndProcessAll(val, outbounds)
				}
			case map[string]any:
				p.walkAndProcessAll(val, outbounds)
			}
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				p.walkAndProcessAll(m, outbounds)
			} else if arr, ok := item.([]any); ok {
				p.walkAndProcessAll(arr, outbounds)
			}
		}
	}
}

// containsAllPlaceholder 检查数组是否包含 {all} 占位符
func (p *TemplateProcessor) containsAllPlaceholder(arr []any) bool {
	for _, item := range arr {
		if str, ok := item.(string); ok && str == "{all}" {
			return true
		}
	}
	return false
}

// processOutboundArray 处理包含 {all} 占位符的outbound数组
func (p *TemplateProcessor) processOutboundArray(arr []any, outbounds []transformer.Outbound, parent map[string]any) []any {
	// 解析过滤规则
	var filterRules []filter.FilterRule
	if filterData, exists := parent["filter"]; exists {
		filterRules = p.parseFilterRules(filterData)
		// 处理完成后删除filter字段，因为它不属于sing-box配置
		delete(parent, "filter")
	}
	
	// 应用过滤器获取匹配的节点tags
	filteredTags := p.filter.Filter(outbounds, filterRules)
	
	// 构建新的outbound数组
	var result []any
	for _, item := range arr {
		if str, ok := item.(string); ok && str == "{all}" {
			// 替换 {all} 为过滤后的节点tags
			for _, tag := range filteredTags {
				result = append(result, tag)
			}
		} else {
			// 保留其他元素
			result = append(result, item)
		}
	}
	
	return result
}

// parseFilterRules 解析过滤规则
func (p *TemplateProcessor) parseFilterRules(filterData any) []filter.FilterRule {
	var rules []filter.FilterRule
	
	switch data := filterData.(type) {
	case []any:
		for _, item := range data {
			if ruleMap, ok := item.(map[string]any); ok {
				rule := p.parseFilterRule(ruleMap)
				if rule != nil {
					rules = append(rules, *rule)
				}
			}
		}
	case map[string]any:
		rule := p.parseFilterRule(data)
		if rule != nil {
			rules = append(rules, *rule)
		}
	}
	
	return rules
}

// parseFilterRule 解析单个过滤规则
func (p *TemplateProcessor) parseFilterRule(ruleMap map[string]any) *filter.FilterRule {
	actionStr, ok := ruleMap["action"].(string)
	if !ok {
		if util.L != nil {
			util.L.Warn("Invalid filter rule: missing action field")
		}
		return nil
	}
	
	var action filter.FilterAction
	switch strings.ToLower(actionStr) {
	case "include":
		action = filter.ActionInclude
	case "exclude":
		action = filter.ActionExclude
	default:
		if util.L != nil {
			util.L.Warn("Unknown filter action", zap.String("action", actionStr))
		}
		return nil
	}
	
	// 解析keywords
	var keywords []string
	if keywordsData, exists := ruleMap["keywords"]; exists {
		switch kw := keywordsData.(type) {
		case []any:
			for _, item := range kw {
				if str, ok := item.(string); ok {
					keywords = append(keywords, str)
				}
			}
		case []string:
			keywords = kw
		case string:
			keywords = []string{kw}
		}
	}
	
	return &filter.FilterRule{
		Action:   action,
		Keywords: keywords,
	}
}

// logFilterResult 记录过滤结果（用于调试）
func (p *TemplateProcessor) logFilterResult(rules []filter.FilterRule, originalCount int, filteredCount int) {
	if util.L != nil && len(rules) > 0 {
		rulesJson, _ := util.Marshal(rules)
		util.L.Debug("Node filter applied",
			zap.String("rules", string(rulesJson)),
			zap.Int("original_count", originalCount),
			zap.Int("filtered_count", filteredCount),
		)
	}
}