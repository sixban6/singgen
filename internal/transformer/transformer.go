package transformer

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/logx"
	"github.com/sixban6/singgen/pkg/model"
)

type Outbound struct {
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	Server     string `json:"server"`
	ServerPort uint16 `json:"server_port"`
	UUID       string `json:"uuid,omitempty"`
	Password   string `json:"password,omitempty"`
	Method     string `json:"method,omitempty"`
	// VMess: 加密方式 (auto/aes-128-gcm/chacha20-poly1305/none/zero) 与 alter_id
	Security                 string         `json:"security,omitempty"`
	AlterID                  int            `json:"alter_id,omitempty"`
	Flow                     string         `json:"flow,omitempty"`
	UpMbps                   int            `json:"up_mbps,omitempty"`
	DownMbps                 int            `json:"down_mbps,omitempty"`
	IdleSessionCheckInterval string         `json:"idle_session_check_interval,omitempty"`
	IdleSessionTimeout       string         `json:"idle_session_timeout,omitempty"`
	MinIdleSession           int            `json:"min_idle_session,omitempty"`
	Transport                map[string]any `json:"transport,omitempty"`
	TLS                      map[string]any `json:"tls,omitempty"`
	Multiplex                map[string]any `json:"multiplex,omitempty"`
	// Hysteria2: obfs 混淆（salamander/gecko）
	Obfs map[string]any `json:"obfs,omitempty"`
	// Hysteria2: 端口跳跃（1.14 新增 hop_interval_max 随机化）
	ServerPorts    []string `json:"server_ports,omitempty"`
	HopInterval    string   `json:"hop_interval,omitempty"`
	HopIntervalMax string   `json:"hop_interval_max,omitempty"`
	// TUIC fields
	CongestionControl string `json:"congestion_control,omitempty"`
	UDPRelayMode      string `json:"udp_relay_mode,omitempty"`
	UDPOverStream     bool   `json:"udp_over_stream,omitempty"`
	ZeroRttHandshake  bool   `json:"zero_rtt_handshake,omitempty"`
	Heartbeat         string `json:"heartbeat,omitempty"`
	Network           string `json:"network,omitempty"`
}

func NewDefaultBlockOutound() Outbound {
	return Outbound{
		Type:       constant.ProtocolSocks,
		Tag:        "block",
		Server:     "0.0.0.0",
		ServerPort: 1080,
	}
}

type Transformer interface {
	Transform(nodes []model.Node) ([]Outbound, error)
}

type SingBoxTransformer struct {
	// Logger 可选；为 nil 时不输出日志
	Logger *slog.Logger
}

func (t *SingBoxTransformer) logger() *slog.Logger { return logx.Or(t.Logger) }

func NewSingBoxTransformer() *SingBoxTransformer {
	return &SingBoxTransformer{}
}

// Transform 将节点逐个转换为 outbound，转换失败的节点记录警告后跳过，保持原有顺序。
// 每个节点只是少量字段拷贝，顺序处理即可，无需并发。
func (t *SingBoxTransformer) Transform(nodes []model.Node) ([]Outbound, error) {
	outbounds := make([]Outbound, 0, len(nodes))
	for _, node := range nodes {
		outbound, err := t.transformNode(node)
		if err != nil {
			t.logger().Warn("Failed to transform node", "tag", node.Tag, "error", err)
			continue
		}
		outbounds = append(outbounds, *outbound)
	}
	return outbounds, nil
}

func (t *SingBoxTransformer) transformNode(node model.Node) (*Outbound, error) {
	outbound := &Outbound{
		Tag:        node.Tag,
		Server:     node.Addr,
		ServerPort: node.Port,
	}

	switch node.Type {
	case constant.ProtocolVMess:
		return t.transformVmess(node, outbound)
	case constant.ProtocolVLESS:
		return t.transformVless(node, outbound)
	case constant.ProtocolTrojan:
		return t.transformTrojan(node, outbound)
	case constant.ProtocolHysteria2:
		return t.transformHysteria2(node, outbound)
	case constant.ProtocolAnyTLS:
		return t.transformAnyTLS(node, outbound)
	case constant.ProtocolTuic:
		return t.transformTuic(node, outbound)
	case constant.ProtocolSS:
		return t.transformShadowsocks(node, outbound)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", node.Type)
	}
}

func (t *SingBoxTransformer) transformVmess(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "vmess"
	outbound.UUID = node.UUID

	// 加密方式：缺省为 auto
	outbound.Security = "auto"
	if scy, ok := node.Extra["security"].(string); ok && scy != "" {
		outbound.Security = scy
	}
	// alter_id > 0 表示旧版 VMess MD5 认证；与 multiplex 无关
	if alterID, ok := node.Extra["alter_id"]; ok {
		outbound.AlterID = toInt(alterID)
	}

	if node.Security.TLS {
		outbound.TLS = buildTLS(node, nil)
	}
	transport, err := buildTransport(node)
	if err != nil {
		return nil, err
	}
	outbound.Transport = transport

	return outbound, nil
}

