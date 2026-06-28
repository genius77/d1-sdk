/*
 * D1 SDK C++ 封装 | 对应 D1 动态库版本: ≥ v1.5.0
 *
 * 本文件对 D1 C API 进行了现代 C++ RAII 封装，提供 PascalCase 风格的方法调用。
 * 所有方法均为静态方法，通过 D1 类直接调用，无需实例化。
 *
 * 特性:
 *   - RAII 资源管理，自动释放 D1 分配的内存
 *   - 移动语义支持，安全传递所有权
 *   - 类型安全的 C++ 接口
 *   - 与原始 C API 零开销抽象
 *
 * 许可证: 参阅 D1 官方协议
 */

#ifndef D1_HPP
#define D1_HPP

#include <cstdint>
#include <cstring>
#include <functional>
#include <stdexcept>
#include <string>
#include <utility>

/* ──────────────────────────────────────────────────────────────────────────
 * C API 前向声明
 * ────────────────────────────────────────────────────────────────────────── */
extern "C" {
const char* D1_Version(void);
int         D1_Init(const char* config_path);
int         D1_Start(void);
int         D1_Stop(void);
int         D1_WaitStop(void);

typedef int (*D1_OnRequestFunc)(uint64_t task_id, const char* method,
                                const char* params, int params_len,
                                char** out_result, int* out_len,
                                char** out_error);
void D1_SetOnRequest(D1_OnRequestFunc handler);

typedef void (*D1_OnResponse)(uint64_t task_id, const char* params,
                              int params_len, const char* error);

int D1_Publish(uint64_t task_id, const char* target, const char* method,
               const char* params, int params_len);
int D1_Call(uint64_t task_id, const char* kind, const char* target,
            const char* method, const char* params, int params_len,
            int timeout_sec, char** out_result, int* out_len,
            char** out_error);
int D1_Request(uint64_t task_id, const char* target, const char* method,
               const char* params, int params_len, int timeout_sec,
               D1_OnResponse callback);
int D1_Reply(uint64_t task_id, const char* method, const char* params,
             int params_len);

int  D1_CacheGet(uint64_t task_id, const char* key, char** result,
                 int* result_len);
int  D1_CacheSet(uint64_t task_id, const char* key, const char* value,
                 int value_len, int ttl_seconds);
int  D1_CacheDelete(uint64_t task_id, const char* key);

int  D1_DBQuery(uint64_t task_id, const char* query, int query_len,
                char** result, int* result_len);
int  D1_DBExec(uint64_t task_id, const char* query, int query_len,
               int64_t* affected_rows);

int  D1_Set(uint64_t task_id, const char* key, const char* value,
            int value_len);
int  D1_Get(uint64_t task_id, const char* key, char** result, int* result_len);

void D1_Free(void* ptr);
}

/* ──────────────────────────────────────────────────────────────────────────
 * D1 类 —— 核心封装
 * ────────────────────────────────────────────────────────────────────────── */

/**
 * D1 —— D1 动态库的 C++ 静态封装类。
 *
 * 所有方法均为静态成员函数，不使用实例状态。
 * 内部通过 extern "C" 声明直接链接到 libd1.so。
 */
class D1 {
public:
    /* ======================================================================
     * ── 生命周期管理 ──
     * ====================================================================== */

    /**
     * Version - 获取 D1 动态库版本号。
     *
     * 返回: 版本号字符串（如 "1.1.0"）。
     */
    static std::string Version() {
        const char* v = D1_Version();
        return v ? std::string(v) : std::string();
    }

    /**
     * Init - 初始化 D1 运行时。
     *
     * 参数:
     *   config_path: 配置文件路径。传空字符串或 nullptr 表示使用默认配置。
     *
     * 抛出: std::runtime_error 当初始化失败时。
     */
    static void Init(const char* config_path = nullptr) {
        int rc = D1_Init(config_path);
        if (rc != 0) {
            throw std::runtime_error(std::string("D1_Init 失败，错误码: ")
                                     + std::to_string(rc));
        }
    }

