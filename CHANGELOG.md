# Changelog

本文档记录 d1-sdk 仓库的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [0.3.0] — 2026-06-25

### 变更
- **协议升级**: 所有扩展和示例统一采用 JSON-RPC 2.0 协议（`method`/`params`/`result`/`error`）
- **术语统一**: `msgName` → `method`，`payload` → `params`（Go SDK 及所有扩展示例）
- **C API 参数名更新**: `api/d1.h` 所有函数签名中 `msg_name` → `method`、`payload` → `params`（与 D1 v1.5.0 对齐）
- **FFI 绑定同步**: C/C++/Python/C#/Java 绑定全部更新为 `method`/`params` 参数名
- **新增**: `extensions/protocol/` 目录，包含 MQTT 和 HTTP 协议驱动示例
- **exec 扩展**: Shell 脚本输出格式改为 JSON-RPC 2.0（`{"result": ...}` / `{"error": {...}}`）
- **script 扩展**: JS 脚本输入改为 `{method, params}`，`d1.call()` 移除 `kind` 参数
- **service 扩展**: Python 规则引擎改为 JSON-RPC 2.0 请求/响应格式
- **Go SDK**: `HandlerFunc`、`Publish`、`Call`、`Request`、`Reply` 签名中 `payload` → `params`
- **示例更新**: C/C++/Python/C#/Java 示例中的处理器参数名更新

### 依赖
- 对应 D1 动态库版本: **≥ v1.5.0**

---

## [0.2.0] — 2026-06-10

### 变更
- **项目重命名**: `d1-examples` → `d1-sdk`
- **目录重命名**: `sdk/` → `lang/`
- **API 变更**: 移除 `Wait()`，新增 `WaitStop()`（与 D1 ≥ v1.2.0 对齐）
- **命名统一**: 配置文件 `scripts.yaml`/`execs.yaml`/`services.yaml` → 单数形式
- **工程化**: 引入 `project-workflow` submodule 作为版本管理规范
- **示例完善**: C++/C#/Java 示例统一使用 `WaitStop()` 简化流程

### 依赖
- 对应 D1 动态库版本: **≥ v1.2.0**

---

## [0.1.0] — 2026-06-09

### 新增 ✨

- **SDK 封装层** (`lang/`)：为 6 种语言提供 D1 全部 17 个 C API 的封装
  - C 头文件 (`lang/c/d1.h`)
  - C++ RAII 封装 (`lang/cpp/d1.hpp`)
  - Go cgo 封装 (`lang/go/d1.go`)
  - Python ctypes 封装 (`lang/python/d1.py`)
  - C# P/Invoke 封装 (`lang/csharp/D1.cs`)
  - Java JNA 封装 (`lang/java/D1.java`)

- **入门示例** (`examples/`)：每种语言一个最小可运行示例
  - `examples/c/01_hello_d1/` — C 语言入门
  - `examples/cpp/01_hello_d1/` — C++ 入门
  - `examples/go/01_hello_d1/` — Go 入门
  - `examples/python/01_hello_d1/` — Python 入门
  - `examples/csharp/01_hello_d1/` — C# 入门
  - `examples/java/01_hello_d1/` — Java 入门

- **工具脚本** (`scripts/`)
  - `download_d1.sh` — Unix/Linux/macOS 一键下载 D1 动态库
  - `download_d1.ps1` — Windows 一键下载 D1 动态库

- **示例配置文件** (`config/`)
  - `d1.yaml` / `connector.yaml` / `router.yaml` 等 D1 配置示例

- **仓库基础设施**
  - `VERSION` — 当前仓库版本
  - `CHANGELOG.md` — 变更日志
  - `README.md` — 项目说明与快速开始指南

### 依赖

- 对应 D1 动态库版本：**≥ v1.1.0**

---

[0.3.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.3.0
[0.2.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.2.0
[0.1.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.1.0