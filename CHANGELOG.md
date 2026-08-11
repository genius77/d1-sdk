# Changelog

本文档记录 d1-sdk 仓库的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，版本号遵循 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

---

## [0.6.0] — 2026-08-11

### 变更

#### 跨语言 API 一致性改进

- **Python SDK**: `call()` 返回类型从 `dict` 改为 `CallResult` namedtuple，包含 `return_code`、`payload`、`error` 字段及 `ok()` 方法，与 C++ `CallResult` / C# `D1CallResult` / Java `CallResult` 保持一致
- **C++ SDK**: `CacheGet()`/`DBQuery()`/`Get()` 返回类型从 `Buffer` 改为新增的 `GetResult` 结构体，包含 `return_code` 和 `data`，允许调用者区分"键不存在"（`ok() && empty()`）和"操作失败"（`!ok()`）
- **Java SDK**: 公开 API 参数名 `payload` → `params`，与 C API（`const char* params`）及 Go/Python/C++ SDK 保持一致；涉及 `handleRequest`/`notify`/`call`/`request`/`reply` 及回调接口 `RequestHandler`/`ResponseCallback`

#### 错误处理修复

- **Java SDK**: `cacheGet()` 和 `get()` 在 C API 返回非零时抛出 `D1Exception` 而非返回 null — 此前返回 null 无法区分"键不存在"和"操作失败"
- **C# SDK**: `CacheGet()` 和 `Get()` 在 C API 返回非零时抛出 `D1Exception` 而非返回 null — 同上

#### 示例同步

- **Python 示例**: `main.py` 中 `d1.call()` 返回值访问从 `result["payload"]` / `result["error"]` 改为 `result.payload` / `result.error` / `result.ok()`
- **C++ 示例**: `main.cpp` 中 `D1::CacheGet` / `D1::DBQuery` 返回类型从 `D1::Buffer` 改为 `D1::GetResult`
- **Java 示例**: `HelloD1.java` 中 handler 参数名 `payload` → `params`

#### 文档同步

- **全部 6 份 SDK 文档** 版本号从 v1.5.0 统一更新至 v1.7.0，日期更新至 2026-08-11，补充"适用于 D1 ≥ v1.7.0"标注
- **sdk-script.html** (SDK-02): `d1.publish()` → `d1.notify()` API 引用修正
- **sdk-service.html** (SDK-03): 反向调用表 `Publish` / `/{host_name}/Publish` → `Notify` / `/{host_name}/Notify`
- **sdk-protocol.html** (SDK-04): 版本号更新（`SetDefaultPublishHandler` / `client.Publish` 为 paho.mqtt 库 API，不变）
- **sdk-host.html** (SDK-05) / **sdk-integration.html** (SDK-06): 日期同步至 2026-08-11
- **config/connector.yaml**: 注释中 `Publish` → `Notify`
- **README.md**: 文档目录编号修正（SDK-01~06 与实际 HTML 标题对齐）

### 依赖
- 对应 D1 动态库版本: **≥ v1.7.0**

---

## [0.5.0] — 2026-08-10

### 变更

#### Critical 修复

