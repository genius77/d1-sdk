// D1 SDK C# 封装 | 对应 D1 动态库版本: ≥ v1.5.0
// 跨平台支持: Windows (.dll) | Linux (.so) | macOS (.dylib)
//
// 用法: 将本文件加入项目，将动态库放入项目 deps/ 目录即可。
//       项目文件中设置 <AllowUnsafeBlocks>true</AllowUnsafeBlocks> 或忽略。
//
// 依赖: .NET 8.0+，无需额外 NuGet 包。
//
// 内存管理说明:
//   - D1_Version() 返回的字符串为静态常量，不需要释放。
//   - D1_Call / D1_CacheGet / D1_DBQuery / D1_Get 返回的字符串由 D1 分配，
//     本 SDK 封装层自动调用 D1_Free 释放，调用者无需关心。
//   - D1_OnRequestFunc 回调中通过 out 参数返回的字符串由 D1 负责释放。

using System;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text;

namespace Genius77.D1
{
    /// <summary>
    /// D1 运行时异常。当 D1 C API 返回非零错误码时抛出。
    /// </summary>
    public class D1Exception : Exception
    {
        /// <summary>D1 C API 返回的原始错误码。</summary>
        public int ErrorCode { get; }

        /// <summary>触发异常的 D1 函数名称。</summary>
        public string FunctionName { get; }

        internal D1Exception(int errorCode, string functionName)
            : base($"D1 {functionName} failed, error code: {errorCode}")
        {
            ErrorCode = errorCode;
            FunctionName = functionName;
        }
    }

    /// <summary>
    /// D1.Call() 方法调用返回结果。
    /// 包含 C API 返回码、负载数据及错误信息。
    /// </summary>
    public class D1CallResult
    {
        /// <summary>C API 原始返回码，0 表示成功。</summary>
        public int ReturnCode { get; }

        /// <summary>返回的负载字符串，若不存在则为 null。</summary>
        public string Payload { get; }

        /// <summary>返回的错误信息，若无错误则为 null。</summary>
        public string Error { get; }

        /// <summary>调用是否成功（返回码为 0 且 Error 为 null）。</summary>
        public bool IsSuccess => ReturnCode == 0 && Error == null;

        internal D1CallResult(int returnCode, string payload, string error)
        {
            ReturnCode = returnCode;
            Payload = payload;
            Error = error;
        }

        /// <summary>返回结果的简要描述字符串。</summary>
        public override string ToString()
            => $"D1CallResult(Code={ReturnCode}, Payload={Payload ?? "(null)"}, Error={Error ?? "(null)"})";
    }

    // ========================================================================
    //  委托定义 — 供用户使用的公开委托类型
    // ========================================================================

    /// <summary>
    /// 默认请求处理回调委托。当 D1 收到未匹配的请求时调用此处理器。
    /// 由 D1.SetOnRequest() 注册。
    /// </summary>
    /// <param name="taskId">任务 ID。</param>
    /// <param name="method">消息名称。</param>
    /// <param name="payload">请求负载（UTF-8 编码字符串）。</param>
    /// <param name="outResult">通过 out 返回的响应负载，可为 null。</param>
    /// <param name="outError">通过 out 返回的错误信息，可为 null。</param>
    /// <returns>返回 0 表示处理成功，非零表示处理失败。</returns>
    public delegate int D1RequestHandler(
        ulong taskId,
        string method,
        string payload,
        out string outResult,
        out string outError);

    /// <summary>
    /// 异步请求响应回调委托。D1.Request() 完成后通过此委托返回结果。
    /// </summary>
    /// <param name="taskId">任务 ID。</param>
    /// <param name="payload">响应负载，可能为 null。</param>
    /// <param name="error">错误信息，可能为 null。</param>
    public delegate void D1ResponseCallback(
        ulong taskId,
        string payload,
        string error);

    // ========================================================================
    //  内部原生委托 — 精确匹配 C ABI (Cdecl 调用约定)
    // ========================================================================

    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    internal delegate int NativeOnRequestFunc(
        ulong taskId,
        IntPtr method,
        IntPtr payload,
        int payloadLen,
        out IntPtr outResult,
        out int outLen,
        out IntPtr outError);

    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    internal delegate void NativeOnResponse(
        ulong taskId,
        IntPtr payload,
        int payloadLen,
        IntPtr error);

