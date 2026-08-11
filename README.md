# D1 SDK — 多语言 SDK 封装与使用示例

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![D1 Version](https://img.shields.io/badge/D1-%E2%89%A5%20v1.7.0-blue)](https://github.com/genius77/D1/releases)

## 概述

本仓库是 **D1** 的官方 SDK 封装与多语言使用示例集合。D1 是一个高性能消息路由与任务调度引擎。

> **Go 项目可直接通过 `import "d1"` 集成 D1 运行时**，无需动态库。详见 [SDK-06 内部集成指南](doc/sdk-integration.html)。
>
> **其他语言项目**通过 C 动态库使用 D1。各语言 SDK 封装在 `lang/` 下，动态库下载方式见下方。

## 目录结构

```
d1-sdk/
├── README.md                     # 本文件
├── VERSION                       # 仓库版本号
├── CHANGELOG.md                  # 变更日志
├── doc/                          # SDK 使用文档
│   ├── sdk-exec.html             # SDK-01: Exec 扩展开发指南
│   ├── sdk-script.html           # SDK-02: Script 扩展开发指南
│   ├── sdk-service.html          # SDK-03: Service 扩展开发指南
│   ├── sdk-protocol.html         # SDK-04: Protocol 开发指南
│   ├── sdk-host.html             # SDK-05: 外部宿主 API 指南
│   └── sdk-integration.html      # SDK-06: 内部集成指南
├── download/                     # D1 动态库下载脚本
│   ├── download_d1.sh            # Unix/Linux/macOS
│   └── download_d1.ps1           # Windows PowerShell
├── config/                       # 示例配置文件
│   ├── d1.yaml
│   ├── connector.yaml
│   ├── router.yaml
│   ├── exec.yaml
│   ├── script.yaml
│   └── service.yaml
├── lang/                         # 各语言 SDK 封装（封装 D1 全部 C API）
│   ├── c/d1.h                    # C 头文件
│   ├── cpp/d1.hpp                # C++ RAII 封装
│   ├── go/d1.go                  # Go CGO 封装
│   ├── python/d1.py              # Python ctypes 封装
│   ├── csharp/D1.cs              # C# P/Invoke 封装
│   └── java/D1.java              # Java JNA 封装
└── examples/                     # 可运行示例
    ├── host/                     # 宿主程序示例（通过 C 动态库）
    │   ├── c/01_hello_d1/        # C 语言入门
    │   ├── cpp/01_hello_d1/      # C++ 入门
    │   ├── csharp/01_hello_d1/   # C# 入门
    │   ├── java/01_hello_d1/     # Java 入门
    │   └── python/01_hello_d1/   # Python 入门
    ├── integration/              # Go 直接集成示例（import "d1"）
    │   └── 01_hello_d1/          # Go 入门
    ├── exec/                     # Exec 扩展示例
    │   ├── default_handler.sh
    │   ├── status_check.sh
    │   └── data_converter.sh
    ├── protocol/                 # Protocol 扩展示例
    │   ├── http_driver/
    │   └── mqtt_driver/
    ├── script/                   # Script 扩展示例
    │   ├── example.js
    │   ├── custom_handler.js
    │   └── data_transform.js
    └── service/                  # Service 扩展示例
        ├── cache_service/
        └── rule_engine/
```

## 快速开始

### 1. Clone 仓库

```bash
git clone https://github.com/genius77/d1-sdk.git
cd d1-sdk
```

### 2. Go 直接集成（推荐 Go 项目）

无需动态库，直接 `import "d1"`：

```go
import "d1"

d1.OnCall(func(ctx d1.Context, req *d1.Request) (*d1.Response, error) {
    return &d1.Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  json.RawMessage(`{"ok":true}`),
    }, nil
})
if err := d1.Init("./config"); err != nil {
    panic(err)
}
if err := d1.Start(); err != nil {
    panic(err)
}
d1.WaitStop()
```

详见 [SDK-06 内部集成指南](doc/sdk-integration.html) 和 [integration 示例](examples/integration/01_hello_d1/)。

### 3. 宿主程序方式（C/C++/Python/C#/Java）

下载 D1 动态库后通过各语言 SDK 封装使用：

```bash
# Unix/Linux/macOS
./download/download_d1.sh

# Windows PowerShell
.\download\download_d1.ps1
```

下载后动态库和头文件位于项目根目录的 `deps/` 下。

| 语言 | SDK 封装 | 入门示例 | 构建方式 |
|------|----------|----------|----------|
| C | [`lang/c/d1.h`](lang/c/d1.h) | [`examples/host/c/01_hello_d1/`](examples/host/c/01_hello_d1/) | `cmake -B build && cmake --build build` |
| C++ | [`lang/cpp/d1.hpp`](lang/cpp/d1.hpp) | [`examples/host/cpp/01_hello_d1/`](examples/host/cpp/01_hello_d1/) | `cmake -B build && cmake --build build` |
| Go (CGO) | [`lang/go/d1.go`](lang/go/d1.go) | — | `go build` |
| Python | [`lang/python/d1.py`](lang/python/d1.py) | [`examples/host/python/01_hello_d1/`](examples/host/python/01_hello_d1/) | `python main.py` |
| C# | [`lang/csharp/D1.cs`](lang/csharp/D1.cs) | [`examples/host/csharp/01_hello_d1/`](examples/host/csharp/01_hello_d1/) | `dotnet run` |
| Java | [`lang/java/D1.java`](lang/java/D1.java) | [`examples/host/java/01_hello_d1/`](examples/host/java/01_hello_d1/) | `mvn compile exec:java` |

## D1 动态库版本依赖

| 本仓库版本 | 最低 D1 动态库版本 | 主要变更 |
|-----------|--------------------|----------|
| **0.6.0** | **≥ v1.7.0** | 跨语言 API 一致性：Python `call()` 返回 `CallResult` namedtuple；C++ `CacheGet`/`DBQuery`/`Get` 返回 `GetResult`（含返回码）；Java 参数名 `payload`→`params`；Java/C# `cacheGet`/`get` 失败时抛出异常 |
| 0.5.0 | ≥ v1.7.0 | 修复库名大小写（C#/Java）、内存分配器不匹配（C++/C#/Python）、Python 示例作用域、Go CGO 回调注册类型不匹配；完善示例和文档 |
| 0.4.0 | ≥ v1.7.0 | 全部语言绑定与 C 头文件同步至 `d1.h` v1.7.0；JS 脚本示例 `d1.publish`→`d1.notify`；README Go 快速开始修正为 `SetOnCall` |
| 0.3.0 | ≥ v1.7.0 | `D1_SetOnRequest`→`D1_SetOnCall`；`D1_Publish`→`D1_Notify`；新增生命周期回调；`kind` 改为字符串 |
| 0.2.0 | ≥ v1.2.0 | 移除 `Wait()`，新增 `WaitStop()`；7 平台编译支持 |
| 0.1.0 | ≥ v1.1.0 | 初始版本 |

> 每个 `lang/<lang>/d1.{ext}` 文件头部均标注了对应的 D1 动态库最低版本要求。
>
> **如何下载 D1 动态库**：访问 [D1 Releases](https://github.com/genius77/D1/releases)，选择与你使用的 d1-sdk 版本匹配的 D1 版本，下载对应平台的 `.so` / `.dylib` / `.dll` 文件放入 `deps/` 目录。

## 贡献

欢迎提交 Issue 和 Pull Request！

- 新增某语言宿主示例 → 在 `examples/host/<lang>/` 下创建新目录
- Go 直接集成示例 → 在 `examples/integration/` 下创建新目录
- 扩展示例 → 在 `examples/exec/`、`examples/script/`、`examples/service/` 下创建
- SDK 封装改进 → 修改 `lang/<lang>/` 下对应文件
- 文档优化 → 编辑对应 README.md 或 `doc/` 下对应文件

## License

本仓库示例代码和 SDK 封装使用 MIT License。D1 核心动态库有独立的商业许可。