package template

import (
	"strings"
	"sync"

	"github.com/sixban6/singgen/internal/filter"
	"github.com/sixban6/singgen/internal/transformer"
	"github.com/sixban6/singgen/internal/util"
	"go.uber.org/zap"
)

var (
	indexSlicePool = sync.Pool{
		New: func() interface{} {
			return make([]int, 0, 10)
		},
	}
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
	// 预构建节点索引
	existingTags := p.buildTagIndex(outbounds)
	
	// 处理占位符
	p.walkAndProcessAll(config, outbounds)
	
	// 高效级联删除空的outbound块
	p.removeEmptyOutboundsWithCascade(config, existingTags)
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
	// 支持两种格式：
	// 1. {"action": "exclude", "keywords": [".*"]}  
	// 2. {"exclude": [".*"]} 或 {"include": ["pattern"]}
	
	var action filter.FilterAction
	var keywords []string
	
	// 尝试新格式 {"exclude": [...]} 或 {"include": [...]}
	if excludeData, exists := ruleMap["exclude"]; exists {
		action = filter.ActionExclude
		keywords = p.parseKeywords(excludeData)
	} else if includeData, exists := ruleMap["include"]; exists {
		action = filter.ActionInclude
		keywords = p.parseKeywords(includeData)
	} else {
		// 尝试旧格式 {"action": "...", "keywords": [...]}
		actionStr, ok := ruleMap["action"].(string)
		if !ok {
			if util.L != nil {
				util.L.Warn("Invalid filter rule: missing action or include/exclude field")
			}
			return nil
		}
		
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
		
		if keywordsData, exists := ruleMap["keywords"]; exists {
			keywords = p.parseKeywords(keywordsData)
		}
	}
	
	if len(keywords) == 0 {
		if util.L != nil {
			util.L.Warn("Filter rule has no keywords")
		}
		return nil
	}
	
	return &filter.FilterRule{
		Action:   action,
		Keywords: keywords,
	}
}

// parseKeywords 解析关键词列表
func (p *TemplateProcessor) parseKeywords(keywordsData any) []string {
	var keywords []string
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
	return keywords
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

// buildTagIndex 预构建节点索引以提高查找效率
func (p *TemplateProcessor) buildTagIndex(outbounds []transformer.Outbound) map[string]bool {
	tags := make(map[string]bool, len(outbounds))
	for _, ob := range outbounds {
		tags[ob.Tag] = true
	}
	return tags
}

// removeEmptyOutboundsWithCascade 级联删除空的outbound块
func (p *TemplateProcessor) removeEmptyOutboundsWithCascade(config map[string]any, existingTags map[string]bool) {
	maxIterations := 10
	
	for iteration := 0; iteration < maxIterations; iteration++ {
		removedCount := 0
		// 更新existingTags包含当前还存在的outbound tags
		currentTags := p.buildCurrentOutboundTags(config, existingTags)
		
		// 先清理无效引用
		p.cleanInvalidReferences(config, currentTags)
		
		// 再删除空的outbound
		p.walkAndRemoveEmpty(config, currentTags, &removedCount)
		if removedCount == 0 {
			break
		}
		
		if util.L != nil {
			util.L.Debug("Removed empty outbounds in iteration",
				zap.Int("iteration", iteration+1),
				zap.Int("removed_count", removedCount),
			)
		}
	}
	
	// 最终清理：确保所有无效引用都被清理
	finalTags := p.buildCurrentOutboundTags(config, existingTags)
	p.cleanInvalidReferences(config, finalTags)
}

// walkAndRemoveEmpty 遍历配置并删除空的outbound块
func (p *TemplateProcessor) walkAndRemoveEmpty(obj any, existingTags map[string]bool, removedCount *int) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			switch val := value.(type) {
			case []any:
				if key == "outbounds" {
					newArray := p.removeEmptyOutboundsFromArray(val, existingTags, removedCount)
					if len(newArray) != len(val) {
						v[key] = newArray
					}
				} else {
					p.walkAndRemoveEmpty(val, existingTags, removedCount)
				}
			case map[string]any:
				p.walkAndRemoveEmpty(val, existingTags, removedCount)
			}
		}
	case []any:
		for _, item := range v {
			p.walkAndRemoveEmpty(item, existingTags, removedCount)
		}
	}
}

// removeEmptyOutboundsFromArray 从outbounds数组中删除空的outbound块
func (p *TemplateProcessor) removeEmptyOutboundsFromArray(outbounds []any, existingTags map[string]bool, removedCount *int) []any {
	toRemoveIndexes := p.getIndexSlice()
	defer p.putIndexSlice(toRemoveIndexes)
	
	// 逆序收集待删除索引
	for i := len(outbounds) - 1; i >= 0; i-- {
		if outbound, ok := outbounds[i].(map[string]any); ok {
			if p.shouldRemoveOutbound(outbound, existingTags) {
				toRemoveIndexes = append(toRemoveIndexes, i)
			}
		}
	}
	
	// 批量删除
	if len(toRemoveIndexes) > 0 {
		*removedCount += len(toRemoveIndexes)
		return p.batchRemove(outbounds, toRemoveIndexes)
	}
	
	return outbounds
}

// shouldRemoveOutbound 判断outbound是否应该被删除
func (p *TemplateProcessor) shouldRemoveOutbound(outbound map[string]any, existingTags map[string]bool) bool {
	outboundsData, exists := outbound["outbounds"]
	if !exists {
		return false
	}
	
	outboundsArray, ok := outboundsData.([]any)
	if !ok {
		return false
	}
	
	if len(outboundsArray) == 0 {
		return true
	}
	
	// 检查是否只包含默认的block节点（过滤结果为空时的默认值）
	if len(outboundsArray) == 1 {
		if tag, ok := outboundsArray[0].(string); ok && tag == "block" {
			return true
		}
	}
	
	// 早期返回优化：找到一个有效节点就返回false
	for _, item := range outboundsArray {
		if tag, ok := item.(string); ok && existingTags[tag] {
			return false
		}
	}
	
	return true
}

