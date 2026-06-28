// D1 SDK Java 封装 | 对应 D1 动态库版本: ≥ v1.5.0
// 基于 JNA (Java Native Access) 实现跨平台动态库调用。
//
// 用法:
//   1. 在 pom.xml 中添加 JNA 依赖: net.java.dev.jna:jna:5.14.0
//   2. 将动态库放入 deps/ 目录:
//        Windows: deps/D1.dll
//        Linux:   deps/libD1.so
//        macOS:   deps/libD1.dylib
//   3. 代码中 import com.genius77.d1.D1 即可使用。
//
// 内存管理说明:
//   - version() 返回的字符串为静态常量，不需要释放。
//   - call() / cacheGet() / dbQuery() / get() 返回的字符串由 D1 分配，
//     本 SDK 封装层自动调用 D1_Free 释放，调用者无需关心。
//   - setOnRequest() 回调中通过 PointerByReference 返回的字符串由 D1 负责释放。

package com.genius77.d1;

import java.io.File;
import java.nio.charset.StandardCharsets;

import com.sun.jna.Callback;
import com.sun.jna.Library;
import com.sun.jna.Memory;
import com.sun.jna.Native;
import com.sun.jna.NativeLibrary;
import com.sun.jna.Pointer;
import com.sun.jna.ptr.IntByReference;
import com.sun.jna.ptr.LongByReference;
import com.sun.jna.ptr.PointerByReference;

/**
 * D1 C 动态库的 JNA 映射接口。
 * 包含全部 17 个 C API 函数的声明，供内部使用。
 */
interface D1Library extends Library {

    /**
     * 获取 D1 版本号（静态字符串，由 D1 内部持有）。
     */
    String D1_Version();

    /**
     * 初始化 D1 运行时。
     * @param configPath 配置文件路径，null 表示默认配置。
     * @return 0 成功，非零失败。
     */
    int D1_Init(String configPath);

    /**
     * 启动 D1 运行时。
     * @return 0 成功，非零失败。
     */
    int D1_Start();

    /**
     * 停止 D1 运行时。
     * @return 0 成功，非零失败。
     */
    int D1_Stop();

    /** 阻塞等待退出信号后自动停止。推荐用法: init() → start() → waitStop() */
    int D1_WaitStop();

    /**
     * 设置默认请求处理器。
     * @param handler 回调函数。
     */
    void D1_SetOnRequest(D1.NativeOnRequestFunc handler);

    /**
     * 发布消息到目标（不等待响应）。
     * @return 0 成功，非零失败。
     */
    int D1_Publish(long taskId, String target, String method, String payload, int payloadLen);

    /**
     * 同步调用目标服务。
     * @param outResult 输出: 响应负载指针。
     * @param outLen 输出: 响应负载长度。
     * @param outError 输出: 错误信息指针。
     * @return 0 成功，非零失败。
     */
    int D1_Call(long taskId, String kind, String target, String method, String payload,
                int payloadLen, int timeoutSec,
                PointerByReference outResult, IntByReference outLen,
                PointerByReference outError);

    /**
     * 异步请求目标服务。
     * @param callback 响应回调函数。
     * @return 0 成功，非零失败。
     */
    int D1_Request(long taskId, String target, String method, String payload,
                   int payloadLen, int timeoutSec, D1.NativeOnResponse callback);

    /**
     * 回复当前请求。
     * @return 0 成功，非零失败。
     */
    int D1_Reply(long taskId, String method, String payload, int payloadLen);

    /**
     * 从缓存中获取键值。
     * @param result 输出: 结果指针。
     * @param resultLen 输出: 结果长度。
     * @return 0 成功，非零失败。
     */
    int D1_CacheGet(long taskId, String key, PointerByReference result, IntByReference resultLen);

    /**
     * 向缓存中设置键值对。
     * @return 0 成功，非零失败。
     */
    int D1_CacheSet(long taskId, String key, String value, int valueLen, int ttlSeconds);

    /**
     * 删除缓存中的键。
     * @return 0 成功，非零失败。
     */
    int D1_CacheDelete(long taskId, String key);

    /**
     * 执行数据库查询。
     * @param result 输出: 查询结果 JSON 指针。
     * @param resultLen 输出: 结果长度。
     * @return 0 成功，非零失败。
     */
    int D1_DBQuery(long taskId, String query, int queryLen,
                   PointerByReference result, IntByReference resultLen);

