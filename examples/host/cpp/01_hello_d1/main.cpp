/*
 * D1 C++ 示例 —— 01_hello_d1
 *
 * 本示例演示 D1 C++ 封装类的完整用法：
 *   1. 获取版本号
 *   2. 初始化 D1 运行时
 *   3. 设置消息处理器（演示 API：Notify、Call、CacheSet/CacheGet、DBQuery）
 *   4. 启动 D1
 *   5. 阻塞等待退出（Ctrl+C）
 *
 * 本示例在 handler 中使用 C++ 封装方法（D1::Notify、D1::Call 等），
 * 而非原始 C API，展示 RAII 封装的优势：无需手动管理内存。
 *
 * D1 动态库依赖: >= v1.7.0
 */

#include <iostream>
#include <string>
#include <cstring>
#include <cstdlib>

#include "d1.hpp"

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

    /* 3. 设置默认消息处理器 — 使用 C++ 封装方法 */
    D1::SetOnCall(
        [](uint64_t task_id, const char* method, const char* params,
           int params_len, char** out_result, int* out_len,
           char** out_error) -> int {
            std::cout << "[Handler] task_id=" << task_id
                      << ", method=" << (method ? method : "(null)")
                      << std::endl;

            /* ── 1. Notify — 发送单向消息（无回复） ── */
            std::string pub_data = R"({"temp":25.5,"unit":"celsius"})";
            int ret = D1::Notify(task_id, "mqtt_client", "sensor.data", pub_data);
            std::cout << "  D1::Notify -> " << ret << std::endl;

            /* ── 2. CacheSet — 写入缓存 ── */
            std::string cache_val = R"({"name":"Alice","role":"admin"})";
            ret = D1::CacheSet(task_id, "user:42", cache_val, 3600);
            std::cout << "  D1::CacheSet -> " << ret << std::endl;

            /* ── 3. CacheGet — 读取缓存（返回 GetResult，包含返回码和数据） ── */
            D1::GetResult cached = D1::CacheGet(task_id, "user:42");
            if (cached.ok() && !cached.empty()) {
                std::cout << "  D1::CacheGet -> " << cached.toString() << std::endl;
            }

            /* ── 4. Call — 同步调用（返回 CallResult，RAII 自动管理内存） ── */
            std::string call_params = R"({"id":123})";
            D1::CallResult call_result = D1::Call(
                task_id, "script", "api_handler", "get_user", call_params, 5);
            if (call_result.ok()) {
                std::cout << "  D1::Call -> " << call_result.payloadString() << std::endl;
            } else if (!call_result.error.empty()) {
                std::cout << "  D1::Call error -> " << call_result.errorString() << std::endl;
            }

            /* ── 5. DBQuery — 数据库查询（返回 GetResult，包含返回码和数据） ── */
            D1::GetResult db_result = D1::DBQuery(task_id, "SELECT * FROM users LIMIT 1");
            if (db_result.ok() && !db_result.empty()) {
                std::cout << "  D1::DBQuery -> " << db_result.toString() << std::endl;
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
