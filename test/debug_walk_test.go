package test

import (
	"fmt"
	"testing"
)

func TestDebugWalk(t *testing.T) {
	config := map[string]any{
		"outbounds": []any{
			map[string]any{
				"tag":       "DNS",
				"type":      "selector",
				"outbounds": []any{"AutoSelect-HK", "ChainProxy", "block"},
			},
			map[string]any{
				"tag":  "AutoSelect-HK",
				"type": "urltest",
			},
		},
	}

	existingTags := map[string]bool{
		"DNS":           true,
		"AutoSelect-HK": true, 
		"block":         true,
		// ChainProxy 不存在
	}

	t.Logf("配置结构:")
	printConfig(config, 0)

	t.Logf("\n存在的tags: %v", existingTags)

	// 手动实现遍历逻辑来调试
	t.Logf("\n开始手动遍历清理:")
	manualWalk(config, existingTags, 0)

	t.Logf("\n清理后配置:")
	printConfig(config, 0)
}

func printConfig(obj any, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			fmt.Printf("%s%s:\n", prefix, key)
			printConfig(value, indent+1)
		}
	case []any:
		for i, item := range v {
			fmt.Printf("%s[%d]:\n", prefix, i)
			printConfig(item, indent+1)
		}
	default:
		fmt.Printf("%s%v\n", prefix, v)
	}
}

func manualWalk(obj any, existingTags map[string]bool, depth int) {
	prefix := ""
	for i := 0; i < depth; i++ {
		prefix += "  "
	}

	switch v := obj.(type) {
	case map[string]any:
		fmt.Printf("%s进入map，键: %v\n", prefix, getKeys(v))
		for key, value := range v {
			fmt.Printf("%s处理键: %s\n", prefix, key)
			switch val := value.(type) {
			case []any:
				if key == "outbounds" {
					fmt.Printf("%s找到outbounds数组，长度: %d\n", prefix, len(val))
					// 检查这是否是引用数组（包含字符串）还是outbound对象数组
					if len(val) > 0 {
						if _, isString := val[0].(string); isString {
							// 这是引用数组，需要清理
							fmt.Printf("%s  这是引用数组: %v\n", prefix, val)
							var cleaned []any
							for _, item := range val {
								if tag, ok := item.(string); ok {
									fmt.Printf("%s    检查引用: %s, 存在: %v\n", prefix, tag, existingTags[tag])
									if existingTags[tag] {
										cleaned = append(cleaned, item)
									}
								} else {
									cleaned = append(cleaned, item)
								}
							}
							if len(cleaned) != len(val) {
								fmt.Printf("%s    清理: %v -> %v\n", prefix, val, cleaned)
								v[key] = cleaned
							} else {
								fmt.Printf("%s    无需清理\n", prefix)
							}
						} else {
							// 这是outbound对象数组，需要递归处理每个对象
							fmt.Printf("%s  这是outbound对象数组，递归处理\n", prefix)
							manualWalk(val, existingTags, depth+1)
						}
					}
				} else {
					manualWalk(val, existingTags, depth+1)
				}
			case map[string]any:
				manualWalk(val, existingTags, depth+1)
			}
		}
	case []any:
		fmt.Printf("%s进入数组，长度: %d\n", prefix, len(v))
		for i, item := range v {
			fmt.Printf("%s  [%d]:\n", prefix, i)
			manualWalk(item, existingTags, depth+1)
		}
	default:
		fmt.Printf("%s叶子节点: %v\n", prefix, v)
	}
}

func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}