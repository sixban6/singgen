package transformer

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"github.com/sixban6/singgen/internal/constant"
	"github.com/sixban6/singgen/internal/util"
	"github.com/sixban6/singgen/pkg/model"
	"go.uber.org/zap"
)

type Outbound struct {
	Type                     string         `json:"type"`
	Tag                      string         `json:"tag"`
	Server                   string         `json:"server"`
	ServerPort               uint16         `json:"server_port"`
	UUID                     string         `json:"uuid,omitempty"`
	Password                 string         `json:"password,omitempty"`
	Method                   string         `json:"method,omitempty"`
	Flow                     string         `json:"flow,omitempty"`
	UpMbps                   int            `json:"up_mbps,omitempty"`
	DownMbps                 int            `json:"down_mbps,omitempty"`
	IdleSessionCheckInterval string         `json:"idle_session_check_interval,omitempty"`
	IdleSessionTimeout       string         `json:"idle_session_timeout,omitempty"`
	MinIdleSession           int            `json:"min_idle_session,omitempty"`
	Transport                map[string]any `json:"transport,omitempty"`
	TLS                      map[string]any `json:"tls,omitempty"`
	Multiplex                map[string]any `json:"multiplex,omitempty"`
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

type SingBoxTransformer struct{}

func NewSingBoxTransformer() *SingBoxTransformer {
	return &SingBoxTransformer{}
}

func (t *SingBoxTransformer) Transform(nodes []model.Node) ([]Outbound, error) {
	if len(nodes) == 0 {
		return []Outbound{}, nil
	}

	// 对于少量节点，使用顺序处理避免并发开销
	if len(nodes) <= 10 {
		return t.transformSequential(nodes)
	}

	// 大量节点使用并发处理
	return t.transformConcurrent(nodes)
}

// transformSequential 顺序处理节点转换
func (t *SingBoxTransformer) transformSequential(nodes []model.Node) ([]Outbound, error) {
	var outbounds []Outbound

	for _, node := range nodes {
		outbound, err := t.transformNode(node)
		if err != nil {
			if util.L != nil {
				util.L.Warn("Failed to transform node", zap.String("tag", node.Tag), zap.Error(err))
			}
			continue
		}
		outbounds = append(outbounds, *outbound)
	}

	return outbounds, nil
}

// transformConcurrent 并发处理节点转换
func (t *SingBoxTransformer) transformConcurrent(nodes []model.Node) ([]Outbound, error) {
	// 计算合适的worker数量
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > len(nodes) {
		numWorkers = len(nodes)
	}

	if numWorkers > 8 {
		numWorkers = 8
	}

	if util.L != nil {
		util.L.Debug("Starting concurrent transformation",
			zap.Int("nodes", len(nodes)),
			zap.Int("workers", numWorkers))
	}

	// 创建任务通道
	jobs := make(chan transformJob, len(nodes))
	results := make(chan transformResult, len(nodes))

	// 启动worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go t.worker(jobs, results, &wg)
	}

	// 发送任务
	go func() {
		defer close(jobs)
		for i, node := range nodes {
			jobs <- transformJob{
				index: i,
				node:  node,
			}
		}
	}()

	// 等待所有worker完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	// 标记有效结果，用于后续过滤（如果有错误）
	// Initialize outbounds with nil semantics if possible, or track validity
	// Here we will use a separate slice or map if we needed to filter, but since we return empty/error on individual failure in sequential,
	// let's mirror sequential behavior: log warning and skip.
	// But `outbounds` is pre-allocated. If we skip, we'll have empty structs unless we compact it.
	// To perform compaction strictly preserving order, we still need index.

	type resultWithIndex struct {
		index int
		val   Outbound
	}
	var validResults []resultWithIndex

	for result := range results {
		if result.err != nil {
			if util.L != nil {
				util.L.Warn("Failed to transform node", zap.String("tag", result.originalTag), zap.Error(result.err))
			}
			continue
		}
		validResults = append(validResults, resultWithIndex{index: result.index, val: result.outbound})
	}

	// Re-order results
	// Since we might skip some, the output length <= input length
	// We need to sort validResults by index to preserve relative order of successful ones

	// Create a slice large enough
	finalOutbounds := make([]Outbound, 0, len(validResults))

	// Sort validResults by index
	// Since we don't want to import sort package if not needed, let's just use a fixed size array and then iterate?
	// Actually, we can just use an array of pointers or struct with a Valid flag.

	// Better approach:
	tempResults := make([]*Outbound, len(nodes))
	for _, res := range validResults {
		cp := res.val
		tempResults[res.index] = &cp
	}

	for _, res := range tempResults {
		if res != nil {
			finalOutbounds = append(finalOutbounds, *res)
		}
	}

	return finalOutbounds, nil
}

// transformJob 转换任务
type transformJob struct {
	index int
	node  model.Node
}