    /**
     * Start - 启动 D1 运行时。
     *
     * 抛出: std::runtime_error 当启动失败时。
     */
    static void Start() {
        int rc = D1_Start();
        if (rc != 0) {
            throw std::runtime_error(std::string("D1_Start 失败，错误码: ")
                                     + std::to_string(rc));
        }
    }

    /**
     * Stop - 停止 D1 运行时。
     *
     * 抛出: std::runtime_error 当停止失败时。
     */
    static void Stop() {
        int rc = D1_Stop();
        if (rc != 0) {
            throw std::runtime_error(std::string("D1_Stop 失败，错误码: ")
                                     + std::to_string(rc));
        }
    }

    /** WaitStop - 阻塞等待退出信号，收到信号后自动调用 Stop()。
     *  推荐用法: Init() → Start() → WaitStop() → 进程退出 */
    static int WaitStop() { return D1_WaitStop(); }

    /**
     * SetOnRequest - 设置默认消息请求处理器。
     *
     * 参数:
     *   handler: 处理回调函数。使用 std::function 包装，支持 lambda。
     */
    static void SetOnRequest(
        std::function<int(uint64_t, const char*, const char*, int,
                          char**, int*, char**)> handler)
    {
        if (!handler) {
            D1_SetOnRequest(nullptr);
            return;
        }

        // 将 std::function 捕获到全局静态存储中，供 C 回调间接调用
        static std::function<int(uint64_t, const char*, const char*, int,
                                 char**, int*, char**)>
            s_handler;

        s_handler = std::move(handler);

        D1_SetOnRequest([](uint64_t task_id, const char* method,
                                const char* params, int params_len,
                                char** out_params, int* out_len,
                                char** out_error) -> int {
            if (s_handler) {
                return s_handler(task_id, method, params, params_len,
                                 out_params, out_len, out_error);
            }
            return -1;
        });
    }

    /** Free - 释放由 D1 API 分配的内存。 */
    static void Free(void* ptr) { D1_Free(ptr); }

    /* ======================================================================
     * ── RAII 辅助类型 ──
     * ====================================================================== */

    /**
     * Buffer —— RAII 内存管理辅助类。
     *
     * 自动管理 D1 API 返回的动态分配内存。
     * 支持移动语义，禁止拷贝。
     */
    struct Buffer {
        char* data   = nullptr;
        int   length = 0;

        Buffer() = default;

        Buffer(char* d, int len) : data(d), length(len) {}

        ~Buffer() { reset(); }

        // 禁止拷贝
        Buffer(const Buffer&)            = delete;
        Buffer& operator=(const Buffer&) = delete;

        // 移动构造
        Buffer(Buffer&& other) noexcept
            : data(other.data), length(other.length)
        {
            other.data   = nullptr;
            other.length = 0;
        }

        // 移动赋值
        Buffer& operator=(Buffer&& other) noexcept {
            if (this != &other) {
                reset();
                data         = other.data;
                length       = other.length;
                other.data   = nullptr;
                other.length = 0;
            }
            return *this;
        }

        /** 转换为 std::string（内容安全拷贝）。 */
        std::string toString() const {
            if (!data || length <= 0) return {};
            return std::string(data, static_cast<size_t>(length));
        }

        /** 是否有效（非空且有数据）。 */
        bool valid() const { return data != nullptr && length > 0; }

        /** 是否为空。 */
        bool empty() const { return !valid(); }

        /** 显式释放内部资源。 */
        void reset() {
            if (data) {
                D1_Free(data);
                data   = nullptr;
                length = 0;
            }
        }

        /** 释放所有权（调用者需手动管理内存）。 */
        char* release(int* out_len = nullptr) {
            if (out_len) *out_len = length;
            char* p  = data;
            data     = nullptr;
            length   = 0;
            return p;
        }
    };

    /**
     * CallResult —— D1::Call 的返回结果结构体。
     */
    struct CallResult {
        Buffer payload;   // 响应载荷
        Buffer error;     // 错误信息（成功时为空）

