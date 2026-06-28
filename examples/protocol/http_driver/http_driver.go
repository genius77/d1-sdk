// Package http_driver 提供 HTTP 协议驱动实现示例。
//
// 本文件展示了如何实现 D1 Protocol 接口以适配 HTTP/WebSocket 协议。
// 协议驱动编译为共享库后，由 D1 ConnManager 通过 connector.yaml 配置加载。
//
// 消息格式遵循 JSON-RPC 2.0 规范：{method, params}
//
// 编译命令:
//
//	go build -buildmode=plugin -o libhttp_driver.so http_driver.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

// ---------------------------------------------------------------------------
// Protocol 接口定义（与 D1 核心接口一致）
// ---------------------------------------------------------------------------

// Protocol 协议驱动接口，所有协议驱动必须实现此接口。
type Protocol interface {
	// Init 初始化协议驱动实例。
	// config 为驱动配置（来自 connector.yaml 的 setting 字段）。
	Init(config map[string]interface{}) error

	// Start 启动协议驱动，建立连接并开始接收消息。
	Start() error

	// Stop 停止协议驱动，断开连接并释放资源。
	Stop() error

	// Send 发送消息到指定目标。
	// method: JSON-RPC 方法名
	// params: 方法参数（原始 JSON 字节）
	Send(target string, method string, params []byte) error
}

// OnMessageFunc 收到消息时的回调函数类型。
// 协议驱动通过此回调将收到的消息上报给 D1 核心。
type OnMessageFunc func(method string, params []byte)

// ---------------------------------------------------------------------------
// HTTP 驱动配置
// ---------------------------------------------------------------------------

// HTTPConfig 定义 HTTP 驱动的配置结构。
// 对应 connector.yaml 中实例的 setting 字段。
type HTTPConfig struct {
	Host       string `json:"host"`        // 监听地址，如 "0.0.0.0"
	Port       int    `json:"port"`        // 监听端口，如 8080
	ReadTimeout  int  `json:"read_timeout"` // 读取超时（秒），默认 30
	WriteTimeout int  `json:"write_timeout"` // 写入超时（秒），默认 30
}

// ---------------------------------------------------------------------------
// HTTP 驱动实现
// ---------------------------------------------------------------------------

// HTTPDriver HTTP 协议驱动实现。
// 提供 HTTP 服务端功能，接收 JSON-RPC 2.0 格式的 POST 请求。
type HTTPDriver struct {
	mu      sync.Mutex
	config  HTTPConfig
	server  *http.Server
	onMsg   OnMessageFunc // 消息回调，由 D1 核心在 Init 后设置
	started bool
}

// 全局实例（编译为插件时，D1 通过符号查找获取此实例）
var Driver = &HTTPDriver{}

// SetOnMessage 设置消息回调函数。
// 由 D1 核心在加载驱动后调用，用于接收驱动上报的消息。
func (d *HTTPDriver) SetOnMessage(fn OnMessageFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onMsg = fn
}

// Init 初始化 HTTP 驱动。
// config 来自 connector.yaml 的 setting 字段，被解析为 HTTPConfig。
func (d *HTTPDriver) Init(config map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 解析配置
	cfgBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("http_driver: marshal config failed: %w", err)
	}
	if err := json.Unmarshal(cfgBytes, &d.config); err != nil {
		return fmt.Errorf("http_driver: parse config failed: %w", err)
	}

	// 设置默认值
	if d.config.Port == 0 {
		d.config.Port = 8080
	}
	if d.config.Host == "" {
		d.config.Host = "0.0.0.0"
	}
	if d.config.ReadTimeout <= 0 {
		d.config.ReadTimeout = 30
	}
	if d.config.WriteTimeout <= 0 {
		d.config.WriteTimeout = 30
	}

	log.Printf("[http_driver] init completed, host=%s, port=%d",
		d.config.Host, d.config.Port)

	return nil
}

// Start 启动 HTTP 驱动，开始监听端口并接收消息。
func (d *HTTPDriver) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.started {
		return fmt.Errorf("http_driver: already started")
	}

	// 创建 HTTP 路由
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"driver": "http_driver",
		})
	})

	// JSON-RPC 2.0 端点 —— 接收 D1 消息
	mux.HandleFunc("/", d.handleJSONRPC)

	addr := fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	d.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  d.config.ReadTimeout,
		WriteTimeout: d.config.WriteTimeout,
	}

	// 在 goroutine 中启动 HTTP 服务
	go func() {
		log.Printf("[http_driver] listening on %s", addr)
		if err := d.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[http_driver] server error: %v", err)
		}
	}()

	d.started = true
	log.Printf("[http_driver] started")
	return nil
}

// Stop 停止 HTTP 驱动，关闭 HTTP 服务并释放资源。
func (d *HTTPDriver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.started || d.server == nil {
		return nil
	}

	// 关闭 HTTP 服务器（优雅关闭）
	if err := d.server.Close(); err != nil {
		log.Printf("[http_driver] close server error: %v", err)
	}

	d.started = false
	log.Printf("[http_driver] stopped")
	return nil
}

// Send 通过 HTTP 发送消息到目标 URL。
// 使用 JSON-RPC 2.0 格式封装消息：{jsonrpc, method, params}
// target 为目标 URL。
func (d *HTTPDriver) Send(target string, method string, params []byte) error {
	// 构建 JSON-RPC 2.0 消息
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  json.RawMessage(params),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("http_driver: marshal message failed: %w", err)
	}

	// 发送 HTTP POST 请求
	// 注意：这里使用 http.Post 发送，实际驱动中可能需要更复杂的 HTTP 客户端配置
	resp, err := http.Post(target, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("http_driver: send failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("http_driver: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[http_driver] sent message: method=%s, target=%s", method, target)
	return nil
}

// ---------------------------------------------------------------------------
// HTTP 请求处理
// ---------------------------------------------------------------------------

// handleJSONRPC 处理 JSON-RPC 2.0 格式的 HTTP POST 请求。
// 解析请求体中的 method 和 params 字段，通过回调上报给 D1 核心。
func (d *HTTPDriver) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("read body failed: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析 JSON-RPC 2.0 请求
	var rpcReq struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
	}

	if err := json.Unmarshal(body, &rpcReq); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON-RPC request: %v", err), http.StatusBadRequest)
		return
	}

	// 验证 JSON-RPC 版本
	if rpcReq.JSONRPC != "" && rpcReq.JSONRPC != "2.0" {
		http.Error(w, "invalid JSON-RPC version, expected 2.0", http.StatusBadRequest)
		return
	}

	// 获取回调函数
	d.mu.Lock()
	onMsg := d.onMsg
	d.mu.Unlock()

	if onMsg == nil {
		http.Error(w, "no message handler set", http.StatusInternalServerError)
		return
	}

	// 路径也作为 method 的一部分（用于路由匹配）
	// 例如: POST /api/v1/device → method = "/api/v1/device"
	effectiveMethod := rpcReq.Method
	if effectiveMethod == "" {
		effectiveMethod = r.URL.Path
	}

	log.Printf("[http_driver] received request: method=%s, path=%s", effectiveMethod, r.URL.Path)

	// 上报给 D1 核心
	onMsg(effectiveMethod, rpcReq.Params)

	// 返回 JSON-RPC 2.0 成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  "accepted",
		"id":      rpcReq.ID,
	})
}

