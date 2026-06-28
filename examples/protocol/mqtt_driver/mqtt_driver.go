// Package mqtt_driver 提供 MQTT 协议驱动实现示例。
//
// 本文件展示了如何实现 D1 Protocol 接口以适配 MQTT 协议。
// 协议驱动编译为共享库后，由 D1 ConnManager 通过 connector.yaml 配置加载。
//
// 消息格式遵循 JSON-RPC 2.0 规范：{method, params}
//
// 编译命令:
//
//	go build -buildmode=plugin -o libmqtt_driver.so mqtt_driver.go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	// 示例使用 paho.mqtt.golang 作为 MQTT 客户端库
	// 实际使用时需 go get github.com/eclipse/paho.mqtt.golang
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
// MQTT 驱动配置
// ---------------------------------------------------------------------------

// MQTTConfig 定义 MQTT 驱动的配置结构。
// 对应 connector.yaml 中实例的 setting 字段。
type MQTTConfig struct {
	Broker   string   `json:"broker"`    // MQTT Broker 地址，如 tcp://localhost:1883
	ClientID string   `json:"client_id"` // 客户端 ID
	Username string   `json:"username"`  // 用户名（可选）
	Password string   `json:"password"`  // 密码（可选）
	Topics   []string `json:"topics"`    // 订阅主题列表
	QoS      byte     `json:"qos"`       // QoS 级别（0/1/2）
	KeepAlive int     `json:"keepalive"` // 心跳间隔（秒）
}

// ---------------------------------------------------------------------------
// MQTT 驱动实现
// ---------------------------------------------------------------------------

// MQTTDriver MQTT 协议驱动实现。
type MQTTDriver struct {
	mu       sync.Mutex
	config   MQTTConfig
	client   mqtt.Client
	onMsg    OnMessageFunc // 消息回调，由 D1 核心在 Init 后设置
	started  bool
}

// 全局实例（编译为插件时，D1 通过符号查找获取此实例）
var Driver = &MQTTDriver{}

// SetOnMessage 设置消息回调函数。
// 由 D1 核心在加载驱动后调用，用于接收驱动上报的消息。
func (d *MQTTDriver) SetOnMessage(fn OnMessageFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onMsg = fn
}

// Init 初始化 MQTT 驱动。
// config 来自 connector.yaml 的 setting 字段，被解析为 MQTTConfig。
func (d *MQTTDriver) Init(config map[string]interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 解析配置
	cfgBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("mqtt_driver: marshal config failed: %w", err)
	}
	if err := json.Unmarshal(cfgBytes, &d.config); err != nil {
		return fmt.Errorf("mqtt_driver: parse config failed: %w", err)
	}

	// 设置默认值
	if d.config.KeepAlive <= 0 {
		d.config.KeepAlive = 60
	}
	if d.config.Broker == "" {
		return fmt.Errorf("mqtt_driver: broker is required")
	}

	// 创建 MQTT 客户端选项
	opts := mqtt.NewClientOptions().
		AddBroker(d.config.Broker).
		SetClientID(d.config.ClientID).
		SetKeepAlive(d.config.KeepAlive)

	if d.config.Username != "" {
		opts.SetUsername(d.config.Username)
		opts.SetPassword(d.config.Password)
	}

	// 设置连接回调
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("[mqtt_driver] connected to broker: %s", d.config.Broker)
		// 订阅配置的主题
		for _, topic := range d.config.Topics {
			if token := c.Subscribe(topic, d.config.QoS, d.onMQTTMessage); token.Wait() && token.Error() != nil {
				log.Printf("[mqtt_driver] subscribe topic %s failed: %v", topic, token.Error())
			} else {
				log.Printf("[mqtt_driver] subscribed to topic: %s", topic)
			}
		}
	})

	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("[mqtt_driver] connection lost: %v", err)
	})

	d.client = mqtt.NewClient(opts)
	log.Printf("[mqtt_driver] init completed, broker=%s, client_id=%s",
		d.config.Broker, d.config.ClientID)

	return nil
}

// Start 启动 MQTT 驱动，连接 Broker 并开始接收消息。
func (d *MQTTDriver) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client == nil {
		return fmt.Errorf("mqtt_driver: not initialized, call Init first")
	}
	if d.started {
		return fmt.Errorf("mqtt_driver: already started")
	}

	token := d.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt_driver: connect failed: %w", token.Error())
	}

	d.started = true
	log.Printf("[mqtt_driver] started")
	return nil
}

// Stop 停止 MQTT 驱动，断开连接并释放资源。
func (d *MQTTDriver) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.started || d.client == nil {
		return nil
	}

	d.client.Disconnect(250) // 等待 250ms 完成断开
	d.started = false
	log.Printf("[mqtt_driver] stopped")
	return nil
}

// Send 通过 MQTT 发送消息。
// 使用 JSON-RPC 2.0 格式封装消息：{jsonrpc, method, params}
// target 为 MQTT 主题名。
func (d *MQTTDriver) Send(target string, method string, params []byte) error {
	d.mu.Lock()
	client := d.client
	started := d.started
	d.mu.Unlock()

	if !started || client == nil {
		return fmt.Errorf("mqtt_driver: not started")
	}

	// 构建 JSON-RPC 2.0 消息
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  json.RawMessage(params),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("mqtt_driver: marshal message failed: %w", err)
	}

	token := client.Publish(target, d.config.QoS, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("mqtt_driver: publish failed: %w", token.Error())
	}

	log.Printf("[mqtt_driver] sent message: method=%s, target=%s", method, target)
	return nil
}

// ---------------------------------------------------------------------------
// MQTT 消息接收回调
// ---------------------------------------------------------------------------

// onMQTTMessage 处理收到的 MQTT 消息，解析 JSON-RPC 2.0 格式并上报给 D1 核心。
func (d *MQTTDriver) onMQTTMessage(client mqtt.Client, msg mqtt.Message) {
	d.mu.Lock()
	onMsg := d.onMsg
	d.mu.Unlock()

	if onMsg == nil {
		log.Printf("[mqtt_driver] no message handler set, dropping message from topic: %s", msg.Topic())
		return
	}

	// 解析 JSON-RPC 2.0 消息
	var rpcMsg struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal(msg.Payload(), &rpcMsg); err != nil {
		// 如果无法解析为 JSON-RPC 格式，则使用 topic 作为 method，
		// 原始 payload 作为 params 传递
		log.Printf("[mqtt_driver] non-JSON-RPC message from topic %s, passing raw payload", msg.Topic())
		onMsg(msg.Topic(), msg.Payload())
		return
	}

	// 上报给 D1 核心
	onMsg(rpcMsg.Method, rpcMsg.Params)
}