- **Java SDK**: 修复库名大小写错误 (`"D1"` → `"d1"`)，修复 Linux/macOS 下无法加载 `libd1.so` 的问题
- **Java SDK**: 修复 `setOnCall` 回调 `Memory` double-free — 回调返回后 D1 和 JNA finalizer 均释放同一内存
- **C++ SDK**: 修复 `Call()` 错误路径三重 bug — `strdup`/`D1_Free` 分配器不匹配 + `length=-1` 导致错误信息丢失 + `ok()` 误判为成功
- **C++ SDK**: `Buffer` 新增 `owns` 标志，支持非 D1 分配的内存；`CallResult` 新增 `returnCode` 字段，`ok()` 检查返回码
- **C# SDK**: 修复 `Request()` 异步回调 GC 回收风险 — 使用 `ConcurrentDictionary` 存储回调引用，回调触发后自动移除
- **C# SDK**: 修复 `SetOnCall` 回调内存分配器不匹配 — `Marshal.AllocHGlobal` (Windows `LocalAlloc`) 与 D1 的 `free()` 不兼容，改用 `NativeMemory.Alloc` (映射到 `malloc`)
- **Python SDK**: 修复 `set_on_call` 回调 use-after-free — `ctypes.create_string_buffer` 在 GC 时释放内存导致悬垂指针，改用 libc `malloc` 直接分配
- **C# 示例**: 修复编译失败 — 缺少 `using Genius77.D1;` 指令
- **Go 集成示例**: 修复使用不存在的 `d1.New()` — 改为包级函数 (`d1.Init/d1.Start/d1.WaitStop/d1.OnCall/d1.OnInit` 等)
- **data_converter.sh**: 修复 heredoc 语法错误 — `EOF <<<"$CSV_DATA"` 不是有效的结束分隔符
- **Go CGO SDK**: 修复回调注册类型不匹配 — `(*C.d1_call_func_t)(unsafe.Pointer(C.goDefaultHandler))` 创建的是指向函数指针的指针而非函数指针本身，改用 `extern` 声明 + C helper 函数桥接

#### 示例改进

- **C/C++ 示例**: 修复 `D1_Call` 错误路径内存泄漏（`call_result` 在 `ret != 0` 时未释放）
- **C/C++ 示例**: 替换硬编码字符串长度为 `strlen()` 调用
- **README**: 修复 Go 快速开始代码片段 — 使用包级函数替代不存在的 `d1.New()`
- **README**: 修复 Go handler 签名 — `*d1.Context` 改为 `d1.Context`（Context 是接口类型，按值传递）；补充 `Init/Start` 错误处理
- **README**: 补充 0.5.0 版本表条目
- **Go CGO SDK**: 修正 API 数量注释 (17 → 22)
- **host 示例 README**: 移除所有 "TODO / 未经验证" 警告标记

#### 文档修正

- **README**: 修复版本表中 0.3.0 的依赖版本和变更描述
- **C# SDK**: 修正 API 数量注释 (17 → 22)

### 依赖
- 对应 D1 动态库版本: **≥ v1.7.0**

---

## [0.4.0] — 2026-07-24

### 变更
- **C 头文件同步**: `deps/d1.h` 和 `lang/c/d1.h` 与 D1 `api/d1.h` v1.7.0 完全对齐
- **Go SDK 修正**: `SetOnRequest`→`SetOnCall`，`Publish`→`Notify`，`Call` 签名增加 `kind string` 参数
- **Python SDK 修正**: `set_on_request`→`set_on_call`，`publish`→`notify`，`call` 签名增加 `kind` 参数
- **C++ 封装修正**: `D1_OnRequestFunc`→`D1_CallFunc`，`D1_SetOnRequest`→`D1_SetOnCall`，`D1_Publish`→`D1_Notify`
- **C# 封装修正**: `D1_SetOnRequest`→`D1_SetOnCall`，`D1_Publish`→`D1_Notify`
- **Java 封装修正**: `D1_SetOnRequest`→`D1_SetOnCall`，`D1_Publish`→`D1_Notify`
- **JS 脚本示例修正**: `d1.publish()`→`d1.notify()`（example.js、custom_handler.js）
- **README 修正**: Go 快速开始代码片段 `OnRequest`→`SetOnCall`
- **全语言示例验证**: C/C++/C#/Java/Python host 示例全部使用最新 API 名称

### 依赖
- 对应 D1 动态库版本: **≥ v1.7.0**

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

[0.6.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.6.0
[0.5.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.5.0
[0.4.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.4.0
[0.3.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.3.0
[0.2.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.2.0
[0.1.0]: https://github.com/genius77/d1-sdk/releases/tag/v0.1.0