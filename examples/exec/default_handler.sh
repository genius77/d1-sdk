#!/usr/bin/env bash
# ============================================================
# default_handler.sh — D1 Exec 扩展 · 默认消息处理器
#
# 协议: JSON-RPC 2.0 over stdin/stdout (LSP framed)
# 输入: {"jsonrpc":"2.0","method":"xxx","params":{...},"id":1}
# 输出: Content-Length: N\r\n\r\n{"jsonrpc":"2.0","result":{...},"id":1}
# ============================================================
set -euo pipefail

# 读取 stdin（JSON-RPC 2.0 请求）
read -r input

# 解析 method 和 params
method=$(echo "$input" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    print(req.get('method', ''))
except:
    print('')
" <<< "$input" 2>/dev/null || echo "")

params=$(echo "$input" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    print(json.dumps(req.get('params', {})))
except:
    print('{}')
" <<< "$input" 2>/dev/null || echo "{}")

req_id=$(echo "$input" | python3 -c "
import sys, json
try:
    req = json.loads(sys.stdin.read())
    print(json.dumps(req.get('id', 0)))
except:
    print('0')
" <<< "$input" 2>/dev/null || echo "0")

# 根据 method 路由
case "$method" in
    "echo")
        result="$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(json.dumps({'echo': p.get('message', 'no message'), 'timestamp': p.get('timestamp', 0)}))
" 2>/dev/null || echo '{"echo":"error"}')"
        ;;

    "health")
        result='{"status":"ok","uptime":"'$(uptime -p | sed 's/^up //')'"}'
        ;;

    "system.info")
        result='{"hostname":"'$(hostname)'","os":"'$(uname -s)'","arch":"'$(uname -m)'","cwd":"'$(pwd)'"}'
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