        /** 调用是否成功。 */
        bool ok() const { return error.empty(); }

        /** 获取载荷字符串。 */
        std::string payloadString() const { return payload.toString(); }

        /** 获取错误字符串。 */
        std::string errorString() const { return error.toString(); }
    };

    /**
     * DBExecResult —— D1::DBExec 的返回结果结构体。
     */
    struct DBExecResult {
        int     return_code;   // API 返回码，0 = 成功
        int64_t affected_rows; // 受影响行数

        /** 调用是否成功。 */
        bool ok() const { return return_code == 0; }
    };

    /* ======================================================================
     * ── 消息发布与调用 ──
     * ====================================================================== */

    /**
     * Publish - 发布（广播）一条消息。
     *
     * 参数:
     *   task_id:  任务 ID。
     *   target:   目标标识符。
     *   method:   消息名称。
     *   params:   消息载荷。
     *
     * 返回: 0 表示成功，非 0 表示失败。
     */
    static int Publish(uint64_t task_id, const std::string& target,
                       const std::string& method,
                       const std::string& params)
    {
        return D1_Publish(task_id, target.c_str(), method.c_str(),
                          params.data(), static_cast<int>(params.size()));
    }

    /**
     * Call - 同步调用远程服务。
     *
     * 参数:
     *   task_id:     任务 ID。
     *   kind:        处理器类型："default"/"conn"/"script"/"service"/"exec"，传空字符串使用默认类型。
     *   target:      目标标识符。
     *   method:      消息名称。
     *   params:      请求载荷。
     *   timeout_sec: 超时时间（秒），0 表示无超时。
     *
     * 返回: CallResult 结构体，包含响应载荷和错误信息。
     */
    static CallResult Call(uint64_t task_id, const std::string& kind,
                           const std::string& target,
                           const std::string& method,
                           const std::string& params, int timeout_sec)
    {
        char* out_result = nullptr;
        int   out_len     = 0;
        char* out_error   = nullptr;

        const char* k = kind.empty() ? nullptr : kind.c_str();

        int rc = D1_Call(task_id, k, target.c_str(), method.c_str(),
                         params.data(), static_cast<int>(params.size()),
                         timeout_sec, &out_result, &out_len, &out_error);

        if (rc != 0 && !out_error) {
            // API 返回错误但没有设置 out_error，合成一个
            CallResult result;
            result.error = Buffer(
                strdup(("D1_Call failed, error code: " + std::to_string(rc))
                           .c_str()),
                -1); // strdup memory must be freed by D1_Free; consider custom allocator for production
            return result;
        }

        CallResult result;
        if (out_result) {
            result.payload = Buffer(out_result, out_len);
        }
        if (out_error) {
            result.error = Buffer(out_error, static_cast<int>(strlen(out_error)));
        }
        return result;
    }

    /**
     * Request - 异步发送请求。
     *
     * 参数:
     *   task_id:     任务 ID。
     *   target:      目标标识符。
     *   method:      消息名称。
     *   params:      请求载荷。
     *   timeout_sec: 超时时间（秒）。
     *   callback:    响应回调（std::function<void(uint64_t, const char*, int, const char*)>）。
     *
     * 返回: 0 表示成功发送。
     */
    static int Request(uint64_t task_id, const std::string& target,
                       const std::string& method,
                       const std::string& params, int timeout_sec,
                       std::function<void(uint64_t, const char*, int,
                                          const char*)> callback)
    {
        if (!callback) {
            return D1_Request(task_id, target.c_str(), method.c_str(),
                              params.data(),
                              static_cast<int>(params.size()), timeout_sec,
                              nullptr);
        }

        // 将回调存储到静态位置
        static std::function<void(uint64_t, const char*, int, const char*)>
            s_callback;

        s_callback = std::move(callback);

        return D1_Request(
            task_id, target.c_str(), method.c_str(), params.data(),
            static_cast<int>(params.size()), timeout_sec,
            [](uint64_t tid, const char* p, int plen, const char* err) {
                if (s_callback) {
                    s_callback(tid, p, plen, err);
                }
            });
    }