    /**
     * 执行数据库写操作。
     * @param affectedRows 输出: 受影响的行数。
     * @return 0 成功，非零失败。
     */
    int D1_DBExec(long taskId, String query, int queryLen, LongByReference affectedRows);

    /**
     * 向键值存储中设置值。
     * @return 0 成功，非零失败。
     */
    int D1_Set(long taskId, String key, String value, int valueLen);

    /**
     * 从键值存储中获取值。
     * @param result 输出: 结果指针。
     * @param resultLen 输出: 结果长度。
     * @return 0 成功，非零失败。
     */
    int D1_Get(long taskId, String key, PointerByReference result, IntByReference resultLen);

    /**
     * 释放由 D1 C API 分配的内存。
     * @param ptr 要释放的指针。
     */
    void D1_Free(Pointer ptr);
}

/**
 * D1 动态库 JNA SDK 封装类。
 * 所有方法均为静态方法，线程安全。
 * 首次使用时会自动加载跨平台动态库。
 */
public final class D1 {

    // ========================================================================
    //  内部类型: 异常、回调接口、调用结果
    // ========================================================================

    /**
     * D1 运行时异常。当 D1 C API 返回非零错误码时抛出。
     */
    public static class D1Exception extends RuntimeException {
        private static final long serialVersionUID = 1L;

        /** 原始错误码。 */
        private final int errorCode;

        /** 触发异常的函数名。 */
        private final String functionName;

        D1Exception(int errorCode, String functionName) {
            super("D1 " + functionName + " failed, error code: " + errorCode);
            this.errorCode = errorCode;
            this.functionName = functionName;
        }

        /** 获取 C API 返回的原始错误码。 */
        public int getErrorCode() { return errorCode; }

        /** 获取触发异常的函数名称。 */
        public String getFunctionName() { return functionName; }
    }

    /**
     * D1.call() 方法调用返回结果。
     */
    public static class CallResult {
        private final int returnCode;
        private final String payload;
        private final String error;

        CallResult(int returnCode, String payload, String error) {
            this.returnCode = returnCode;
            this.payload = payload;
            this.error = error;
        }

        /** C API 原始返回码，0 表示成功。 */
        public int getReturnCode() { return returnCode; }

        /** 返回的负载字符串，可能为 null。 */
        public String getPayload() { return payload; }

        /** 返回的错误信息，可能为 null。 */
        public String getError() { return error; }

        /** 调用是否成功（返回码为 0 且 Error 为 null）。 */
        public boolean isSuccess() { return returnCode == 0 && error == null; }

        @Override
        public String toString() {
            return "D1CallResult(Code=" + returnCode +
                   ", Payload=" + (payload != null ? payload : "(null)") +
                   ", Error=" + (error != null ? error : "(null)") + ")";
        }
    }

    // ========================================================================
    //  JNA 回调接口 — 精确匹配 C ABI
    // ========================================================================

    /**
     * JNA 原生默认请求处理回调。适配 C 函数签名:
     * <pre>int (*)(uint64_t task_id, const char* method, const char* payload, int payload_len,
     *          char** out_result, int* out_len, char** out_error)</pre>
     */
    public interface NativeOnRequestFunc extends Callback {
        int invoke(long taskId, String method, String payload, int payloadLen,
                   PointerByReference outResult, IntByReference outLen,
                   PointerByReference outError);
    }

    /**
     * JNA 原生异步响应回调。适配 C 函数签名:
     * <pre>void (*)(uint64_t task_id, const char* payload, int payload_len, const char* error)</pre>
     */
    public interface NativeOnResponse extends Callback {
        void invoke(long taskId, String payload, int payloadLen, String error);
    }

    /**
     * 用户友好的请求处理函数式接口。
     */
    @FunctionalInterface
    public interface RequestHandler {
        /**
         * 处理请求。
         * @param taskId   任务 ID。
         * @param method  消息名称。
         * @param payload  请求负载。
         * @return 返回一个 Object[] 数组: [0]=响应负载(String), [1]=错误信息(String 或 null), [2]=返回码(Integer)。
         */
        Object[] handle(long taskId, String method, String payload);
    }

