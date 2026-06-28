# cache_service — 缓存服务示例

Shell + Python 实现的 D1 缓存服务示例。

## 协议

JSON-RPC 2.0 over stdin/stdout (LSP framed)。

- 输入: `{"jsonrpc":"2.0","method":"xxx","params":{...},"id":1}`
- 输出: `Content-Length: N\r\n\r\n{"jsonrpc":"2.0","result":{...},"id":1}`

注意: 正式环境推荐使用 gRPC 通信（见 sdk-service.html SDK-03）。

## 支持的方法

| method | 说明 | params |
|--------|------|--------|
| `cache.get` | 获取缓存值 | `{"key": "..."}` |
| `cache.set` | 设置缓存值 | `{"key": "...", "value": ...}` |
| `cache.delete` | 删除缓存 | `{"key": "..."}` |
| `cache.keys` | 列出所有键 | `{}` |
| `health` | 健康检查 | `{}` |

## D1 配置

在 `service.yaml` 中配置：

```yaml
items:
  - name: "cache_service"
    command: "./extensions/service/cache_service/cache_service.sh"
```