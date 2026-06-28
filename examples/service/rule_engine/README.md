# rule_engine — 规则引擎示例

Python 实现的 D1 规则引擎服务示例。

## 协议

JSON-RPC 2.0 over stdin/stdout (LSP framed)。

- 输入: `{"jsonrpc":"2.0","method":"xxx","params":{...},"id":1}`
- 输出: `Content-Length: N\r\n\r\n{"jsonrpc":"2.0","result":{...},"id":1}`

注意: 正式环境推荐使用 gRPC 通信（见 sdk-service.html SDK-03）。

## 规则示例

| 规则 | 触发条件 | 动作 |
|------|----------|------|
| `risk_score` | 金额 > 10000 | 需要人工审核 |
| `fraud_check` | IP 国家 ≠ 卡国家 | 拦截 |
| `velocity` | 1小时交易 > 50笔 | 需要人工审核 |

## 支持的方法

| method | 说明 | params |
|--------|------|--------|
| `evaluate` | 评估规则 | `{"event": {...}}` |
| `list_rules` | 列出所有规则 | `{}` |
| `health` | 健康检查 | `{}` |

## D1 配置

在 `service.yaml` 中配置：

```yaml
items:
  - name: "rule_engine"
    command: "python3 ./extensions/service/rule_engine/rule_engine.py"
```