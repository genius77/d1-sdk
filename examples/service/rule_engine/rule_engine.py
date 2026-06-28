#!/usr/bin/env python3
# ============================================================
# D1 Service 扩展示例 —— 规则引擎
#
# 协议: JSON-RPC 2.0 over stdin/stdout (LSP framed)
# 输入: {"jsonrpc":"2.0","method":"xxx","params":{...},"id":1}
# 输出: Content-Length: N\r\n\r\n{"jsonrpc":"2.0","result":{...},"id":1}
#
# 注意: 正式环境推荐使用 gRPC 通信（见 sdk-service.html SDK-03）。
#       本示例使用 stdio 模式便于快速演示。
#
# 对应 D1 版本: >= v1.5.0
# ============================================================

import json
import sys
import os
from datetime import datetime

# ─── 规则定义 ───────────────────────────────────────────
RULES = {
    "risk_score": {
        "condition": lambda d: d.get("amount", 0) > 10000,
        "action": "flag_review",
        "message": "交易金额超过 10000，需要人工审核"
    },
    "fraud_check": {
        "condition": lambda d: d.get("ip_country") != d.get("card_country"),
        "action": "block",
        "message": "IP 所在国与银行卡所在国不匹配"
    },
    "velocity": {
        "condition": lambda d: d.get("recent_tx_count", 0) > 50,
        "action": "flag_review",
        "message": "最近1小时交易次数超过 50 次"
    },
}


def read_request():
    """读取 LSP 帧格式的 JSON-RPC 2.0 请求"""
    # 读取 Content-Length 头
    header = sys.stdin.readline().strip()
    if not header:
        return None
    if not header.startswith("Content-Length:"):
        # 尝试直接解析 JSON（兼容无 LSP 帧的原始输入）
        try:
            return json.loads(header)
        except json.JSONDecodeError:
            return None

    length = int(header.split(":")[1].strip())
    # 跳过分隔空行
    sys.stdin.readline()
    # 读取 body
    body = sys.stdin.read(length)
    try:
        return json.loads(body)
    except json.JSONDecodeError:
        return None


def write_response(response):
    """以 LSP 帧格式输出 JSON-RPC 2.0 响应"""
    output = json.dumps(response, ensure_ascii=False)
    sys.stdout.write(f"Content-Length: {len(output)}\r\n\r\n{output}")
    sys.stdout.flush()


def handle_evaluate(params):
    """评估规则"""
    event = params.get("event", {})
    hits = []
    for rule_name, rule in RULES.items():
        if rule["condition"](event):
            hits.append({
                "rule": rule_name,
                "action": rule["action"],
                "message": rule["message"]
            })

    return {
        "timestamp": datetime.now().isoformat(),
        "rules_checked": len(RULES),
        "hits": len(hits),
        "results": hits
    }


def handle_list_rules(_params):
    """列出所有规则"""
    rules_info = {}
    for name, rule in RULES.items():
        rules_info[name] = {
            "action": rule["action"],
            "message": rule["message"]
        }
    return {"rules": rules_info}


# ─── 主循环 ─────────────────────────────────────────────
sys.stderr.write(f"规则引擎已启动, PID: {os.getpid()}\n")
sys.stderr.flush()

while True:
    try:
        request = read_request()
    except EOFError:
        break

    if request is None:
        continue

    # 提取 JSON-RPC 2.0 字段
    method = request.get("method", "")
    params = request.get("params", {})
    req_id = request.get("id")

    # 路由
    try:
        if method == "evaluate":
            result = handle_evaluate(params)
        elif method == "list_rules":
            result = handle_list_rules(params)
        elif method == "health":
            result = {"status": "ok", "pid": os.getpid()}
        else:
            response = {
                "jsonrpc": "2.0",
                "error": {"code": -32601, "message": f"未知方法: {method}"},
                "id": req_id
            }
            write_response(response)
            continue

        response = {
            "jsonrpc": "2.0",
            "result": result,
            "id": req_id
        }
        write_response(response)

    except Exception as e:
        response = {
            "jsonrpc": "2.0",
            "error": {"code": -32000, "message": str(e)},
            "id": req_id
        }
        write_response(response)