func (t *SingBoxTransformer) transformVless(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "vless"
	outbound.UUID = node.UUID

	// Handle flow control for XTLS
	if flow, ok := node.Extra["flow"].(string); ok && flow != "" {
		outbound.Flow = flow
	}

	if node.Security.TLS {
		tls := buildTLS(node, nil)

		// Handle Reality protocol
		if security, ok := node.Extra["security"].(string); ok && security == "reality" {
			reality := map[string]any{
				"enabled": true,
			}

			if publicKey, ok := node.Extra["public_key"].(string); ok && publicKey != "" {
				reality["public_key"] = publicKey
			}
			if shortID, ok := node.Extra["short_id"].(string); ok && shortID != "" {
				reality["short_id"] = shortID
			}

			tls["reality"] = reality
		}

		outbound.TLS = tls
	}

	transport, err := buildTransport(node)
	if err != nil {
		return nil, err
	}
	outbound.Transport = transport

	return outbound, nil
}

func (t *SingBoxTransformer) transformTrojan(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "trojan"
	outbound.Password = node.Password

	outbound.TLS = buildTLS(node, nil)
	transport, err := buildTransport(node)
	if err != nil {
		return nil, err
	}
	outbound.Transport = transport

	return outbound, nil
}

func (t *SingBoxTransformer) transformHysteria2(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "hysteria2"
	outbound.Password = node.Password

	outbound.TLS = buildTLS(node, nil)

	// 设置带宽限制参数（防止运营商检测异常流量）
	if upMbps, ok := node.Extra["up_mbps"]; ok {
		outbound.UpMbps = toInt(upMbps)
	}
	if downMbps, ok := node.Extra["down_mbps"]; ok {
		outbound.DownMbps = toInt(downMbps)
	}

	// obfs 混淆（salamander / gecko）：服务端启用混淆时缺失会导致无法连接
	if obfsData, ok := node.Extra["obfs"]; ok {
		if obfsMap, ok := obfsData.(map[string]string); ok && obfsMap["type"] != "" {
			obfs := map[string]any{"type": obfsMap["type"]}
			if obfsMap["password"] != "" {
				obfs["password"] = obfsMap["password"]
			}
			outbound.Obfs = obfs
		}
	}

	// 端口跳跃：解析器已规范化为 sing-box 的 []string 冒号格式，1.14 新增 hop_interval_max 随机化跳跃间隔
	if serverPorts, ok := node.Extra["server_ports"].([]string); ok && len(serverPorts) > 0 {
		outbound.ServerPorts = serverPorts
	}
	if hopInterval, ok := node.Extra["hop_interval"].(string); ok && hopInterval != "" {
		outbound.HopInterval = hopInterval
	}
	if hopIntervalMax, ok := node.Extra["hop_interval_max"].(string); ok && hopIntervalMax != "" {
		outbound.HopIntervalMax = hopIntervalMax
	}

	return outbound, nil
}

func (t *SingBoxTransformer) transformAnyTLS(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "anytls"
	outbound.Password = node.Password

	outbound.TLS = buildTLS(node, nil)

	if interval, ok := node.Extra["idle_session_check_interval"].(string); ok && interval != "" {
		outbound.IdleSessionCheckInterval = interval
	}
	if timeout, ok := node.Extra["idle_session_timeout"].(string); ok && timeout != "" {
		outbound.IdleSessionTimeout = timeout
	}
	if minIdleSession, ok := node.Extra["min_idle_session"]; ok {
		outbound.MinIdleSession = toInt(minIdleSession)
	}

	return outbound, nil
}

func (t *SingBoxTransformer) transformTuic(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "tuic"
	outbound.UUID = node.UUID
	outbound.Password = node.Password

	tls := buildTLS(node, []string{"h3"})
	if disableSNI, ok := node.Extra["disable_sni"].(bool); ok && disableSNI {
		tls["disable_sni"] = true
	}
	outbound.TLS = tls

	if cc, ok := node.Extra["congestion_control"].(string); ok && cc != "" {
		outbound.CongestionControl = cc
	}
	if udpMode, ok := node.Extra["udp_relay_mode"].(string); ok && udpMode != "" {
		outbound.UDPRelayMode = udpMode
	}
	if udpOverStream, ok := node.Extra["udp_over_stream"].(bool); ok && udpOverStream {
		outbound.UDPOverStream = true
	}
	if zeroRtt, ok := node.Extra["zero_rtt_handshake"].(bool); ok && zeroRtt {
		outbound.ZeroRttHandshake = true
	}
	if heartbeat, ok := node.Extra["heartbeat"].(string); ok && heartbeat != "" {
		outbound.Heartbeat = heartbeat
	}
	if network, ok := node.Extra["network"].(string); ok && network != "" {
		outbound.Network = network
	}

	return outbound, nil
}

// toInt 安全地将各种数值类型转换为 int
func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

func (t *SingBoxTransformer) transformShadowsocks(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "shadowsocks"
	outbound.Password = node.Password

	if method, ok := node.Extra["method"].(string); ok {
		outbound.Method = method
	}

	return outbound, nil
}
