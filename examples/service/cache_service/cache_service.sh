#!/usr/bin/env bash
# ============================================================
# cache_service.sh — D1 Service 扩展 · 缓存服务示例
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
set -euo pipefail

# 临时数据文件（进程生命周期内有效）
DATA_FILE="/tmp/d1_cache_service_$$.json"
echo "{}" > "$DATA_FILE"

cleanup() {
    rm -f "$DATA_FILE"
    exit 0
}
trap cleanup SIGTERM SIGINT EXIT

echo "缓存服务已启动, PID: $$" >&2

# ─── 主循环：读取 LSP 帧格式的 JSON-RPC 2.0 请求 ────────
while IFS= read -r header; do
    # 跳过空行
    [[ -z "$header" ]] && continue

    # 尝试解析 Content-Length
    if [[ "$header" =~ ^Content-Length:\ ([0-9]+)$ ]]; then
        length="${BASH_REMATCH[1]}"
        # 跳过空行
        read -r _
        # 读取 body
        read -r -n "$length" body
        # 消费剩余换行
        read -r _ || true
    else
        # 兼容无 LSP 帧的原始 JSON 输入
        body="$header"
        length=${#body}
    fi

    # 解析 JSON-RPC 2.0
    method=$(echo "$body" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    print(req.get('method', ''))
except:
    print('')
" 2>/dev/null || echo "")

    params=$(echo "$body" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    print(json.dumps(req.get('params', {})))
except:
    print('{}')
" 2>/dev/null || echo "{}")

    req_id=$(echo "$body" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    rid = req.get('id')
    print(json.dumps(rid) if rid is not None else 'null')
except:
    print('null')
" 2>/dev/null || echo "null")

    # 根据 method 路由
    case "$method" in
        "cache.get")
            key=$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(p.get('key', ''))
" 2>/dev/null || echo "")
            result=$(python3 -c "
import json
try:
    with open('$DATA_FILE') as f:
        d = json.load(f)
    v = d.get('$key')
    print(json.dumps(v) if v is not None else 'null')
except:
    print('null')
" 2>/dev/null || echo "null")
            ;;

        "cache.set")
            key=$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(p.get('key', ''))
" 2>/dev/null || echo "")
            value=$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(json.dumps(p.get('value')))
" 2>/dev/null || echo "null")
            python3 -c "
import json
try:
    with open('$DATA_FILE') as f:
        d = json.load(f)
except:
    d = {}
d['$key'] = json.loads('''$value''')
with open('$DATA_FILE', 'w') as f:
    json.dump(d, f)
" 2>/dev/null
            result='{"ok":true}'
            ;;

        "cache.delete")
            key=$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(p.get('key', ''))
" 2>/dev/null || echo "")
            python3 -c "
import json
try:
    with open('$DATA_FILE') as f:
        d = json.load(f)
except:
    d = {}
d.pop('$key', None)
with open('$DATA_FILE', 'w') as f:
    json.dump(d, f)
" 2>/dev/null
            result='{"ok":true}'
            ;;

        "cache.keys")
            result=$(python3 -c "
import json
try:
    with open('$DATA_FILE') as f:
        d = json.load(f)
    print(json.dumps(list(d.keys())))
except:
    print('[]')
" 2>/dev/null || echo "[]")
            ;;

        "health")
            result='{"status":"ok","pid":'$$'}'
            ;;

        *)
            result='{"error":"unknown method: '"$method"'"}'
            ;;
    esac

    # JSON-RPC 2.0 响应
    output='{"jsonrpc":"2.0","result":'"$result"',"id":'$req_id'}'

    # LSP 帧格式输出
    len=${#output}
    printf "Content-Length: %d\r\n\r\n%s" "$len" "$output"
done