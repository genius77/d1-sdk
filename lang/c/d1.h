#ifndef D1_H
#define D1_H

/*
 * D1 SDK C 头文件 | 对应 D1 动态库版本: ≥ v1.5.0
 */

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// ============================================================
// 生命周期函数
// ============================================================

// 获取版本号
const char* D1_Version(void);

// 初始化底座
int D1_Init(const char* config_path);

// 启动底座
int D1_Start(void);

// 停止底座
int D1_Stop(void);

// 等待底座退出并自动停止（收到 SIGINT/SIGTERM 后内部调用 D1_Stop()）
// 推荐用法：Init() → Start() → WaitStop() → 进程退出
// 返回值: 0=成功, -1=停止失败
int D1_WaitStop(void);

// ============================================================
// D1_OnRequestFunc — 宿主程序默认处理器（kind="default" 回调）
//
// 参数:
//   task_id      任务 ID（uint64）
//   method       消息名称
//   params       消息载荷（JSON）
//   params_len   载荷长度
//   out_result   输出：JSON-RPC 结果（由 D1 管理内存，宿主无需释放）
//   out_len      输出：结果长度
//   out_error    输出：错误信息指针（由 D1 管理内存）
//
// 返回值: 0=成功, -1=失败
// ============================================================
typedef int (*D1_OnRequestFunc)(uint64_t task_id, const char* method,
                              const char* params, int params_len,
                              char** out_result, int* out_len,
                              char** out_error);

// 注册宿主程序默认处理器
void D1_SetOnRequest(D1_OnRequestFunc handler);

// 注册宿主程序异步响应回调（Request 使用）
typedef void (*D1_OnResponse)(uint64_t task_id, const char* params,
                              int params_len, const char* error);

// ============================================================
// 核心 API（所有操作基于 task_id）
// ============================================================

// Publish 发送单向消息（无回复），仅支持 kind="conn"
// params/params_len: 消息数据
// 返回值: 0=成功, 负数=错误码
int D1_Publish(uint64_t task_id, const char* target,
               const char* method, const char* params, int params_len);

// Call 同步调用-响应（阻塞等待结果）
// timeout_sec: 超时秒数，0=使用默认
// out_result/out_len: 输出 JSON-RPC 结果（由 D1 管理内存）
// out_error: 输出错误信息（由 D1 管理内存）
// 返回值: 0=成功, -1=失败
int D1_Call(uint64_t task_id, const char* kind, const char* target,
            const char* method, const char* params, int params_len,
            int timeout_sec,
            char** out_result, int* out_len, char** out_error);

// Request 异步发送消息（通过 callback 接收回复），仅支持 kind="conn"
// callback: 收到响应或超时时调用
// 返回值: 0=成功, -1=失败
int D1_Request(uint64_t task_id, const char* target,
               const char* method, const char* params, int params_len,
               int timeout_sec, D1_OnResponse callback);

// Reply 回复请求（向 OriginMsg 发送响应）
int D1_Reply(uint64_t task_id, const char* method,
             const char* params, int params_len);

// ============================================================
// 缓存 API
// ============================================================

// CacheGet 获取缓存值（result 由 D1 管理内存）
int D1_CacheGet(uint64_t task_id, const char* key,
                char** result, int* result_len);

// CacheSet 设置缓存值
int D1_CacheSet(uint64_t task_id, const char* key,
                const char* value, int value_len, int ttl_seconds);

// CacheDelete 删除缓存
int D1_CacheDelete(uint64_t task_id, const char* key);

// ============================================================
// 数据库 API
// ============================================================

// DBQuery 查询数据库（result 为 JSON 数组，由 D1 管理内存）
int D1_DBQuery(uint64_t task_id, const char* query, int query_len,
               char** result, int* result_len);

// DBExec 执行数据库写入
int D1_DBExec(uint64_t task_id, const char* query, int query_len,
              int64_t* affected_rows);

// ============================================================
// 临时数据 API（Task 生命周期内有效）
// ============================================================

// Set 设置临时数据
int D1_Set(uint64_t task_id, const char* key,
           const char* value, int value_len);

// Get 获取临时数据（result 为 JSON，由 D1 管理内存）
int D1_Get(uint64_t task_id, const char* key,
           char** result, int* result_len);

// ============================================================
// 内存管理
// ============================================================

// Free 释放由 D1 分配的内存（out_result/result/error）
void D1_Free(void* ptr);

#ifdef __cplusplus
}
#endif

#endif // D1_H