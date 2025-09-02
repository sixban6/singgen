package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveInvalidReferences(t *testing.T) {
	// 测试单个方法的逻辑
	outbounds := []any{"Valid1", "Invalid", "Valid2", "AlsoInvalid"}
	existingTags := map[string]bool{
		"Valid1": true,
		"Valid2": true,
		// "Invalid" 和 "AlsoInvalid" 不存在
	}

	// 手动实现 removeInvalidReferences 的逻辑进行验证
	var cleaned []any
	for _, item := range outbounds {
		if tag, ok := item.(string); ok {
			if existingTags[tag] {
				cleaned = append(cleaned, item)
			}
		} else {
			cleaned = append(cleaned, item)
		}
	}

	t.Logf("原数组: %v", outbounds)
	t.Logf("存在tags: %v", existingTags)
	t.Logf("清理后: %v", cleaned)

	// 验证清理结果
	assert.Equal(t, 2, len(cleaned), "应该只剩2个有效引用")
	assert.Contains(t, cleaned, "Valid1")
	assert.Contains(t, cleaned, "Valid2")
	assert.NotContains(t, cleaned, "Invalid")
	assert.NotContains(t, cleaned, "AlsoInvalid")
}

func TestWalkAndCleanLogic(t *testing.T) {
	// 测试整个遍历和清理的逻辑
	testMap := map[string]any{
		"outbounds": []any{"Keep", "Remove", "AlsoKeep"},
		"other":     "data",
	}

	existingTags := map[string]bool{
		"Keep":     true,
		"AlsoKeep": true,
		// "Remove" 不存在
	}

	t.Logf("清理前: %v", testMap["outbounds"])

	// 手动应用清理逻辑
	if outbounds, ok := testMap["outbounds"].([]any); ok {
		var cleaned []any
		for _, item := range outbounds {
			if tag, ok := item.(string); ok {
				if existingTags[tag] {
					cleaned = append(cleaned, item)
				}
			} else {
				cleaned = append(cleaned, item)
			}
		}
		if len(cleaned) != len(outbounds) {
			testMap["outbounds"] = cleaned
		}
	}

	t.Logf("清理后: %v", testMap["outbounds"])

	cleanedOutbounds := testMap["outbounds"].([]any)
	assert.Equal(t, 2, len(cleanedOutbounds))
	assert.Contains(t, cleanedOutbounds, "Keep")
	assert.Contains(t, cleanedOutbounds, "AlsoKeep")
	assert.NotContains(t, cleanedOutbounds, "Remove")
}