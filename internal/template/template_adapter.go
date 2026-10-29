package template

import (
	"strconv"
	"strings"
)

// adaptTemplateForVersion adapts a baseline v1.14+ template in-memory for older sing-box versions (e.g. v1.12, v1.13)
// or modern versions, ensuring single baseline template maintenance without duplication.
func adaptTemplateForVersion(config map[string]any, version string) map[string]any {
	minor := parseVersionMinor(version)
	if minor < 14 {
		// Adapting v1.14+ baseline to v1.12/v1.13:
		// 1. Remove top-level http_clients
		delete(config, "http_clients")

		// 2. Remove top-level services
		delete(config, "services")

		// 3. Adapt route: remove default_http_client, add download_detour to remote rule_sets
		if route, ok := config["route"].(map[string]any); ok {
			delete(route, "default_http_client")

			if ruleSets, ok := route["rule_set"].([]any); ok {
				for _, rs := range ruleSets {
					if rsMap, ok := rs.(map[string]any); ok {
						if t, ok := rsMap["type"].(string); ok && t == "remote" {
							if _, hasDetour := rsMap["download_detour"]; !hasDetour {
								rsMap["download_detour"] = "DirectConn"
							}
						}
					}
				}
			} else if ruleSetsMap, ok := route["rule_set"].([]map[string]any); ok {
				for _, rsMap := range ruleSetsMap {
					if t, ok := rsMap["type"].(string); ok && t == "remote" {
						if _, hasDetour := rsMap["download_detour"]; !hasDetour {
							rsMap["download_detour"] = "DirectConn"
						}
					}
				}
			}
		}

		// 4. Adapt dns: optimistic -> independent_cache
		if dns, ok := config["dns"].(map[string]any); ok {
			delete(dns, "optimistic")
			dns["independent_cache"] = true
		}

		// 5. Adapt experimental.cache_file: remove store_dns and store_fakeip
		if exp, ok := config["experimental"].(map[string]any); ok {
			if cacheFile, ok := exp["cache_file"].(map[string]any); ok {
				delete(cacheFile, "store_dns")
				delete(cacheFile, "store_fakeip")
			}
		}

		// 6. Adapt tailscale endpoint: state_directory -> directory
		if endpoints, ok := config["endpoints"].([]any); ok {
			for _, ep := range endpoints {
				if epMap, ok := ep.(map[string]any); ok {
					if t, _ := epMap["type"].(string); t == "tailscale" {
						if dir, exists := epMap["state_directory"]; exists {
							epMap["directory"] = dir
							delete(epMap, "state_directory")
						}
					}
				}
			}
		} else if endpointsMap, ok := config["endpoints"].([]map[string]any); ok {
			for _, epMap := range endpointsMap {
				if t, _ := epMap["type"].(string); t == "tailscale" {
					if dir, exists := epMap["state_directory"]; exists {
						epMap["directory"] = dir
						delete(epMap, "state_directory")
					}
				}
			}
		}
	} else {
		// v1.14+: ensure modern fields
		if route, ok := config["route"].(map[string]any); ok {
			if _, ok := route["default_http_client"]; !ok {
				route["default_http_client"] = "hc-direct"
			}
			if ruleSets, ok := route["rule_set"].([]any); ok {
				for _, rs := range ruleSets {
					if rsMap, ok := rs.(map[string]any); ok {
						delete(rsMap, "download_detour")
					}
				}
			} else if ruleSetsMap, ok := route["rule_set"].([]map[string]any); ok {
				for _, rsMap := range ruleSetsMap {
					delete(rsMap, "download_detour")
				}
			}
		}
		if dns, ok := config["dns"].(map[string]any); ok {
			delete(dns, "independent_cache")
			dns["optimistic"] = true
		}
		if endpoints, ok := config["endpoints"].([]any); ok {
			for _, ep := range endpoints {
				if epMap, ok := ep.(map[string]any); ok {
					if t, _ := epMap["type"].(string); t == "tailscale" {
						if dir, exists := epMap["directory"]; exists {
							epMap["state_directory"] = dir
							delete(epMap, "directory")
						}
					}
				}
			}
		} else if endpointsMap, ok := config["endpoints"].([]map[string]any); ok {
			for _, epMap := range endpointsMap {
				if t, _ := epMap["type"].(string); t == "tailscale" {
					if dir, exists := epMap["directory"]; exists {
						epMap["state_directory"] = dir
						delete(epMap, "directory")
					}
				}
			}
		}
	}
	return config
}

func parseVersionMinor(v string) int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		if minor, err := strconv.Atoi(parts[1]); err == nil {
			return minor
		}
	}
	return 14 // default to 14
}
