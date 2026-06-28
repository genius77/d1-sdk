/*
 * D1 C 语言示例 —— 01_hello_d1
 *
 * 本示例演示 D1 C API 的完整用法：
 *   1. 获取版本号
 *   2. 初始化 D1 运行时
 *   3. 设置消息处理器（演示 API：Publish、Call、CacheSet/CacheGet、DBQuery）
 *   4. 启动 D1
 *   5. 阻塞等待退出（Ctrl+C）
 *
 * 编译时需要链接 libd1.so，详见 CMakeLists.txt。
 * D1 动态库依赖: >= v1.5.0
 */

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "../../../lang/c/d1.h"

/* ─────────────────────────────────────────────────────────
 * 消息处理器 — 演示所有 D1 API 的调用方式
 * ───────────────────────────────────────────────────────── */

static int on_request(uint64_t task_id,
                      const char* method,
                      const char* params,
                      int params_len,
                      char** out_result,
                      int* out_len,
                      char** out_error) {
    printf("[处理器] task_id=%lu, method=%s\n", (unsigned long)task_id, method);

    /* ── 1. Publish — 发送单向消息（无回复） ── */
    const char* pub_data = "{\"temp\":25.5,\"unit\":\"celsius\"}";
    int ret = D1_Publish(task_id, "mqtt_client", "sensor.data",
                         pub_data, (int)strlen(pub_data));
    printf("  D1_Publish → %d\n", ret);

    /* ── 2. CacheSet — 写入缓存 ── */
    const char* cache_val = "{\"name\":\"Alice\",\"role\":\"admin\"}";
    ret = D1_CacheSet(task_id, "user:42", cache_val, (int)strlen(cache_val), 3600);
    printf("  D1_CacheSet → %d\n", ret);

    /* ── 3. CacheGet — 读取缓存 ── */
    char* cached = NULL;
    int cached_len = 0;
    ret = D1_CacheGet(task_id, "user:42", &cached, &cached_len);
    if (ret == 0 && cached) {
        printf("  D1_CacheGet → %.*s\n", cached_len, cached);
        D1_Free(cached);
    }

    /* ── 4. Call — 同步调用（阻塞等待结果） ── */
    char* call_result = NULL;
    int call_len = 0;
    char* call_error = NULL;
    ret = D1_Call(task_id, "script", "api_handler", "get_user",
                  "{\"id\":123}", 10, 5,
                  &call_result, &call_len, &call_error);
    if (ret == 0 && call_result) {
        printf("  D1_Call → %.*s\n", call_len, call_result);
        D1_Free(call_result);
    } else if (call_error) {
        printf("  D1_Call error → %s\n", call_error);
        D1_Free(call_error);
    }

    /* ── 5. DBQuery — 数据库查询 ── */
    char* db_result = NULL;
    int db_len = 0;
    ret = D1_DBQuery(task_id, "SELECT * FROM users LIMIT 1", 30,
                     &db_result, &db_len);
    if (ret == 0 && db_result) {
        printf("  D1_DBQuery → %.*s\n", db_len, db_result);
        D1_Free(db_result);
    }

    /* ── 6. 返回响应 ── */
    const char* reply = "{\"status\":\"ok\",\"msg\":\"hello from C handler\"}";
    *out_result = (char*)malloc((size_t)strlen(reply));
    memcpy(*out_result, reply, strlen(reply));
    *out_len = (int)strlen(reply);
    *out_error = NULL;
    return 0;
}

/* ─────────────────────────────────────────────────────────
 * 主函数
 * ───────────────────────────────────────────────────────── */

int main(int argc, char* argv[]) {
    (void)argc; (void)argv;

    printf("===== D1 C Hello World =====\n");

    /* 1. 获取版本号 */
    const char* version = D1_Version();
    printf("D1 Version: %s\n", version ? version : "(null)");

    /* 2. 初始化 D1 运行时 */
    if (D1_Init(NULL) != 0) {
        fprintf(stderr, "D1_Init failed\n");
        return EXIT_FAILURE;
    }
    printf("D1_Init OK\n");

    /* 3. 设置默认消息处理器 */
    D1_SetOnRequest(on_request);
    printf("Default handler registered\n");

    /* 4. 启动 D1 运行时 */
    if (D1_Start() != 0) {
        fprintf(stderr, "D1_Start failed\n");
        return EXIT_FAILURE;
    }
    printf("D1 running (press Ctrl+C to exit)\n");

    /* 5. 阻塞等待退出 */
    D1_WaitStop();
    printf("D1 stopped, exiting.\n");
    return EXIT_SUCCESS;
}