package transformer

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/pkg/model"
)

// buildTLS 构建 sing-box outbound 的 tls 字段（所有协议共用）。
// defaultALPN 在节点未指定 ALPN 时使用（如 TUIC 默认 h3），可为 nil。
func buildTLS(node model.Node, defaultALPN []string) map[string]any {
	tls := map[string]any{
		"enabled":  true,
		"insecure": node.Security.SkipVerify,
	}

	// SNI：未显式指定时回退到传输层 Host（与 v2rayN 等客户端行为一致）
	serverName := node.Security.ServerName
	if serverName == "" {
		serverName = sniFromHost(node.Transport.Host)
	}
	if serverName != "" {
		tls["server_name"] = serverName
	}

	if len(node.Security.ALPN) > 0 {
		tls["alpn"] = node.Security.ALPN
	} else if len(defaultALPN) > 0 {
		tls["alpn"] = defaultALPN
	}

	// uTLS 指纹：仅适用于基于 TCP 的 TLS，QUIC 协议（hysteria2/tuic）不支持
	if !isQUICProtocol(node.Type) {
		if fp, ok := node.Extra["fingerprint"].(string); ok && fp != "" {
			tls["utls"] = map[string]any{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
	}

	return tls
}

// buildTransport 按 sing-box 各 transport 类型的合法字段构建 transport。
// 返回 nil 表示使用默认 TCP；对 sing-box 不支持的传输类型返回错误，避免生成无法加载的配置。
func buildTransport(node model.Node) (map[string]any, error) {
	netType := strings.ToLower(strings.TrimSpace(node.Transport.Net))
	switch netType {
	case "", "tcp", "raw", "none":
		return nil, nil

	case "ws", "websocket":
		transport := map[string]any{"type": "ws"}
		path, maxEarlyData := splitEarlyData(node.Transport.Path)
		if path != "" {
			transport["path"] = path
		}
		if maxEarlyData > 0 {
			transport["max_early_data"] = maxEarlyData
			transport["early_data_header_name"] = "Sec-WebSocket-Protocol"
		}
		headers := make(map[string]any, len(node.Transport.Headers)+1)
		for k, v := range node.Transport.Headers {
			headers[k] = v
		}
		// ws transport 没有 host 字段，Host 必须放在 headers 中
		if node.Transport.Host != "" {
			headers["Host"] = node.Transport.Host
		}
		if len(headers) > 0 {
			transport["headers"] = headers
		}
		return transport, nil

	case "grpc":
		transport := map[string]any{"type": "grpc"}
		serviceName, _ := node.Extra["service_name"].(string)
		if serviceName == "" {
			// VMess JSON 链接习惯把 serviceName 放在 path 中
			serviceName = strings.TrimPrefix(node.Transport.Path, "/")
		}
		if serviceName != "" {
			transport["service_name"] = serviceName
		}
		return transport, nil

	case "http", "h2":
		transport := map[string]any{"type": "http"}
		if hosts := splitHosts(node.Transport.Host); len(hosts) > 0 {
			transport["host"] = hosts
		}
		if node.Transport.Path != "" {
			transport["path"] = node.Transport.Path
		}
		if len(node.Transport.Headers) > 0 {
			transport["headers"] = node.Transport.Headers
		}
		return transport, nil

	case "httpupgrade":
		transport := map[string]any{"type": "httpupgrade"}
		if node.Transport.Host != "" {
			transport["host"] = node.Transport.Host
		}
		if node.Transport.Path != "" {
			transport["path"] = node.Transport.Path
		}
		if len(node.Transport.Headers) > 0 {
			transport["headers"] = node.Transport.Headers
		}
		return transport, nil

	case "quic":
		return map[string]any{"type": "quic"}, nil

	default:
		return nil, fmt.Errorf("transport %q is not supported by sing-box", node.Transport.Net)
	}
}

// splitEarlyData 处理 "/path?ed=2048" 形式的 WebSocket early data 参数
func splitEarlyData(path string) (string, int) {
	idx := strings.Index(path, "?")
	if idx < 0 {
		return path, 0
	}
	query, err := url.ParseQuery(path[idx+1:])
	if err != nil {
		return path, 0
	}
	ed, err := strconv.Atoi(query.Get("ed"))
	if err != nil || ed <= 0 {
		return path, 0
	}
	query.Del("ed")
	cleaned := path[:idx]
	if rest := query.Encode(); rest != "" {
		cleaned += "?" + rest
	}
	return cleaned, ed
}

func splitHosts(host string) []string {
	var hosts []string
	for _, h := range strings.Split(host, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// sniFromHost 从传输层 Host 推导 SNI：取第一个域名，IP 地址不作为 SNI
func sniFromHost(host string) string {
	hosts := splitHosts(host)
	if len(hosts) == 0 {
		return ""
	}
	if net.ParseIP(hosts[0]) != nil {
		return ""
	}
	return hosts[0]
}

func isQUICProtocol(protocol string) bool {
	return protocol == constant.ProtocolHysteria2 || protocol == constant.ProtocolTuic
}
