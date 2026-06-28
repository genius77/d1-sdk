/*
 * D1 C++ 示例 —— 01_hello_d1
 *
 * 本示例演示 D1 C++ 封装类的完整用法：
 *   1. 获取版本号
 *   2. 初始化 D1 运行时
 *   3. 设置消息处理器（演示 API：Publish、Call、CacheSet/CacheGet、DBQuery）
 *   4. 启动 D1
 *   5. 阻塞等待退出（Ctrl+C）
 *
 * D1 动态库依赖: >= v1.5.0
 */

#include <iostream>
#include <string>
#include <cstring>
#include <cstdlib>

#include "../../../lang/cpp/d1.hpp"

int main(int argc, char* argv[]) {
    (void)argc; (void)argv;

    std::cout << "===== D1 C++ Hello World =====" << std::endl;

    /* 1. 获取版本号 */
    std::string version = D1::Version();
    std::cout << "D1 Version: " << version << std::endl;

    /* 2. 初始化 D1 运行时 */
    try {
        D1::Init(nullptr);
        std::cout << "D1::Init OK" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "D1::Init failed: " << e.what() << std::endl;
        return EXIT_FAILURE;
    }

    /* 3. 设置默认消息处理器 — 演示所有 D1 API 的调用方式 */
    D1::SetOnRequest(
        [](uint64_t task_id, const char* method, const char* params,
           int params_len, char** out_result, int* out_len,
           char** out_error) -> int {
            std::cout << "[Handler] task_id=" << task_id
                      << ", method=" << (method ? method : "(null)")
                      << std::endl;

            /* ── 1. Publish — 发送单向消息（无回复） ── */
            std::string pub_data = R"({"temp":25.5,"unit":"celsius"})";
            int ret = D1_Publish(task_id, "mqtt_client", "sensor.data",
                                 pub_data.c_str(), (int)pub_data.size());
            std::cout << "  D1_Publish -> " << ret << std::endl;

            /* ── 2. CacheSet — 写入缓存 ── */
            std::string cache_val = R"({"name":"Alice","role":"admin"})";
            ret = D1_CacheSet(task_id, "user:42", cache_val.c_str(),
                              (int)cache_val.size(), 3600);
            std::cout << "  D1_CacheSet -> " << ret << std::endl;

            /* ── 3. CacheGet — 读取缓存 ── */
            char* cached = nullptr;
            int cached_len = 0;
            ret = D1_CacheGet(task_id, "user:42", &cached, &cached_len);
            if (ret == 0 && cached) {
                std::cout << "  D1_CacheGet -> "
                          << std::string(cached, cached_len) << std::endl;
                D1_Free(cached);
            }

            /* ── 4. Call — 同步调用（阻塞等待结果） ── */
            char* call_result = nullptr;
            int call_len = 0;
            char* call_error = nullptr;
            ret = D1_Call(task_id, "script", "api_handler", "get_user",
                          R"({"id":123})", 10, 5,
                          &call_result, &call_len, &call_error);
            if (ret == 0 && call_result) {
                std::cout << "  D1_Call -> "
                          << std::string(call_result, call_len) << std::endl;
                D1_Free(call_result);
            } else if (call_error) {
                std::cout << "  D1_Call error -> " << call_error << std::endl;
                D1_Free(call_error);
            }

            /* ── 5. DBQuery — 数据库查询 ── */
            char* db_result = nullptr;
            int db_len = 0;
            ret = D1_DBQuery(task_id, "SELECT * FROM users LIMIT 1", 30,
                             &db_result, &db_len);
            if (ret == 0 && db_result) {
                std::cout << "  D1_DBQuery -> "
                          << std::string(db_result, db_len) << std::endl;
                D1_Free(db_result);
            }

            /* ── 6. 返回响应 ── */
            std::string reply = R"({"status":"ok","msg":"hello from C++ handler"})";
            *out_result = static_cast<char*>(std::malloc(reply.size()));
            std::memcpy(*out_result, reply.data(), reply.size());
            *out_len = (int)reply.size();
            *out_error = nullptr;
            return 0;
        });
    std::cout << "Default handler registered" << std::endl;

    /* 4. 启动 D1 运行时 */
    try {
        D1::Start();
        std::cout << "D1::Start OK" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "D1::Start failed: " << e.what() << std::endl;
        return EXIT_FAILURE;
    }
    std::cout << "D1 running (press Ctrl+C to exit)" << std::endl;

    /* 5. 阻塞等待退出 */
    D1::WaitStop();
    std::cout << "D1 stopped, exiting." << std::endl;

    return EXIT_SUCCESS;
}