    /**
     * 用户友好的响应回调函数式接口。
     */
    @FunctionalInterface
    public interface ResponseCallback {
        /**
         * 接收响应。
         * @param taskId  任务 ID。
         * @param payload 响应负载，可能为 null。
         * @param error   错误信息，可能为 null。
         */
        void onResponse(long taskId, String payload, String error);
    }

    // ========================================================================
    //  动态库加载
    // ========================================================================

    private static final D1Library lib;
    private static RequestHandler requestHandlerRef;
    private static NativeOnRequestFunc nativeRequestHandlerRef;

    static {
        loadLibrary();
    }

    private static void loadLibrary() {
        // 自动搜索 deps/ 目录并注册 JNA 搜索路径
        String[] searchPaths = {
            "./deps",
            "../deps",
            "../../deps",
            "../../../deps",
            "./lib",
            "../lib"
        };

        boolean found = false;
        for (String path : searchPaths) {
            File dir = new File(path);
            if (dir.exists() && dir.isDirectory()) {
                NativeLibrary.addSearchPath("D1", dir.getAbsolutePath());
                found = true;
                break;
            }
        }

        try {
            lib = Native.load("D1", D1Library.class);
        } catch (UnsatisfiedLinkError e) {
            String os = System.getProperty("os.name", "").toLowerCase();
            String libName;
            if (os.contains("win")) {
                libName = "D1.dll";
            } else if (os.contains("mac")) {
                libName = "libD1.dylib";
            } else {
                libName = "libD1.so";
            }

            StringBuilder msg = new StringBuilder();
            msg.append("Failed to load D1 native library. Please check:\n");
            msg.append("  1. Is the library file placed in deps/ directory?\n");
            msg.append("  2. Is the filename correct: ").append(libName).append("\n");
            msg.append("  3. Set system property: -Djna.library.path=<directory>\n");
            if (found) {
                msg.append("  deps/ found but loading failed. Verify file exists and arch matches\n");
            } else {
                msg.append("  deps/ not found. Verify working directory is correct\n");
            }
            throw new RuntimeException(msg.toString(), e);
        }
    }

    /** 私有构造器，禁止实例化。 */
    private D1() {}

    // ========================================================================
    //  内部辅助方法
    // ========================================================================

    /** 计算 UTF-8 字符串的字节长度（不包括 null 终止符）。 */
    private static int utf8ByteLength(String s) {
        if (s == null) return 0;
        return s.getBytes(StandardCharsets.UTF_8).length;
    }

    /**
     * 将 JNA Pointer 转为 Java String，然后调用 D1_Free 释放。
     * 用于 call() / cacheGet() / dbQuery() / get() 等返回的需要释放的字符串。
     */
    private static String ptrToStringAndFree(Pointer ptr, int len) {
        if (ptr == null || Pointer.nativeValue(ptr) == 0) {
            return null;
        }

        String result;
        if (len > 0) {
            byte[] bytes = ptr.getByteArray(0, len);
            result = new String(bytes, StandardCharsets.UTF_8);
        } else {
            // 回退方案: 按照 null-terminated 字符串读取
            result = ptr.getString(0);
        }

        lib.D1_Free(ptr);
        return result;
    }

    /** 如果返回码非零则抛出 D1Exception。 */
    private static void throwIfError(int code, String functionName) {
        if (code != 0) {
            throw new D1Exception(code, functionName);
        }
    }

    // ========================================================================
    //  公开 API
    // ========================================================================

    /**
     * 获取 D1 动态库的版本号。
     * @return 版本号字符串，例如 "1.1.0"。
     */
    public static String version() {
        return lib.D1_Version();
    }

    /**
     * 初始化 D1 运行时。必须在 start() 之前调用。
     * @param configPath 配置文件路径，传入 null 使用默认配置。
     * @throws D1Exception 初始化失败时抛出。
     */
    public static void init(String configPath) {
        int ret = lib.D1_Init(configPath);
        throwIfError(ret, "Init");
    }

    /**
     * 启动 D1 运行时。init() 成功后调用。
     * @throws D1Exception 启动失败时抛出。
     */
    public static void start() {
        int ret = lib.D1_Start();
        throwIfError(ret, "Start");
    }

    /**
     * 停止 D1 运行时。阻塞直到所有任务完成。
     * @throws D1Exception 停止失败时抛出。
     */
    public static void stop() {
        int ret = lib.D1_Stop();
        throwIfError(ret, "Stop");
    }

