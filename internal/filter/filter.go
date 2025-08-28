package filter

import (
	"regexp"
	"strings"

	"github.com/sixban6/singgen/internal/transformer"
)

// FilterAction 定义过滤动作类型
type FilterAction string

const (
	ActionInclude FilterAction = "include"
	ActionExclude FilterAction = "exclude"
)

// FilterRule 定义单个过滤规则
type FilterRule struct {
	Action   FilterAction `json:"action"`
	Keywords []string     `json:"keywords"`
}

// NodeFilter 定义节点过滤器接口
type NodeFilter interface {
	Filter(outbounds []transformer.Outbound, rules []FilterRule) []string
}

// DefaultNodeFilter 实现默认的节点过滤逻辑
type DefaultNodeFilter struct{}

// NewNodeFilter 创建新的节点过滤器
func NewNodeFilter() NodeFilter {
	return &DefaultNodeFilter{}
}

// Filter 根据规则过滤节点并返回匹配的节点tag列表
func (f *DefaultNodeFilter) Filter(outbounds []transformer.Outbound, rules []FilterRule) []string {
	if len(rules) == 0 {
		// 如果没有过滤规则，返回所有节点
		return f.getAllTags(outbounds)
	}

	var result []transformer.Outbound

	// 按顺序应用过滤规则
	for _, rule := range rules {
		switch rule.Action {
		case ActionInclude:
			result = f.includeFilter(outbounds, rule.Keywords)
		case ActionExclude:
			if len(result) == 0 {
				result = outbounds // 如果还没有结果，先从所有节点开始
			}
			result = f.excludeFilter(result, rule.Keywords)
		}
	}

	if len(result) == 0 {
		result = []transformer.Outbound{transformer.NewDefaultBlockOutound()}
	}

	return f.outboundsToTags(result)
}

// includeFilter 实现包含过滤逻辑
func (f *DefaultNodeFilter) includeFilter(outbounds []transformer.Outbound, keywords []string) []transformer.Outbound {
	var result []transformer.Outbound

	for _, outbound := range outbounds {
		if f.matchKeywords(outbound.Tag, keywords) {
			result = append(result, outbound)
		}
	}

	return result
}

// excludeFilter 实现排除过滤逻辑
func (f *DefaultNodeFilter) excludeFilter(outbounds []transformer.Outbound, keywords []string) []transformer.Outbound {
	var result []transformer.Outbound

	for _, outbound := range outbounds {
		if !f.matchKeywords(outbound.Tag, keywords) {
			result = append(result, outbound)
		}
	}

	return result
}

// matchKeywords 检查标签是否匹配关键词列表
func (f *DefaultNodeFilter) matchKeywords(tag string, keywords []string) bool {
	for _, keyword := range keywords {
		if f.matchPattern(tag, keyword) {
			return true
		}
	}
	return false
}

// matchPattern 检查标签是否匹配单个关键词模式（支持正则表达式）
func (f *DefaultNodeFilter) matchPattern(tag, pattern string) bool {
	// 将|分隔的模式转换为正则表达式
	patterns := strings.Split(pattern, "|")

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// 尝试作为正则表达式匹配
		if matched, err := regexp.MatchString(p, tag); err == nil && matched {
			return true
		}

		// 如果正则表达式失败，尝试简单的字符串包含匹配
		if strings.Contains(tag, p) {
			return true
		}
	}

	return false
}

// getAllTags 获取所有outbound的tag
func (f *DefaultNodeFilter) getAllTags(outbounds []transformer.Outbound) []string {
	return f.outboundsToTags(outbounds)
}

// outboundsToTags 将outbound数组转换为tag数组
func (f *DefaultNodeFilter) outboundsToTags(outbounds []transformer.Outbound) []string {
	tags := make([]string, len(outbounds))
	for i, outbound := range outbounds {
		tags[i] = outbound.Tag
	}
	return tags
}