    // ========================================================================
    //  D1 静态封装类 — 全部 17 个 C API 的 C# 封装
    // ========================================================================

    /// <summary>
    /// D1 动态库 C# SDK 静态封装类。
    /// 所有方法均为线程安全的静态方法。首次使用前会自动初始化跨平台库加载。
    /// </summary>
    public static class D1
    {
        // ==============================
        //  跨平台动态库解析
        // ==============================

        static D1()
        {
            NativeLibrary.SetDllImportResolver(typeof(D1).Assembly, ResolveD1Library);
        }

        private static IntPtr ResolveD1Library(
            string libraryName,
            Assembly assembly,
            DllImportSearchPath? searchPath)
        {
            if (libraryName != "D1")
                return IntPtr.Zero;

            if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
                return NativeLibrary.Load("D1.dll", assembly, searchPath);
            else if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX))
                return NativeLibrary.Load("libD1.dylib", assembly, searchPath);
            else // Linux / FreeBSD 等
                return NativeLibrary.Load("libD1.so", assembly, searchPath);
        }

        // ==============================
        //  委托存活引用（防止 GC 回收）
        // ==============================

        private static D1RequestHandler? _requestHandler;
        private static NativeOnRequestFunc? _nativeHandler;

        // ==============================
        //  原生 P/Invoke 声明（17 个函数）
        //  全部使用 Cdecl 调用约定
        // ==============================

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern IntPtr D1_Version();

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Init(IntPtr configPath);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Start();

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Stop();

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_WaitStop();

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern void D1_SetOnRequest(NativeOnRequestFunc handler);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Publish(
            ulong taskId,
            IntPtr target,
            IntPtr method,
            IntPtr payload,
            int payloadLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Call(
            ulong taskId,
            IntPtr kind,
            IntPtr target,
            IntPtr method,
            IntPtr payload,
            int payloadLen,
            int timeoutSec,
            out IntPtr outResult,
            out int outLen,
            out IntPtr outError);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Request(
            ulong taskId,
            IntPtr target,
            IntPtr method,
            IntPtr payload,
            int payloadLen,
            int timeoutSec,
            NativeOnResponse callback);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Reply(
            ulong taskId,
            IntPtr method,
            IntPtr payload,
            int payloadLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_CacheGet(
            ulong taskId,
            IntPtr key,
            out IntPtr result,
            out int resultLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_CacheSet(
            ulong taskId,
            IntPtr key,
            IntPtr value,
            int valueLen,
            int ttlSeconds);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_CacheDelete(ulong taskId, IntPtr key);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_DBQuery(
            ulong taskId,
            IntPtr query,
            int queryLen,
            out IntPtr result,
            out int resultLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_DBExec(
            ulong taskId,
            IntPtr query,
            int queryLen,
            out long affectedRows);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Set(
            ulong taskId,
            IntPtr key,
            IntPtr value,
            int valueLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        private static extern int D1_Get(
            ulong taskId,
            IntPtr key,
            out IntPtr result,
            out int resultLen);

        [DllImport("D1", CallingConvention = CallingConvention.Cdecl)]
        internal static extern void D1_Free(IntPtr ptr);

        // ==============================
        //  内部辅助方法
        // ==============================

        /// <summary>
        /// 将 C# 字符串编码为 UTF-8 并分配非托管内存。
        /// 调用者需负责释放返回的指针。
        /// </summary>
        private static IntPtr StringToUtf8Ptr(string? s)
        {
            if (s == null)
                return IntPtr.Zero;

            byte[] bytes = Encoding.UTF8.GetBytes(s);
            IntPtr ptr = Marshal.AllocHGlobal(bytes.Length);
            Marshal.Copy(bytes, 0, ptr, bytes.Length);
            return ptr;
        }

        /// <summary>
        /// 将 UTF-8 非托管内存指针转换为 C# 字符串，并调用 D1_Free 释放。
        /// 用于 Call / CacheGet / DBQuery / Get 等返回的需要释放的字符串。
        /// </summary>
        private static string? PtrToStringAndFree(IntPtr ptr, int len)
        {
            if (ptr == IntPtr.Zero)
                return null;

            string result;
            if (len > 0)
            {
                byte[] bytes = new byte[len];
                Marshal.Copy(ptr, bytes, 0, len);
                result = Encoding.UTF8.GetString(bytes);
            }
            else
            {
                // 回退方案: 当作 null-terminated 字符串读取
                result = Marshal.PtrToStringUTF8(ptr) ?? string.Empty;
            }

            D1_Free(ptr);
            return result;
        }

        /// <summary>
        /// 检查返回值，若非零则抛出 D1Exception。
        /// </summary>
        private static void ThrowIfError(int code, string functionName)
        {
            if (code != 0)
                throw new D1Exception(code, functionName);
        }

        // ==============================
        //  公开 API — PascalCase 风格
        //  所有方法均包含中文 XML 文档注释
        // ==============================

        /// <summary>
        /// 获取 D1 动态库的版本号字符串。
        /// </summary>
        /// <returns>版本号，例如 "1.1.0"。返回值由 D1 内部持有，无需释放。</returns>
        public static string Version()
        {
            IntPtr ptr = D1_Version();
            // 版本字符串为静态常量，不需要 D1_Free
            return Marshal.PtrToStringUTF8(ptr) ?? string.Empty;
        }

        /// <summary>
        /// 初始化 D1 运行时环境。必须在 Start() 之前调用。
        /// </summary>
        /// <param name="configPath">
        /// 配置文件路径。传入 null 使用默认配置（从 deps/ 目录或环境变量读取）。
        /// </param>
        /// <exception cref="D1Exception">初始化失败时抛出，可从 ErrorCode 获取详细错误码。</exception>
        public static void Init(string? configPath)
        {
            IntPtr pathPtr = StringToUtf8Ptr(configPath);
            try
            {
                int ret = D1_Init(pathPtr);
                ThrowIfError(ret, "Init");
            }
            finally
            {
                if (pathPtr != IntPtr.Zero)
                    Marshal.FreeHGlobal(pathPtr);
            }
        }

        /// <summary>
        /// 启动 D1 运行时。必须在 Init() 成功后调用。
        /// </summary>
        /// <exception cref="D1Exception">启动失败时抛出。</exception>
        public static void Start()
        {
            int ret = D1_Start();
            ThrowIfError(ret, "Start");
        }

        /// <summary>
        /// 停止 D1 运行时。阻塞当前线程直到所有进行中的任务完成。
        /// </summary>
        /// <exception cref="D1Exception">停止失败时抛出。</exception>
        public static void Stop()
        {
            int ret = D1_Stop();
            ThrowIfError(ret, "Stop");
        }

        /// <summary>
        /// 阻塞等待退出信号（Ctrl+C），收到信号后自动调用 Stop()。
        /// 推荐用法: Init() → Start() → WaitStop() → 进程退出
        /// </summary>
        public static int WaitStop()
        {
            return D1_WaitStop();
        }

        /// <summary>
        /// 设置默认请求处理器。当收到未通过路由匹配的请求时回调此处理器。
        /// 同一时间只能注册一个处理器，重复调用会覆盖之前的处理器。
        /// </summary>
        /// <param name="handler">请求处理回调。不能为 null。</param>
        /// <exception cref="ArgumentNullException">handler 为 null 时抛出。</exception>
        public static void SetOnRequest(D1RequestHandler handler)
        {
            if (handler == null)
                throw new ArgumentNullException(nameof(handler));

            _requestHandler = handler;

            // 构建原生回调: 将 C 字符串参数转为 C# string，调用用户处理器，
            // 再将 out 参数序列化回非托管内存（由 D1 负责 D1_Free）
            _nativeHandler = (taskId, methodPtr, payloadPtr, payloadLen,
                out IntPtr outResultPtr, out int outLen, out IntPtr outErrorPtr) =>
            {
                // 入参转换: 原生指针 -> C# 字符串
                string method = Marshal.PtrToStringUTF8(methodPtr) ?? string.Empty;

                string payload;
                if (payloadLen > 0 && payloadPtr != IntPtr.Zero)
                {
                    byte[] buf = new byte[payloadLen];
                    Marshal.Copy(payloadPtr, buf, 0, payloadLen);
                    payload = Encoding.UTF8.GetString(buf);
                }
                else
                {
                    payload = string.Empty;
                }

                // 调用用户处理器
                int result = handler(taskId, method, payload,
                    out string outResult, out string outError);

                // 出参转换: C# 字符串 -> 非托管内存
                if (outResult != null)
                {
                    byte[] bytes = Encoding.UTF8.GetBytes(outResult);
                    outResultPtr = Marshal.AllocHGlobal(bytes.Length);
                    Marshal.Copy(bytes, 0, outResultPtr, bytes.Length);
                    outLen = bytes.Length;
                }
                else
                {
                    outResultPtr = IntPtr.Zero;
                    outLen = 0;
                }

                if (outError != null)
                {
                    byte[] bytes = Encoding.UTF8.GetBytes(outError);
                    outErrorPtr = Marshal.AllocHGlobal(bytes.Length);
                    Marshal.Copy(bytes, 0, outErrorPtr, bytes.Length);
                }
                else
                {
                    outErrorPtr = IntPtr.Zero;
                }

                return result;
            };

            D1_SetOnRequest(_nativeHandler);
        }

        /// <summary>
        /// 发布（推送）消息到指定目标，不等待响应。适用于事件广播场景。
        /// </summary>
        /// <param name="taskId">任务 ID，用于关联上下文。</param>
        /// <param name="target">目标标识字符串。</param>
        /// <param name="method">消息名称。</param>
        /// <param name="payload">消息负载（UTF-8 字符串），可为 null。</param>
        /// <exception cref="D1Exception">发布失败时抛出。</exception>
        public static void Publish(ulong taskId, string target, string method, string? payload)
        {
            IntPtr targetPtr = StringToUtf8Ptr(target);
            IntPtr methodPtr = StringToUtf8Ptr(method);
            IntPtr payloadPtr = StringToUtf8Ptr(payload);
            try
            {
                int ret = D1_Publish(taskId, targetPtr, methodPtr, payloadPtr,
                    payload != null ? Encoding.UTF8.GetByteCount(payload) : 0);
                ThrowIfError(ret, "Publish");
            }
            finally
            {
                if (targetPtr != IntPtr.Zero) Marshal.FreeHGlobal(targetPtr);
                if (methodPtr != IntPtr.Zero) Marshal.FreeHGlobal(methodPtr);
                if (payloadPtr != IntPtr.Zero) Marshal.FreeHGlobal(payloadPtr);
            }
        }

        /// <summary>
        /// 同步调用目标服务，阻塞等待响应。适用于 RPC 风格的请求-回复模式。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="kind">处理器类型："default"/"conn"/"script"/"service"/"exec"。</param>
        /// <param name="target">目标标识。</param>
        /// <param name="method">消息名称。</param>
        /// <param name="payload">请求负载，可为 null。</param>
        /// <param name="timeoutSec">超时时间（秒），0 表示永不超时。</param>
        /// <returns>包含返回码、负载和错误信息的 D1CallResult。</returns>
        public static D1CallResult Call(
            ulong taskId,
            string kind,
            string target,
            string method,
            string? payload,
            int timeoutSec)
        {
            IntPtr kindPtr = StringToUtf8Ptr(kind);
            IntPtr targetPtr = StringToUtf8Ptr(target);
            IntPtr methodPtr = StringToUtf8Ptr(method);
            IntPtr payloadPtr = StringToUtf8Ptr(payload);
            try
            {
                int ret = D1_Call(taskId, kindPtr, targetPtr, methodPtr, payloadPtr,
                    payload != null ? Encoding.UTF8.GetByteCount(payload) : 0,
                    timeoutSec,
                    out IntPtr outResultPtr, out int outLen, out IntPtr outErrorPtr);

                string? outResult = PtrToStringAndFree(outResultPtr, outLen);
                string? outError = PtrToStringAndFree(outErrorPtr, 0);

                return new D1CallResult(ret, outResult, outError);
            }
            finally
            {
                if (kindPtr != IntPtr.Zero) Marshal.FreeHGlobal(kindPtr);
                if (targetPtr != IntPtr.Zero) Marshal.FreeHGlobal(targetPtr);
                if (methodPtr != IntPtr.Zero) Marshal.FreeHGlobal(methodPtr);
                if (payloadPtr != IntPtr.Zero) Marshal.FreeHGlobal(payloadPtr);
            }
        }

        /// <summary>
        /// 异步请求目标服务，通过回调函数接收响应。适用于非阻塞调用场景。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="target">目标标识。</param>
        /// <param name="method">消息名称。</param>
        /// <param name="payload">请求负载，可为 null。</param>
        /// <param name="timeoutSec">超时时间（秒）。</param>
        /// <param name="callback">响应回调函数，不能为 null。</param>
        /// <exception cref="ArgumentNullException">callback 为 null 时抛出。</exception>
        /// <exception cref="D1Exception">请求发送失败时抛出。</exception>
        public static void Request(
            ulong taskId,
            string target,
            string method,
            string? payload,
            int timeoutSec,
            D1ResponseCallback callback)
        {
            if (callback == null)
                throw new ArgumentNullException(nameof(callback));

            IntPtr targetPtr = StringToUtf8Ptr(target);
            IntPtr methodPtr = StringToUtf8Ptr(method);
            IntPtr payloadPtr = StringToUtf8Ptr(payload);

            // 创建原生回调 — 必须保持引用以防 GC 回收
            NativeOnResponse nativeCb = (tId, pPtr, pLen, errPtr) =>
            {
                string? pl = null;
                if (pLen > 0 && pPtr != IntPtr.Zero)
                {
                    byte[] buf = new byte[pLen];
                    Marshal.Copy(pPtr, buf, 0, pLen);
                    pl = Encoding.UTF8.GetString(buf);
                }

                string? err = errPtr != IntPtr.Zero
                    ? (Marshal.PtrToStringUTF8(errPtr) ?? string.Empty)
                    : null;

                callback(tId, pl, err);
            };

            try
            {
                int ret = D1_Request(taskId, targetPtr, methodPtr, payloadPtr,
                    payload != null ? Encoding.UTF8.GetByteCount(payload) : 0,
                    timeoutSec, nativeCb);
                ThrowIfError(ret, "Request");
            }
            finally
            {
                if (targetPtr != IntPtr.Zero) Marshal.FreeHGlobal(targetPtr);
                if (methodPtr != IntPtr.Zero) Marshal.FreeHGlobal(methodPtr);
                if (payloadPtr != IntPtr.Zero) Marshal.FreeHGlobal(payloadPtr);
            }

            // 确保回调委托在 Request 调用期间不被 GC 回收
            GC.KeepAlive(nativeCb);
        }

        /// <summary>
        /// 在当前请求处理的上下文中回复消息。通常在处理器的回调内调用。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="method">消息名称。</param>
        /// <param name="payload">回复负载，可为 null。</param>
        /// <exception cref="D1Exception">回复失败时抛出。</exception>
        public static void Reply(ulong taskId, string method, string? payload)
        {
            IntPtr methodPtr = StringToUtf8Ptr(method);
            IntPtr payloadPtr = StringToUtf8Ptr(payload);
            try
            {
                int ret = D1_Reply(taskId, methodPtr, payloadPtr,
                    payload != null ? Encoding.UTF8.GetByteCount(payload) : 0);
                ThrowIfError(ret, "Reply");
            }
            finally
            {
                if (methodPtr != IntPtr.Zero) Marshal.FreeHGlobal(methodPtr);
                if (payloadPtr != IntPtr.Zero) Marshal.FreeHGlobal(payloadPtr);
            }
        }

        /// <summary>
        /// 从 D1 内置缓存中获取键对应的值。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="key">缓存键。</param>
        /// <returns>缓存值字符串，若键不存在或获取失败则返回 null。</returns>
        public static string? CacheGet(ulong taskId, string key)
        {
            IntPtr keyPtr = StringToUtf8Ptr(key);
            try
            {
                int ret = D1_CacheGet(taskId, keyPtr, out IntPtr resultPtr, out int resultLen);
                if (ret != 0)
                    return null;
                return PtrToStringAndFree(resultPtr, resultLen);
            }
            finally
            {
                if (keyPtr != IntPtr.Zero) Marshal.FreeHGlobal(keyPtr);
            }
        }

        /// <summary>
        /// 向 D1 内置缓存设置键值对。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="key">缓存键。</param>
        /// <param name="value">缓存值，不可为 null。</param>
        /// <param name="ttlSeconds">过期时间（秒），0 或负数表示永不过期。</param>
        /// <exception cref="D1Exception">设置失败时抛出。</exception>
        public static void CacheSet(ulong taskId, string key, string value, int ttlSeconds)
        {
            IntPtr keyPtr = StringToUtf8Ptr(key);
            IntPtr valuePtr = StringToUtf8Ptr(value);
            try
            {
                int ret = D1_CacheSet(taskId, keyPtr, valuePtr,
                    value != null ? Encoding.UTF8.GetByteCount(value) : 0,
                    ttlSeconds);
                ThrowIfError(ret, "CacheSet");
            }
            finally
            {
                if (keyPtr != IntPtr.Zero) Marshal.FreeHGlobal(keyPtr);
                if (valuePtr != IntPtr.Zero) Marshal.FreeHGlobal(valuePtr);
            }
        }

        /// <summary>
        /// 从 D1 内置缓存中删除指定键。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="key">缓存键。</param>
        /// <exception cref="D1Exception">删除失败时抛出。</exception>
        public static void CacheDelete(ulong taskId, string key)
        {
            IntPtr keyPtr = StringToUtf8Ptr(key);
            try
            {
                int ret = D1_CacheDelete(taskId, keyPtr);
                ThrowIfError(ret, "CacheDelete");
            }
            finally
            {
                if (keyPtr != IntPtr.Zero) Marshal.FreeHGlobal(keyPtr);
            }
        }

        /// <summary>
        /// 执行数据库查询（SELECT 等），返回 JSON 格式的结果集。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="query">SQL 查询语句。</param>
        /// <returns>JSON 格式的查询结果字符串。注意调用后内存已自动释放。</returns>
        /// <exception cref="D1Exception">查询失败时抛出。</exception>
        public static string DBQuery(ulong taskId, string query)
        {
            IntPtr queryPtr = StringToUtf8Ptr(query);
            try
            {
                int ret = D1_DBQuery(taskId, queryPtr,
                    query != null ? Encoding.UTF8.GetByteCount(query) : 0,
                    out IntPtr resultPtr, out int resultLen);
                ThrowIfError(ret, "DBQuery");
                return PtrToStringAndFree(resultPtr, resultLen) ?? string.Empty;
            }
            finally
            {
                if (queryPtr != IntPtr.Zero) Marshal.FreeHGlobal(queryPtr);
            }
        }

        /// <summary>
        /// 执行数据库写操作（INSERT / UPDATE / DELETE 等）。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="query">SQL 语句。</param>
        /// <returns>受影响的行数。</returns>
        /// <exception cref="D1Exception">执行失败时抛出。</exception>
        public static long DBExec(ulong taskId, string query)
        {
            IntPtr queryPtr = StringToUtf8Ptr(query);
            try
            {
                int ret = D1_DBExec(taskId, queryPtr,
                    query != null ? Encoding.UTF8.GetByteCount(query) : 0,
                    out long affectedRows);
                ThrowIfError(ret, "DBExec");
                return affectedRows;
            }
            finally
            {
                if (queryPtr != IntPtr.Zero) Marshal.FreeHGlobal(queryPtr);
            }
        }

        /// <summary>
        /// 向 D1 键值存储中设置键值对。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="key">键。</param>
        /// <param name="value">值。</param>
        /// <exception cref="D1Exception">设置失败时抛出。</exception>
        public static void Set(ulong taskId, string key, string value)
        {
            IntPtr keyPtr = StringToUtf8Ptr(key);
            IntPtr valuePtr = StringToUtf8Ptr(value);
            try
            {
                int ret = D1_Set(taskId, keyPtr, valuePtr,
                    value != null ? Encoding.UTF8.GetByteCount(value) : 0);
                ThrowIfError(ret, "Set");
            }
            finally
            {
                if (keyPtr != IntPtr.Zero) Marshal.FreeHGlobal(keyPtr);
                if (valuePtr != IntPtr.Zero) Marshal.FreeHGlobal(valuePtr);
            }
        }

        /// <summary>
        /// 从 D1 键值存储中获取键对应的值。
        /// </summary>
        /// <param name="taskId">任务 ID。</param>
        /// <param name="key">键。</param>
        /// <returns>值字符串，若键不存在则返回 null。注意调用后内存已自动释放。</returns>
        public static string? Get(ulong taskId, string key)
        {
            IntPtr keyPtr = StringToUtf8Ptr(key);
            try
            {
                int ret = D1_Get(taskId, keyPtr, out IntPtr resultPtr, out int resultLen);
                if (ret != 0)
                    return null;
                return PtrToStringAndFree(resultPtr, resultLen);
            }
            finally
            {
                if (keyPtr != IntPtr.Zero) Marshal.FreeHGlobal(keyPtr);
            }
        }
    }
}