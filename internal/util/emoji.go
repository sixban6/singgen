package util

import (
	"regexp"
	"strings"
	"unicode"
)

// RemoveEmoji 从字符串中移除所有emoji字符
func RemoveEmoji(text string) string {
	// 使用正则表达式匹配emoji范围
	emojiPattern := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]|[\x{1F900}-\x{1F9FF}]|[\x{1F018}-\x{1F270}]|[\x{238C}]|[\x{2194}-\x{2199}]|[\x{21A9}-\x{21AA}]|[\x{2B05}-\x{2B07}]|[\x{2B1B}-\x{2B1C}]|[\x{2B50}]|[\x{2B55}]|[\x{3030}]|[\x{303D}]|[\x{3297}]|[\x{3299}]`)
	
	result := emojiPattern.ReplaceAllString(text, "")
	
	// 额外处理一些特殊符号
	result = removeSpecialSymbols(result)
	
	// 清理多余的空格
	result = strings.TrimSpace(result)
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	
	return result
}

// removeSpecialSymbols 移除特殊符号和emoji相关字符
func removeSpecialSymbols(text string) string {
	var result strings.Builder
	
	for _, r := range text {
		// 跳过emoji相关的unicode符号
		if isEmojiRelated(r) {
			continue
		}
		result.WriteRune(r)
	}
	
	return result.String()
}

// isEmojiRelated 判断字符是否为emoji相关字符
func isEmojiRelated(r rune) bool {
	// 检查常见的emoji unicode范围
	switch {
	case r >= 0x1F600 && r <= 0x1F64F: // 表情符号
		return true
	case r >= 0x1F300 && r <= 0x1F5FF: // 杂项符号和象形文字
		return true
	case r >= 0x1F680 && r <= 0x1F6FF: // 交通和地图符号
		return true
	case r >= 0x1F1E0 && r <= 0x1F1FF: // 旗帜 (国家/地区)
		return true
	case r >= 0x2600 && r <= 0x26FF: // 杂项符号
		return true
	case r >= 0x2700 && r <= 0x27BF: // 装饰符号
		return true
	case r >= 0x1F900 && r <= 0x1F9FF: // 补充符号和象形文字
		return true
	case r >= 0x1F018 && r <= 0x1F270: // 其他符号
		return true
	case r == 0x238C: // 三角形标点
		return true
	case r >= 0x2194 && r <= 0x2199: // 箭头
		return true
	case r >= 0x21A9 && r <= 0x21AA: // 钩箭头
		return true
	case r >= 0x2B05 && r <= 0x2B07: // 箭头
		return true
	case r >= 0x2B1B && r <= 0x2B1C: // 方块
		return true
	case r == 0x2B50: // 星星
		return true
	case r == 0x2B55: // 圆圈
		return true
	case r == 0x3030: // 波浪号
		return true
	case r == 0x303D: // 部分标记
		return true
	case r == 0x3297 || r == 0x3299: // 表意文字
		return true
	// 检查其他可能的emoji字符
	case unicode.Is(unicode.Symbol, r):
		// 进一步检查是否为表情符号
		return isLikelyEmoji(r)
	default:
		return false
	}
}

// isLikelyEmoji 进一步判断符号是否可能是emoji
func isLikelyEmoji(r rune) bool {
	// 一些启发式规则来判断可能的emoji
	switch r {
	case '🏳', '🏴': // 旗帜基础
		return true
	case '♂', '♀': // 性别符号
		return true
	case '⚡', '⭐', '❤', '💙', '💚', '💛', '🧡', '💜', '🖤', '🤍', '🤎': // 常见符号
		return true
	default:
		// 检查是否在其他emoji范围内
		return r >= 0x1F000 && r <= 0x1FAFF
	}
}

// CleanNodeTag 清理节点标签，可选择是否移除emoji
func CleanNodeTag(tag string, removeEmoji bool) string {
	if removeEmoji {
		return RemoveEmoji(tag)
	}
	return strings.TrimSpace(tag)
}

// ValidateDNSServer 验证DNS服务器地址格式
func ValidateDNSServer(server string) bool {
	if server == "" {
		return false
	}
	
	// 简单的IP地址格式验证
	parts := strings.Split(server, ".")
	if len(parts) != 4 {
		return false
	}
	
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		
		for _, r := range part {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		
		// 转换为数字验证范围
		var num int
		for _, r := range part {
			num = num*10 + int(r-'0')
		}
		
		if num < 0 || num > 255 {
			return false
		}
	}
	
	return true
}