    /**
     * 阻塞等待退出信号（Ctrl+C），收到信号后自动调用 stop()。
     * 推荐用法: init() → start() → waitStop() → 进程退出
     */
    public static int waitStop() {
        return lib.D1_WaitStop();
    }

    /**
     * 设置默认请求处理器。当收到未匹配的请求时回调此处理器。
     * 同一时间只能注册一个处理器，重复调用会覆盖之前的处理器。
     * @param handler 请求处理回调，不能为 null。
     * @throws IllegalArgumentException handler 为 null 时抛出。
     */
    public static void setOnRequest(RequestHandler handler) {
        if (handler == null) {
            throw new IllegalArgumentException("handler 不能为 null");
        }

        requestHandlerRef = handler;

        nativeRequestHandlerRef = (taskId, method, payload, payloadLen,
                outResult, outLen, outError) -> {
            // 调用用户处理器
            Object[] result = handler.handle(taskId, method, payload);
            String pl = (String) result[0];
            String err = result.length > 1 ? (String) result[1] : null;
            int retCode = result.length > 2 ? (Integer) result[2] : 0;

            // 将响应负载写入非托管内存（D1 负责释放）
            if (pl != null && !pl.isEmpty()) {
                byte[] bytes = pl.getBytes(StandardCharsets.UTF_8);
                Memory mem = new Memory(bytes.length);
                mem.write(0, bytes, 0, bytes.length);
                outResult.setValue(mem);
                outLen.setValue(bytes.length);
            } else {
                outResult.setValue(Pointer.NULL);
                outLen.setValue(0);
            }

            // 将错误信息写入非托管内存（D1 负责释放）
            if (err != null && !err.isEmpty()) {
                byte[] bytes = err.getBytes(StandardCharsets.UTF_8);
                Memory mem = new Memory(bytes.length);
                mem.write(0, bytes, 0, bytes.length);
                outError.setValue(mem);
            } else {
                outError.setValue(Pointer.NULL);
            }

            return retCode;
        };

        lib.D1_SetOnRequest(nativeRequestHandlerRef);
    }

    /**
     * 发布（推送）消息到指定目标，不等待响应。
     * @param taskId  任务 ID。
     * @param target  目标标识。
     * @param method 消息名称。
     * @param payload 消息负载，可为 null。
     * @throws D1Exception 发布失败时抛出。
     */
    public static void publish(long taskId, String target, String method, String payload) {
        int len = utf8ByteLength(payload);
        int ret = lib.D1_Publish(taskId, target, method, payload, len);
        throwIfError(ret, "Publish");
    }

    /**
     * 同步调用目标服务，阻塞等待响应。
     * @param taskId     任务 ID。
     * @param kind       调用类型（如 "rpc"）。
     * @param target     目标标识。
     * @param method    消息名称。
     * @param payload    请求负载，可为 null。
     * @param timeoutSec 超时时间（秒），0 表示不超时。
     * @return 包含返回码、负载和错误信息的 CallResult。
     */
    public static CallResult call(long taskId, String kind, String target, String method,
                                   String payload, int timeoutSec) {
        int payloadLen = utf8ByteLength(payload);
        PointerByReference outResult = new PointerByReference();
        IntByReference outLen = new IntByReference();
        PointerByReference outError = new PointerByReference();

        int ret = lib.D1_Call(taskId, kind, target, method, payload,
                payloadLen, timeoutSec, outResult, outLen, outError);

        String pl = ptrToStringAndFree(outResult.getValue(), outLen.getValue());
        String err = ptrToStringAndFree(outError.getValue(), 0);

        return new CallResult(ret, pl, err);
    }

    /**
     * 异步请求目标服务，通过回调接收响应。
     * @param taskId     任务 ID。
     * @param target     目标标识。
     * @param method    消息名称。
     * @param payload    请求负载，可为 null。
     * @param timeoutSec 超时时间（秒）。
     * @param callback   响应回调，不能为 null。
     * @throws IllegalArgumentException callback 为 null 时抛出。
     * @throws D1Exception 请求发送失败时抛出。
     */
    public static void request(long taskId, String target, String method,
                                String payload, int timeoutSec, ResponseCallback callback) {
        if (callback == null) {
            throw new IllegalArgumentException("callback 不能为 null");
        }

        int payloadLen = utf8ByteLength(payload);

        // 创建原生回调（由 JNA 管理其生命周期）
        NativeOnResponse cb = (tId, pl, plLen, err) -> {
            callback.onResponse(tId, pl, err);
        };

        int ret = lib.D1_Request(taskId, target, method, payload,
                payloadLen, timeoutSec, cb);
        throwIfError(ret, "Request");
    }