// transformResult 转换结果
type transformResult struct {
	index       int
	outbound    Outbound
	originalTag string
	err         error
}

// worker 并发处理worker
func (t *SingBoxTransformer) worker(jobs <-chan transformJob, results chan<- transformResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		outbound, err := t.transformNode(job.node)
		result := transformResult{
			index:       job.index,
			originalTag: job.node.Tag,
			err:         err,
		}

		if err == nil && outbound != nil {
			result.outbound = *outbound
		}

		results <- result
	}
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
	case constant.ProtocolSS:
		return t.transformShadowsocks(node, outbound)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", node.Type)
	}
}

func (t *SingBoxTransformer) transformVmess(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "vmess"
	outbound.UUID = node.UUID

	if node.Security.TLS {
		outbound.TLS = map[string]any{
			"enabled":     true,
			"insecure":    node.Security.SkipVerify,
			"server_name": node.Security.ServerName,
		}
		if len(node.Security.ALPN) > 0 {
			outbound.TLS["alpn"] = node.Security.ALPN
		}
	}

	if node.Transport.Net != "" && node.Transport.Net != "tcp" {
		transport := make(map[string]any)
		transport["type"] = node.Transport.Net

		if node.Transport.Host != "" {
			transport["host"] = node.Transport.Host
		}
		if node.Transport.Path != "" {
			transport["path"] = node.Transport.Path
		}
		if len(node.Transport.Headers) > 0 {
			transport["headers"] = node.Transport.Headers
		}

		outbound.Transport = transport
	}

	if alterId, ok := node.Extra["alter_id"].(int); ok {
		outbound.Multiplex = map[string]any{
			"enabled": alterId > 0,
		}
	}

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
		tls := map[string]any{
			"enabled":     true,
			"insecure":    node.Security.SkipVerify,
			"server_name": node.Security.ServerName,
		}

		if len(node.Security.ALPN) > 0 {
			tls["alpn"] = node.Security.ALPN
		}

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

			// Set uTLS fingerprint for Reality
			if fp, ok := node.Extra["fingerprint"].(string); ok && fp != "" {
				tls["utls"] = map[string]any{
					"enabled":     true,
					"fingerprint": fp,
				}
			}
		}

		outbound.TLS = tls
	}

	if node.Transport.Net != "" && node.Transport.Net != "tcp" {
		transport := make(map[string]any)
		transport["type"] = node.Transport.Net

		if node.Transport.Host != "" {
			transport["host"] = node.Transport.Host
		}
		if node.Transport.Path != "" {
			transport["path"] = node.Transport.Path
		}
		if len(node.Transport.Headers) > 0 {
			transport["headers"] = node.Transport.Headers
		}

		outbound.Transport = transport
	}

	return outbound, nil
}

func (t *SingBoxTransformer) transformTrojan(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "trojan"
	outbound.Password = node.Password

	outbound.TLS = map[string]any{
		"enabled":     true,
		"insecure":    node.Security.SkipVerify,
		"server_name": node.Security.ServerName,
	}
	if len(node.Security.ALPN) > 0 {
		outbound.TLS["alpn"] = node.Security.ALPN
	}

	if node.Transport.Net != "" && node.Transport.Net != "tcp" {
		transport := make(map[string]any)
		transport["type"] = node.Transport.Net

		if node.Transport.Host != "" {
			transport["host"] = node.Transport.Host
		}
		if node.Transport.Path != "" {
			transport["path"] = node.Transport.Path
		}
		if len(node.Transport.Headers) > 0 {
			transport["headers"] = node.Transport.Headers
		}

		outbound.Transport = transport
	}

	return outbound, nil
}

func (t *SingBoxTransformer) transformHysteria2(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "hysteria2"
	outbound.Password = node.Password

	outbound.TLS = map[string]any{
		"enabled":     true,
		"insecure":    node.Security.SkipVerify,
		"server_name": node.Security.ServerName,
	}
	if len(node.Security.ALPN) > 0 {
		outbound.TLS["alpn"] = node.Security.ALPN
	}

	// 设置带宽限制参数（防止运营商检测异常流量）
	if upMbps, ok := node.Extra["up_mbps"]; ok {
		outbound.UpMbps = toInt(upMbps)
	}
	if downMbps, ok := node.Extra["down_mbps"]; ok {
		outbound.DownMbps = toInt(downMbps)
	}

	return outbound, nil
}

func (t *SingBoxTransformer) transformAnyTLS(node model.Node, outbound *Outbound) (*Outbound, error) {
	outbound.Type = "anytls"
	outbound.Password = node.Password

	outbound.TLS = map[string]any{
		"enabled":     true,
		"insecure":    node.Security.SkipVerify,
		"server_name": node.Security.ServerName,
	}
	if len(node.Security.ALPN) > 0 {
		outbound.TLS["alpn"] = node.Security.ALPN
	}

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
