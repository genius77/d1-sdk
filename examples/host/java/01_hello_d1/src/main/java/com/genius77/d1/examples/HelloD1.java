/*
 * D1 Java 示例 —— 01_hello_d1
 *
 * 本示例演示 D1 Java SDK 的完整用法：
 *   1. 获取版本号
 *   2. 初始化 D1 运行时
 *   3. 设置默认消息处理器（演示 API：Publish、Call、CacheSet/CacheGet、DBQuery）
 *   4. 启动 D1
 *   5. 阻塞等待退出（Ctrl+C）
 *
 * D1 动态库依赖: >= v1.5.0
 */

package com.genius77.d1.examples;

import com.genius77.d1.D1;
import com.genius77.d1.D1.RequestHandler;
import com.genius77.d1.D1.CallResult;

public class HelloD1 {

    /**
     * 默认消息处理器 — 演示所有 D1 API 的调用方式
     */
    static class DefaultHandler implements RequestHandler {
        @Override
        public Object[] handle(long taskId, String method, String payload) {
            System.out.println("[Handler] taskID=" + taskId +
                " | method=" + method + " | payload=" + (payload != null ? payload : "null"));

            /* ── 1. Publish — 发送单向消息（无回复） ── */
            D1.publish(taskId, "mqtt_client", "sensor.data",
                       "{\"temp\":25.5,\"unit\":\"celsius\"}");
            System.out.println("  D1_Publish -> OK");

            /* ── 2. CacheSet — 写入缓存 ── */
            D1.cacheSet(taskId, "user:42",
                        "{\"name\":\"Alice\",\"role\":\"admin\"}", 3600);
            System.out.println("  D1_CacheSet -> OK");

            /* ── 3. CacheGet — 读取缓存 ── */
            String cached = D1.cacheGet(taskId, "user:42");
            System.out.println("  D1_CacheGet -> " + (cached != null ? cached : "null"));

            /* ── 4. Call — 同步调用（阻塞等待结果） ── */
            CallResult callResult = D1.call(taskId, "script", "api_handler",
                    "get_user", "{\"id\":123}", 5);
            if (callResult.getPayload() != null) {
                System.out.println("  D1_Call -> " + callResult.getPayload());
            } else if (callResult.getError() != null) {
                System.out.println("  D1_Call error -> " + callResult.getError());
            }

            /* ── 5. DBQuery — 数据库查询 ── */
            String rows = D1.dbQuery(taskId, "SELECT * FROM users LIMIT 1");
            System.out.println("  D1_DBQuery -> " + (rows != null ? rows : "null"));

            /* ── 6. 返回响应 ── */
            return new Object[] {
                "{\"status\":\"ok\",\"msg\":\"hello from Java handler\"}",
                null,
                0
            };
        }
    }

    public static void main(String[] args) {
        System.out.println("===== D1 Java Hello World =====");

        /* 1. 获取版本号 */
        String version = D1.version();
        System.out.println("D1 Version: " + version);

        /* 2. 初始化 D1 运行时 */
        D1.init(null);
        System.out.println("D1::init OK");

        /* 3. 设置消息处理器 */
        D1.setOnRequest(new DefaultHandler());
        System.out.println("OnRequest handler registered");

        /* 4. 启动 D1 运行时 */
        D1.start();
        System.out.println("D1::start OK, running (press Ctrl+C to exit)");

        /* 5. 阻塞等待退出 */
        D1.waitStop();
        System.out.println("D1 stopped, exiting.");
    }
}