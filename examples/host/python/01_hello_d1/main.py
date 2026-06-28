#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
示例: D1 Hello World (Python)

本示例演示 D1 Python SDK 的完整用法:
  1. 获取版本号
  2. 初始化 D1 运行时
  3. 设置默认消息处理器（演示 API：Publish、Call、CacheSet/CacheGet、DBQuery）
  4. 启动 D1
  5. 阻塞等待退出（Ctrl+C）

使用前请确保已将 libd1.so 和 d1.h 放入 deps/ 目录。

运行方式:
    cd examples/host/python/01_hello_d1
    python3 main.py
"""

import json
import os
import sys

# 将 lang/ 目录加入 Python 路径
_sdk_path = os.path.normpath(
    os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", "..", "lang", "python")
)
sys.path.insert(0, _sdk_path)

from d1 import D1


def default_handler(task_id, method, params):
    """默认消息处理器。

    演示所有 D1 API 的调用方式：
    - Publish: 发送单向消息
    - CacheSet/CacheGet: 缓存读写
    - Call: 同步调用
    - DBQuery: 数据库查询
    """
    params_str = params.decode("utf-8") if params else ""
    print(f"[Handler] taskID={task_id} | method={method} | params={params_str}")

    # 1. Publish — 发送单向消息（无回复）
    pub_data = json.dumps({"temp": 25.5, "unit": "celsius"})
    d1.publish(task_id, "mqtt_client", "sensor.data", pub_data)
    print("  D1_Publish -> OK")

    # 2. CacheSet — 写入缓存
    cache_val = json.dumps({"name": "Alice", "role": "admin"})
    d1.cache_set(task_id, "user:42", cache_val, 3600)
    print("  D1_CacheSet -> OK")

    # 3. CacheGet — 读取缓存
    cached = d1.cache_get(task_id, "user:42")
    print(f"  D1_CacheGet -> {cached.decode() if cached else 'null'}")

    # 4. Call — 同步调用（阻塞等待结果）
    # kind: 0=conn, 1=default, 2=script, 3=service, 4=exec
    result = d1.call(task_id, 2, "api_handler", "get_user",
                     json.dumps({"id": 123}), 5)
    if result["error"] is None:
        print(f"  D1_Call -> {result['payload'].decode()}")
    else:
        print(f"  D1_Call error -> {result['error']}")

    # 5. DBQuery — 数据库查询
    rows = d1.db_query(task_id, "SELECT * FROM users LIMIT 1")
    print(f"  D1_DBQuery -> {rows.decode() if rows else 'null'}")

    # 返回响应
    return json.dumps({"status": "ok", "msg": "hello from Python handler"}).encode()


def main():
    """主函数: D1 Hello World 示例。"""
    print("===== D1 Python Hello World =====")

    d1 = D1()

    # 1. 输出版本信息
    ver = d1.version()
    print(f"D1 Version: {ver}")

    # 2. 初始化 D1 运行时
    config_path = os.environ.get("D1_CONFIG", None)
    d1.init(config_path)
    print("D1::init OK")

    # 3. 设置默认消息处理器
    d1.set_on_request(default_handler)
    print("Default handler registered")

    # 4. 启动 D1 运行时
    d1.start()
    print("D1::start OK, running (press Ctrl+C to exit)")

    # 5. 阻塞等待退出
    d1.wait_stop()
    print("D1 stopped, exiting.")


if __name__ == "__main__":
    main()