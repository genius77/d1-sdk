#!/usr/bin/env bash
# ============================================================
# status_check.sh — D1 Exec 扩展 · 系统状态检查
#
# 协议: JSON-RPC 2.0 over stdin/stdout (LSP framed)
# 输入: {"jsonrpc":"2.0","method":"status","params":{"check":"disk|memory|cpu"},"id":1}
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
    print(req.get('method', 'status'))
except:
    print('status')
" <<< "$input" 2>/dev/null || echo "status")

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

check=$(echo "$params" | python3 -c "
import sys, json
p = json.loads(sys.stdin.read())
print(p.get('check', 'all'))
" 2>/dev/null || echo "all")

# 收集系统状态
disk_info=""
memory_info=""
cpu_info=""

case "$check" in
    disk|all)
        disk_info=$(df -h / 2>/dev/null | tail -1 | awk '{printf "{\"filesystem\":\"%s\",\"size\":\"%s\",\"used\":\"%s\",\"avail\":\"%s\",\"use_pct\":\"%s\"}", $1, $2, $3, $4, $5}')
        ;;
esac

case "$check" in
    memory|all)
        memory_free=$(free -m 2>/dev/null | awk '/^Mem:/{print $4}')
        memory_total=$(free -m 2>/dev/null | awk '/^Mem:/{print $2}')
        memory_info="{\"free_mb\":${memory_free:-0},\"total_mb\":${memory_total:-0}}"
        ;;
esac

case "$check" in
    cpu|all)
        cpu_load=$(uptime 2>/dev/null | awk -F'load average:' '{print $2}' | tr -d ' ')
        cpu_info="{\"load\":\"${cpu_load:-unknown}\"}"
        ;;
esac

# JSON-RPC 2.0 响应
result="{\"disk\":${disk_info:-null},\"memory\":${memory_info:-null},\"cpu\":${cpu_info:-null}}"
output='{"jsonrpc":"2.0","result":'"$result"',"id":'$req_id'}'

# LSP 帧格式输出
len=${#output}
printf "Content-Length: %d\r\n\r\n%s" "$len" "$output"