    /**
     * Reply - 在处理请求的回调中发送响应。
     *
     * 参数:
     *   task_id:  任务 ID（与收到请求时一致）。
     *   method:   消息名称。
     *   params:   响应载荷。
     *
     * 返回: 0 表示成功。
     */
    static int Reply(uint64_t task_id, const std::string& method,
                     const std::string& params)
    {
        return D1_Reply(task_id, method.c_str(), params.data(),
                        static_cast<int>(params.size()));
    }

    /* ======================================================================
     * ── 缓存操作 ──
     * ====================================================================== */

    /**
     * CacheGet - 从 D1 内部缓存中读取键值。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   key:     缓存键名。
     *
     * 返回: Buffer 对象，包含缓存值和长度。
     */
    static Buffer CacheGet(uint64_t task_id, const std::string& key) {
        char* result      = nullptr;
        int   result_len  = 0;
        D1_CacheGet(task_id, key.c_str(), &result, &result_len);
        return Buffer(result, result_len);
    }

    /**
     * CacheSet - 向 D1 内部缓存写入键值对。
     *
     * 参数:
     *   task_id:     任务 ID。
     *   key:         缓存键名。
     *   value:       缓存值。
     *   ttl_seconds: 生存时间（秒），0 表示永不过期。
     *
     * 返回: 0 表示成功。
     */
    static int CacheSet(uint64_t task_id, const std::string& key,
                        const std::string& value, int ttl_seconds)
    {
        return D1_CacheSet(task_id, key.c_str(), value.data(),
                           static_cast<int>(value.size()), ttl_seconds);
    }

    /**
     * CacheDelete - 从 D1 内部缓存中删除指定键。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   key:     缓存键名。
     *
     * 返回: 0 表示成功。
     */
    static int CacheDelete(uint64_t task_id, const std::string& key) {
        return D1_CacheDelete(task_id, key.c_str());
    }

    /* ======================================================================
     * ── 数据库操作 ──
     * ====================================================================== */

    /**
     * DBQuery - 执行数据库查询，返回 JSON 格式结果集。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   query:   SQL 查询语句。
     *
     * 返回: Buffer 对象，包含 JSON 格式结果集。
     */
    static Buffer DBQuery(uint64_t task_id, const std::string& query) {
        char* result     = nullptr;
        int   result_len = 0;
        D1_DBQuery(task_id, query.c_str(), static_cast<int>(query.size()),
                   &result, &result_len);
        return Buffer(result, result_len);
    }

    /**
     * DBExec - 执行数据库变更语句。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   query:   SQL 变更语句（INSERT/UPDATE/DELETE 等）。
     *
     * 返回: DBExecResult，包含返回码和受影响行数。
     */
    static DBExecResult DBExec(uint64_t task_id, const std::string& query) {
        int64_t affected = 0;
        int rc = D1_DBExec(task_id, query.c_str(),
                           static_cast<int>(query.size()), &affected);
        return DBExecResult{rc, affected};
    }

    /* ======================================================================
     * ── 临时键值存储 ──
     * ====================================================================== */

    /**
     * Set - 设置临时键值对（内存存储）。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   key:     键名。
     *   value:   键值。
     *
     * 返回: 0 表示成功。
     */
    static int Set(uint64_t task_id, const std::string& key,
                   const std::string& value)
    {
        return D1_Set(task_id, key.c_str(), value.data(),
                      static_cast<int>(value.size()));
    }

    /**
     * Get - 获取临时键值对。
     *
     * 参数:
     *   task_id: 任务 ID。
     *   key:     键名。
     *
     * 返回: Buffer 对象，包含键值和长度。
     */
    static Buffer Get(uint64_t task_id, const std::string& key) {
        char* result     = nullptr;
        int   result_len = 0;
        D1_Get(task_id, key.c_str(), &result, &result_len);
        return Buffer(result, result_len);
    }

private:
    // 纯静态工具类，禁止实例化
    D1()  = delete;
    ~D1() = delete;
};

#endif /* D1_HPP */