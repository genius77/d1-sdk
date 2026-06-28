# 协议驱动扩展

本目录包含 D1 协议驱动扩展的参考实现和文档。

## 概述

协议驱动（Protocol Driver）是 D1 框架中负责底层通信协议适配的 Go 插件模块。
每个协议驱动实现统一的 `Protocol` 接口，将不同通信协议（MQTT、HTTP、Modbus 等）
的数据收发适配为 D1 内部统一的消息流转格式。

## Protocol 接口

协议驱动必须实现以下 `Protocol` 接口：

```go
type Protocol interface {
    // Init 初始化协议驱动实例。
    // config 为驱动配置（通常来自 connector.yaml 的 setting 字段）。
    Init(config map[string]interface{}) error

    // Start 启动协议驱动，开始接收和发送消息。
    Start() error

    // Stop 停止协议驱动，释放连接资源。
    Stop() error

    // Send 发送消息到目标。
    // 消息格式使用 JSON-RPC 2.0 协议，包含 method 和 params 字段。
    Send(target string, method string, params []byte) error
}
```

## 消息格式

协议驱动与 D1 核心之间使用 JSON-RPC 2.0 格式交换消息：

```json
{
    "jsonrpc": "2.0",
    "method": "device.report",
    "params": {
        "temperature": 25.3,
        "humidity": 60.1
    },
    "id": 1
}
```

- `method` - 消息名称，用于路由匹配
- `params` - 方法参数，可以是任意 JSON 值（对象、数组、字符串等）

## 回调机制

协议驱动通过以下回调函数向上层 D1 核心报告收到消息：

```go
// OnMessage 当收到来自外部设备的消息时回调。
// method: 消息名称
// params: 方法参数（原始字节）
type OnMessageFunc func(method string, params []byte)
```

## 目录结构

```
extensions/protocol/
├── README.md              # 本文档
├── mqtt_driver/           # MQTT 协议驱动示例
│   └── mqtt_driver.go     # 驱动实现骨架
└── http_driver/           # HTTP 协议驱动示例
    └── http_driver.go     # 驱动实现骨架
```

## 如何开发新的协议驱动

1. 在本目录下创建新的驱动目录，如 `modbus_driver/`
2. 实现 `Protocol` 接口的四个方法
3. 在 `connector.yaml` 中添加对应的驱动实例配置
4. 编译为共享库（`.so` / `.dll`），放入 `deps/` 目录
5. 重启 D1 并进行测试

## 参考

- D1 主配置文件: `config/d1.yaml`
- 连接器配置: `config/connector.yaml`
- 路由配置: `config/router.yaml`