// batchRemove 批量删除指定索引的元素
func (p *TemplateProcessor) batchRemove(slice []any, indexes []int) []any {
	if len(indexes) == 0 {
		return slice
	}
	
	// indexes已经是逆序的，直接删除不会影响后续索引
	for _, idx := range indexes {
		slice = append(slice[:idx], slice[idx+1:]...)
	}
	
	return slice
}

// getIndexSlice 从对象池获取索引slice
func (p *TemplateProcessor) getIndexSlice() []int {
	return indexSlicePool.Get().([]int)[:0]
}

// putIndexSlice 归还索引slice到对象池
func (p *TemplateProcessor) putIndexSlice(s []int) {
	indexSlicePool.Put(s)
}

// buildCurrentOutboundTags 构建当前配置中所有outbound的tag索引
func (p *TemplateProcessor) buildCurrentOutboundTags(config map[string]any, originalTags map[string]bool) map[string]bool {
	currentTags := make(map[string]bool)
	
	// 先添加原有的节点tags
	for tag := range originalTags {
		currentTags[tag] = true
	}
	
	// 再添加当前配置中的outbound tags
	p.collectOutboundTags(config, currentTags)
	
	return currentTags
}

// cleanInvalidReferences 清理无效的outbound引用
func (p *TemplateProcessor) cleanInvalidReferences(config map[string]any, existingTags map[string]bool) {
	if util.L != nil {
		tagCount := len(existingTags)
		tagList := make([]string, 0, tagCount)
		for tag := range existingTags {
			tagList = append(tagList, tag)
		}
		util.L.Debug("Starting reference cleanup",
			zap.Int("existing_tags_count", tagCount),
			zap.Strings("existing_tags", tagList),
		)
	}
	p.walkAndCleanReferences(config, existingTags)
}

// walkAndCleanReferences 遍历配置并清理无效引用
func (p *TemplateProcessor) walkAndCleanReferences(obj any, existingTags map[string]bool) {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			switch val := value.(type) {
			case []any:
				if key == "outbounds" {
					// 需要区分两种outbounds数组：
					// 1. 引用数组（包含字符串） - 需要清理
					// 2. outbound对象数组 - 需要递归处理
					if len(val) > 0 {
						if _, isString := val[0].(string); isString {
							// 这是引用数组，清理无效引用
							cleanedArray := p.removeInvalidReferences(val, existingTags)
							if len(cleanedArray) != len(val) {
								v[key] = cleanedArray
								if util.L != nil {
									util.L.Debug("Cleaned invalid references in outbound",
										zap.String("outbound", p.getOutboundTag(v)),
										zap.Int("before", len(val)),
										zap.Int("after", len(cleanedArray)),
									)
								}
							}
						} else {
							// 这是outbound对象数组，递归处理每个对象
							p.walkAndCleanReferences(val, existingTags)
						}
					}
				} else {
					p.walkAndCleanReferences(val, existingTags)
				}
			case map[string]any:
				p.walkAndCleanReferences(val, existingTags)
			}
		}
	case []any:
		for _, item := range v {
			p.walkAndCleanReferences(item, existingTags)
		}
	}
}

// removeInvalidReferences 从outbounds数组中移除无效引用
func (p *TemplateProcessor) removeInvalidReferences(outbounds []any, existingTags map[string]bool) []any {
	var cleanedArray []any
	for _, item := range outbounds {
		if tag, ok := item.(string); ok {
			// 只保留在existingTags中存在的引用
			if existingTags[tag] {
				cleanedArray = append(cleanedArray, item)
			} else {
				if util.L != nil {
					util.L.Debug("Removing invalid reference", zap.String("reference", tag))
				}
			}
		} else {
			// 非字符串项保留
			cleanedArray = append(cleanedArray, item)
		}
	}
	return cleanedArray
}

// getOutboundTag 获取outbound的tag用于日志
func (p *TemplateProcessor) getOutboundTag(outbound map[string]any) string {
	if tag, exists := outbound["tag"]; exists {
		if tagStr, ok := tag.(string); ok {
			return tagStr
		}
	}
	return "unknown"
}

// CleanInvalidReferencesPublic 公开的清理方法用于测试
func (p *TemplateProcessor) CleanInvalidReferencesPublic(config map[string]any, existingTags map[string]bool) {
	p.cleanInvalidReferences(config, existingTags)
}

// collectOutboundTags 收集配置中所有outbound的tag  
func (p *TemplateProcessor) collectOutboundTags(obj any, tags map[string]bool) {
	// 只在outbounds数组级别收集，避免收集引用
	if config, ok := obj.(map[string]any); ok {
		if outboundsData, exists := config["outbounds"]; exists {
			if outboundsArray, ok := outboundsData.([]any); ok {
				for _, item := range outboundsArray {
					if outbound, ok := item.(map[string]any); ok {
						if tag, exists := outbound["tag"]; exists {
							if tagStr, ok := tag.(string); ok && outbound["type"] != nil {
								tags[tagStr] = true
							}
						}
					}
				}
			}
		}
		// 递归处理其他字段（如route等）
		for key, value := range config {
			if key != "outbounds" { // 避免重复处理outbounds
				p.collectOutboundTags(value, tags)
			}
		}
	} else if arr, ok := obj.([]any); ok {
		for _, item := range arr {
			p.collectOutboundTags(item, tags)
		}
	}
}