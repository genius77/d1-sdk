/*
 * D1 C# 示例 —— 01_hello_d1
 *
 * 本示例演示 D1 C# SDK 的完整用法：
 *   1. 获取版本号
 *   2. 初始化 D1 运行时
 *   3. 设置默认消息处理器（演示 API：Publish、Call、CacheSet/CacheGet、DBQuery）
 *   4. 启动 D1
 *   5. 阻塞等待退出（Ctrl+C）
 *
 * D1 动态库依赖: >= v1.5.0
 */

using System;
using System.Text.Json;

class Program
{
    /*
     * 默认消息处理器 — 演示所有 D1 API 的调用方式
     */
    static int DefaultHandler(ulong taskId, string method, string payload,
                              out string outResult, out string outError)
    {
        Console.WriteLine($"[Handler] taskID={taskId} | method={method} | payload={payload ?? "null"}");

        /* ── 1. Publish — 发送单向消息（无回复） ── */
        var pubData = JsonSerializer.Serialize(new { temp = 25.5, unit = "celsius" });
        D1.Publish(taskId, "mqtt_client", "sensor.data", pubData);
        Console.WriteLine("  D1_Publish -> OK");

        /* ── 2. CacheSet — 写入缓存 ── */
        var cacheVal = JsonSerializer.Serialize(new { name = "Alice", role = "admin" });
        D1.CacheSet(taskId, "user:42", cacheVal, 3600);
        Console.WriteLine("  D1_CacheSet -> OK");

        /* ── 3. CacheGet — 读取缓存 ── */
        var cached = D1.CacheGet(taskId, "user:42");
        Console.WriteLine($"  D1_CacheGet -> {cached ?? "null"}");

        /* ── 4. Call — 同步调用（阻塞等待结果） ── */
        var callParams = JsonSerializer.Serialize(new { id = 123 });
        var callResult = D1.Call(taskId, "script", "api_handler", "get_user", callParams, 5);
        if (callResult.Payload != null)
            Console.WriteLine($"  D1_Call -> {callResult.Payload}");
        else if (callResult.Error != null)
            Console.WriteLine($"  D1_Call error -> {callResult.Error}");

        /* ── 5. DBQuery — 数据库查询 ── */
        var rows = D1.DBQuery(taskId, "SELECT * FROM users LIMIT 1");
        Console.WriteLine($"  D1_DBQuery -> {rows ?? "null"}");

        /* ── 6. 返回响应 ── */
        outResult = JsonSerializer.Serialize(new { status = "ok", msg = "hello from C# handler" });
        outError = null;
        return 0;
    }

    static void Main(string[] args)
    {
        Console.WriteLine("===== D1 C# Hello World =====");

        /* 1. 获取版本号 */
        var version = D1.Version();
        Console.WriteLine($"D1 Version: {version}");

        /* 2. 初始化 D1 运行时 */
        D1.Init(null);
        Console.WriteLine("D1::Init OK");

        /* 3. 设置默认消息处理器 */
        D1.SetOnRequest(DefaultHandler);
        Console.WriteLine("Default handler registered");

        /* 4. 启动 D1 运行时 */
        D1.Start();
        Console.WriteLine("D1::Start OK, running (press Ctrl+C to exit)");

        /* 5. 阻塞等待退出 */
        D1.WaitStop();
        Console.WriteLine("D1 stopped, exiting.");
    }
}