    /**
     * 在当前请求处理上下文中回复消息。
     * @param taskId  任务 ID。
     * @param method 消息名称。
     * @param payload 回复负载，可为 null。
     * @throws D1Exception 回复失败时抛出。
     */
    public static void reply(long taskId, String method, String payload) {
        int len = utf8ByteLength(payload);
        int ret = lib.D1_Reply(taskId, method, payload, len);
        throwIfError(ret, "Reply");
    }

    /**
     * 从 D1 内置缓存中获取键对应的值。
     * @param taskId 任务 ID。
     * @param key    缓存键。
     * @return 缓存值，若不存在返回 null。
     */
    public static String cacheGet(long taskId, String key) {
        PointerByReference result = new PointerByReference();
        IntByReference resultLen = new IntByReference();

        int ret = lib.D1_CacheGet(taskId, key, result, resultLen);
        if (ret != 0) {
            return null;
        }

        return ptrToStringAndFree(result.getValue(), resultLen.getValue());
    }

    /**
     * 向 D1 内置缓存设置键值对。
     * @param taskId     任务 ID。
     * @param key        缓存键。
     * @param value      缓存值。
     * @param ttlSeconds 过期时间（秒），0 或负数表示永不过期。
     * @throws D1Exception 设置失败时抛出。
     */
    public static void cacheSet(long taskId, String key, String value, int ttlSeconds) {
        int len = utf8ByteLength(value);
        int ret = lib.D1_CacheSet(taskId, key, value, len, ttlSeconds);
        throwIfError(ret, "CacheSet");
    }

    /**
     * 从 D1 内置缓存中删除指定键。
     * @param taskId 任务 ID。
     * @param key    缓存键。
     * @throws D1Exception 删除失败时抛出。
     */
    public static void cacheDelete(long taskId, String key) {
        int ret = lib.D1_CacheDelete(taskId, key);
        throwIfError(ret, "CacheDelete");
    }

    /**
     * 执行数据库查询（SELECT 等），返回 JSON 格式的结果集。
     * @param taskId 任务 ID。
     * @param query  SQL 查询语句。
     * @return JSON 格式的查询结果字符串。
     * @throws D1Exception 查询失败时抛出。
     */
    public static String dbQuery(long taskId, String query) {
        int len = utf8ByteLength(query);
        PointerByReference result = new PointerByReference();
        IntByReference resultLen = new IntByReference();

        int ret = lib.D1_DBQuery(taskId, query, len, result, resultLen);
        throwIfError(ret, "DBQuery");

        String str = ptrToStringAndFree(result.getValue(), resultLen.getValue());
        return str != null ? str : "";
    }

    /**
     * 执行数据库写操作（INSERT / UPDATE / DELETE 等）。
     * @param taskId 任务 ID。
     * @param query  SQL 语句。
     * @return 受影响的行数。
     * @throws D1Exception 执行失败时抛出。
     */
    public static long dbExec(long taskId, String query) {
        int len = utf8ByteLength(query);
        LongByReference affectedRows = new LongByReference();

        int ret = lib.D1_DBExec(taskId, query, len, affectedRows);
        throwIfError(ret, "DBExec");

        return affectedRows.getValue();
    }

    /**
     * 向 D1 键值存储中设置键值对。
     * @param taskId 任务 ID。
     * @param key    键。
     * @param value  值。
     * @throws D1Exception 设置失败时抛出。
     */
    public static void set(long taskId, String key, String value) {
        int len = utf8ByteLength(value);
        int ret = lib.D1_Set(taskId, key, value, len);
        throwIfError(ret, "Set");
    }

    /**
     * 从 D1 键值存储中获取键对应的值。
     * @param taskId 任务 ID。
     * @param key    键。
     * @return 值字符串，若键不存在返回 null。
     */
    public static String get(long taskId, String key) {
        PointerByReference result = new PointerByReference();
        IntByReference resultLen = new IntByReference();

        int ret = lib.D1_Get(taskId, key, result, resultLen);
        if (ret != 0) {
            return null;
        }

        return ptrToStringAndFree(result.getValue(), resultLen.getValue());
    }
}