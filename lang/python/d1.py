#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
D1 SDK Python 封装 | 对应 D1 动态库版本: >= v1.5.0

本模块通过 ctypes 封装 D1 动态库的全部 17 个 C API，
提供 Pythonic 的调用方式。

使用前请将动态库及头文件放入 deps/ 目录:
    deps/
    ├── d1.h
    ├── libd1.so       (Linux)
    ├── libd1.dylib    (macOS)
    └── d1.dll         (Windows)

自动检测平台并加载对应动态库。

基本用法:
    from d1 import D1

    with D1() as d1:
        print("版本:", d1.version())
        d1.init(None)
        d1.start()
        d1.wait_stop()
"""

import ctypes
import os
import platform
import sys
from ctypes import (
    CFUNCTYPE,
    POINTER,
    byref,
    c_char_p,
    c_int,
    c_int64,
    c_uint64,
    c_void_p,
    pointer,
    string_at,
)

# ---------------------------------------------------------------------------
# 平台检测 & 动态库加载
# ---------------------------------------------------------------------------

_SYSTEM = platform.system()
if _SYSTEM == "Linux":
    _LIBNAME = "libd1.so"
elif _SYSTEM == "Darwin":
    _LIBNAME = "libd1.dylib"
elif _SYSTEM == "Windows":
    _LIBNAME = "d1.dll"
else:
    raise RuntimeError(f"Unsupported OS: {_SYSTEM}")

# SDK 文件所在目录的 deps 路径
# lang/python/d1.py -> ../../deps/
_SDK_DIR = os.path.dirname(os.path.abspath(__file__))
_DEPS_DIR = os.path.normpath(os.path.join(_SDK_DIR, "..", "..", "deps"))
_LIB_PATH = os.path.join(_DEPS_DIR, _LIBNAME)

if not os.path.exists(_LIB_PATH):
    raise FileNotFoundError(
        f"找不到 D1 动态库文件: {_LIB_PATH}\n"
        f"请将 {_LIBNAME} 放入 {_DEPS_DIR}/ 目录"
    )

# 尝试加载动态库
try:
    _lib = ctypes.CDLL(_LIB_PATH)
except OSError as e:
    raise OSError(
        f"加载 D1 动态库失败: {e}\n"
        f"库路径: {_LIB_PATH}\n"
        f"Linux: 请检查 LD_LIBRARY_PATH 是否包含 deps/ 目录\n"
        f"macOS: 请检查 DYLD_LIBRARY_PATH 是否包含 deps/ 目录"
    ) from e

# ---------------------------------------------------------------------------
# 回调函数类型定义
# ---------------------------------------------------------------------------

# D1 默认消息处理器回调类型
# int (*)(uint64_t task_id, const char* method,
#          const char* params, int params_len,
#          char** out_result, int* out_len, char** out_error)
D1_HANDLER_CB = CFUNCTYPE(
    c_int,                        # 返回值
    c_uint64,                      # task_id
    c_char_p,                      # method
    c_char_p,                      # params
    c_int,                         # params_len
    POINTER(c_char_p),             # out_result
    POINTER(c_int),                # out_len
    POINTER(c_char_p),             # out_error
)

# D1 异步请求回调类型
# void (*)(uint64_t task_id, const char* params, int params_len, const char* error)
D1_REQUEST_CB = CFUNCTYPE(
    None,                          # 返回值 void
    c_uint64,                      # task_id
    c_char_p,                      # params
    c_int,                         # params_len
    c_char_p,                      # error
)

# ---------------------------------------------------------------------------
# 为所有 17 个 C API 设置 argtypes / restype
# ---------------------------------------------------------------------------

# 1. D1_Version() -> const char*
_lib.D1_Version.restype = c_char_p

# 2. D1_Init(const char* config_path) -> int
_lib.D1_Init.argtypes = [c_char_p]
_lib.D1_Init.restype = c_int

# 3. D1_Start(void) -> int
_lib.D1_Start.argtypes = []
_lib.D1_Start.restype = c_int

# 4. D1_Stop(void) -> int
_lib.D1_Stop.argtypes = []
_lib.D1_Stop.restype = c_int

# 5. D1_WaitStop(void) -> int
_lib.D1_WaitStop.argtypes = []
_lib.D1_WaitStop.restype = c_int

# 6. D1_SetOnRequest(handler) -> void
_lib.D1_SetOnRequest.argtypes = [D1_HANDLER_CB]
_lib.D1_SetOnRequest.restype = None

# 7. D1_Publish(task_id, target, method, params, payload_len) -> int
_lib.D1_Publish.argtypes = [c_uint64, c_char_p, c_char_p, c_char_p, c_int]
_lib.D1_Publish.restype = c_int

# 8. D1_Call(task_id, kind, target, method, params, payload_len,
#            timeout_sec, out_result, out_len, out_error) -> int
_lib.D1_Call.argtypes = [
    c_uint64, c_int, c_char_p, c_char_p, c_char_p,
    c_int, c_int, POINTER(c_char_p), POINTER(c_int), POINTER(c_char_p),
]
_lib.D1_Call.restype = c_int

# 9. D1_Request(task_id, target, method, params, payload_len,
#               timeout_sec, callback) -> int
_lib.D1_Request.argtypes = [
    c_uint64, c_char_p, c_char_p, c_char_p,
    c_int, c_int, D1_REQUEST_CB,
]
_lib.D1_Request.restype = c_int

# 10. D1_Reply(task_id, method, params, payload_len) -> int
_lib.D1_Reply.argtypes = [c_uint64, c_char_p, c_char_p, c_int]
_lib.D1_Reply.restype = c_int

# 11. D1_CacheGet(task_id, key, result, result_len) -> int
_lib.D1_CacheGet.argtypes = [c_uint64, c_char_p, POINTER(c_char_p), POINTER(c_int)]
_lib.D1_CacheGet.restype = c_int

# 12. D1_CacheSet(task_id, key, value, value_len, ttl_seconds) -> int
_lib.D1_CacheSet.argtypes = [c_uint64, c_char_p, c_char_p, c_int, c_int]
_lib.D1_CacheSet.restype = c_int

# 13. D1_CacheDelete(task_id, key) -> int
_lib.D1_CacheDelete.argtypes = [c_uint64, c_char_p]
_lib.D1_CacheDelete.restype = c_int

# 14. D1_DBQuery(task_id, query, query_len, result, result_len) -> int
_lib.D1_DBQuery.argtypes = [c_uint64, c_char_p, c_int, POINTER(c_char_p), POINTER(c_int)]
_lib.D1_DBQuery.restype = c_int

# 15. D1_DBExec(task_id, query, query_len, affected_rows) -> int
_lib.D1_DBExec.argtypes = [c_uint64, c_char_p, c_int, POINTER(c_int64)]
_lib.D1_DBExec.restype = c_int

# 16. D1_Set(task_id, key, value, value_len) -> int
_lib.D1_Set.argtypes = [c_uint64, c_char_p, c_char_p, c_int]
_lib.D1_Set.restype = c_int

# 17. D1_Get(task_id, key, result, result_len) -> int
_lib.D1_Get.argtypes = [c_uint64, c_char_p, POINTER(c_char_p), POINTER(c_int)]
_lib.D1_Get.restype = c_int

# D1_Free(ptr) -> void
_lib.D1_Free.argtypes = [c_void_p]
_lib.D1_Free.restype = None


# ---------------------------------------------------------------------------
# 异常类
# ---------------------------------------------------------------------------

class D1Error(Exception):
    """D1 运行时错误。

    当 D1 C API 返回非零错误码时抛出。
    """

    def __init__(self, message, code=-1):
        super().__init__(message)
        self.code = code


# ---------------------------------------------------------------------------
# D1 主类
# ---------------------------------------------------------------------------

class D1:
    """D1 运行时 Python 封装。

    封装 D1 动态库的全部 17 个 C API，提供 Python 风格的调用接口。

    支持上下文管理器（with 语句），在退出时自动调用 stop()。

    基本用法:
        with D1() as d1:
            print("版本:", d1.version())
            d1.init("config.yaml")
            d1.start()
            d1.set_on_request(my_handler)
            d1.wait_stop()

    或手动管理生命周期:
        d1 = D1()
        d1.init("config.yaml")
        d1.start()
        # ... 使用 D1 ...
        d1.stop()
    """

    def __init__(self):
        """创建 D1 实例。

        注意: 实际使用时通常只创建一个 D1 实例。
        """
        self._handler_ref = None   # 保持回调引用，防止被 GC
        self._request_cbs = {}     # task_id -> (callback, keepalive)
        self._initialized = False
        self._started = False

    # === 上下文管理器 ===

    def __enter__(self):
        """进入 with 语句块。"""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """退出 with 语句块时自动停止 D1。

        无论是否发生异常，都确保 stop() 被调用。
        """
        try:
            self.stop()
        except D1Error:
            pass  # 忽略退出时的停止错误
        return False  # 不吞异常

    # === 1. Version ===

    def version(self):
        """获取 D1 动态库版本号。

        Returns:
            str: 版本号字符串，如 "v1.1.0"
        """
        result = _lib.D1_Version()
        if result is None:
            return ""
        return result.decode("utf-8") if isinstance(result, bytes) else str(result)

    # === 2. Init ===

    def init(self, config_path=None):
        """初始化 D1 运行时。

        Args:
            config_path: 配置文件路径，None 表示使用默认配置。

        Raises:
            D1Error: 初始化失败。
        """
        if self._initialized:
            raise D1Error("D1 already initialized")

        c_config = None
        if config_path is not None:
            c_config = config_path.encode("utf-8") if isinstance(config_path, str) else config_path

        ret = _lib.D1_Init(c_config)
        if ret != 0:
            raise D1Error(f"D1_Init failed, error code: {ret}", code=ret)
        self._initialized = True

    # === 3. Start ===

    def start(self):
        """启动 D1 运行时。

        Raises:
            D1Error: 尚未初始化或启动失败。
        """
        if not self._initialized:
            raise D1Error("D1 not initialized, call init() first")
        if self._started:
            raise D1Error("D1 already started")

        ret = _lib.D1_Start()
        if ret != 0:
            raise D1Error(f"D1_Start failed, error code: {ret}", code=ret)
        self._started = True

    # === 4. Stop ===

    def stop(self):
        """停止 D1 运行时并释放资源。

        Raises:
            D1Error: 停止失败。
        """
        if not self._started:
            return

        ret = _lib.D1_Stop()
        self._started = False
        if ret != 0:
            raise D1Error(f"D1_Stop failed, error code: {ret}", code=ret)

    # === 5. Wait ===

    def wait_stop(self):
        """阻塞等待退出信号（Ctrl+C），收到信号后自动调用 stop()。

        推荐用法: init() → start() → wait_stop() → 进程退出
        D1_WaitStop 内部监听 SIGINT/SIGTERM，无需用户手动处理信号。
        """
        return _lib.D1_WaitStop()

    # === 6. SetOnRequest ===

    def set_on_request(self, handler):
        """设置默认消息处理器。

        Args:
            handler: 可调用对象，签名为 handler(task_id, method, params) -> bytes 或 None。
                     返回 bytes 作为响应载荷，或返回 None 表示无响应。
                     如需返回错误，抛异常即可。

        示例:
            def my_handler(task_id, method, params):
                print(f"[{task_id}] {method}: {params}")
                return b"ack"

            d1.set_on_request(my_handler)
        """
        # 创建 C 回调闭包
        def _cb(task_id, method, params, payload_len, out_result, out_len, out_error):
            try:
                py_params = string_at(params, payload_len) if params and payload_len > 0 else b""
                py_method = method.decode("utf-8") if method else ""

                result = handler(task_id, py_method, py_params)

                if result is not None:
                    result_bytes = result if isinstance(result, bytes) else str(result).encode("utf-8")
                    out_result[0] = ctypes.cast(
                        ctypes.create_string_buffer(result_bytes, len(result_bytes)),
                        c_char_p,
                    )
                    out_len[0] = len(result_bytes)

                return 0
            except Exception as exc:
                err_bytes = str(exc).encode("utf-8")
                out_error[0] = ctypes.cast(
                    ctypes.create_string_buffer(err_bytes, len(err_bytes)),
                    c_char_p,
                )
                return -1

        self._handler_ref = D1_HANDLER_CB(_cb)
        _lib.D1_SetOnRequest(self._handler_ref)

    # === 7. Publish ===

    def publish(self, task_id, target, method, params):
        """发布消息到指定目标（单向，不等待响应）。

        Args:
            task_id (int): 任务标识。
            target (str): 目标地址。
            method (str): 消息方法。
            params (str | bytes): 消息参数。

        Raises:
            D1Error: 发布失败。
        """
        c_target = target.encode("utf-8")
        c_method = method.encode("utf-8")
        c_params, p_len = self._encode_payload(params)

        ret = _lib.D1_Publish(
            c_uint64(task_id), c_target, c_method, c_params, p_len
        )
        if ret != 0:
            raise D1Error(f"D1_Publish failed, error code: {ret}", code=ret)

    # === 8. Call ===

    def call(self, task_id, kind, target, method, params, timeout_sec):
        """同步调用远程目标并等待响应。

        Args:
            task_id (int): 任务标识。
            kind (int): 调用类型。
            target (str): 目标地址。
            method (str): 消息方法。
            params (str | bytes): 消息参数。
            timeout_sec (int): 超时秒数。

        Returns:
            dict: {"payload": bytes, "error": str | None}
                  成功时 error 为 None。

        Raises:
            D1Error: 调用失败。
        """
        c_target = target.encode("utf-8")
        c_method = method.encode("utf-8")
        c_params, p_len = self._encode_payload(params)

        out_result = c_char_p()
        out_len = c_int()
        out_error = c_char_p()

        ret = _lib.D1_Call(
            c_uint64(task_id), c_int(kind),
            c_target, c_method, c_params, p_len, c_int(timeout_sec),
            byref(out_result), byref(out_len), byref(out_error),
        )

        # 提取结果（在释放前复制）
        result_payload = None
        result_error = None

        if out_result.value is not None:
            result_payload = string_at(out_result.value, out_len.value)

        if out_error.value is not None:
            result_error = string_at(out_error.value).decode("utf-8")

        # 释放 D1 分配的内存
        if out_result.value is not None:
            _lib.D1_Free(out_result.value)
        if out_error.value is not None:
            _lib.D1_Free(out_error.value)

        if ret != 0:
            err_msg = result_error if result_error else f"错误码: {ret}"
            return {"payload": result_payload, "error": err_msg}

        return {"payload": result_payload, "error": None}

    # === 9. Request ===

    def request(self, task_id, target, method, params, timeout_sec, callback):
        """异步请求远程目标。

        Args:
            task_id (int): 任务标识。
            target (str): 目标地址。
            method (str): 消息方法。
            params (str | bytes): 消息参数。
            timeout_sec (int): 超时秒数。
            callback: 回调函数，签名 callback(task_id, payload, error)。

        Raises:
            D1Error: 请求提交失败。
        """
        # 创建 C 回调闭包
        def _cb(tid, cb_payload, cb_payload_len, cb_error):
            py_payload = None
            py_error = None
            if cb_payload is not None and cb_payload_len > 0:
                py_payload = string_at(cb_payload, cb_payload_len)
            if cb_error is not None:
                py_error = cb_error.decode("utf-8") if isinstance(cb_error, bytes) else str(cb_error)
            callback(tid, py_payload, py_error)

        cb_ref = D1_REQUEST_CB(_cb)

        # 保持引用防止被 GC
        self._request_cbs[task_id] = (callback, cb_ref)

        c_target = target.encode("utf-8")
        c_method = method.encode("utf-8")
        c_params, p_len = self._encode_payload(params)

        ret = _lib.D1_Request(
            c_uint64(task_id), c_target, c_method, c_params, p_len,
            c_int(timeout_sec), cb_ref,
        )

        if ret != 0:
            self._request_cbs.pop(task_id, None)
            raise D1Error(f"D1_Request failed, error code: {ret}", code=ret)

    # === 10. Reply ===

    def reply(self, task_id, method, params):
        """回复消息（通常在 handler 内部使用）。

        Args:
            task_id (int): 原始任务标识。
            method (str): 回复消息方法。
            params (str | bytes): 回复参数。

        Raises:
            D1Error: 回复失败。
        """
        c_method = method.encode("utf-8")
        c_params, p_len = self._encode_payload(params)

        ret = _lib.D1_Reply(c_uint64(task_id), c_method, c_params, p_len)
        if ret != 0:
            raise D1Error(f"D1_Reply failed, error code: {ret}", code=ret)

    # === 11. CacheGet ===

    def cache_get(self, task_id, key):
        """从缓存中获取键对应的值。

        Args:
            task_id (int): 任务标识。
            key (str): 缓存键。

        Returns:
            bytes | None: 缓存值，键不存在返回 None。

        Raises:
            D1Error: 获取失败。
        """
        c_key = key.encode("utf-8")
        result = c_char_p()
        result_len = c_int()

        ret = _lib.D1_CacheGet(c_uint64(task_id), c_key, byref(result), byref(result_len))

        py_result = None
        if result.value is not None and result_len.value > 0:
            py_result = string_at(result.value, result_len.value)

        if result.value is not None:
            _lib.D1_Free(result.value)

        if ret != 0:
            raise D1Error(f"D1_CacheGet failed, error code: {ret}", code=ret)

        return py_result

    # === 12. CacheSet ===

    def cache_set(self, task_id, key, value, ttl_seconds):
        """设置缓存键值对。

        Args:
            task_id (int): 任务标识。
            key (str): 缓存键。
            value (str | bytes): 缓存值。
            ttl_seconds (int): 过期时间（秒），<=0 表示永不过期。

        Raises:
            D1Error: 设置失败。
        """
        c_key = key.encode("utf-8")
        c_val, v_len = self._encode_payload(value)

        ret = _lib.D1_CacheSet(c_uint64(task_id), c_key, c_val, v_len, c_int(ttl_seconds))
        if ret != 0:
            raise D1Error(f"D1_CacheSet failed, error code: {ret}", code=ret)

    # === 13. CacheDelete ===

    def cache_delete(self, task_id, key):
        """从缓存中删除指定键。

        Args:
            task_id (int): 任务标识。
            key (str): 缓存键。

        Raises:
            D1Error: 删除失败。
        """
        c_key = key.encode("utf-8")

        ret = _lib.D1_CacheDelete(c_uint64(task_id), c_key)
        if ret != 0:
            raise D1Error(f"D1_CacheDelete failed, error code: {ret}", code=ret)

    # === 14. DBQuery ===

    def db_query(self, task_id, query):
        """执行数据库查询并返回结果。

        Args:
            task_id (int): 任务标识。
            query (str): SQL 查询语句。

        Returns:
            bytes: 查询结果（通常为 JSON 格式）。

        Raises:
            D1Error: 查询失败。
        """
        c_query = query.encode("utf-8")
        result = c_char_p()
        result_len = c_int()

        ret = _lib.D1_DBQuery(
            c_uint64(task_id), c_query, c_int(len(c_query)),
            byref(result), byref(result_len),
        )

        py_result = None
        if result.value is not None and result_len.value > 0:
            py_result = string_at(result.value, result_len.value)

        if result.value is not None:
            _lib.D1_Free(result.value)

        if ret != 0:
            raise D1Error(f"D1_DBQuery failed, error code: {ret}", code=ret)

        return py_result

    # === 15. DBExec ===

    def db_exec(self, task_id, query):
        """执行数据库写操作。

        Args:
            task_id (int): 任务标识。
            query (str): SQL 写操作语句。

        Returns:
            int: 受影响的行数。

        Raises:
            D1Error: 执行失败。
        """
        c_query = query.encode("utf-8")
        affected = c_int64()

        ret = _lib.D1_DBExec(c_uint64(task_id), c_query, c_int(len(c_query)), byref(affected))
        if ret != 0:
            raise D1Error(f"D1_DBExec failed, error code: {ret}", code=ret)

        return affected.value

    # === 16. Set ===

    def set(self, task_id, key, value):
        """向 D1 内建键值存储设置键值对。

        Args:
            task_id (int): 任务标识。
            key (str): 键。
            value (str | bytes): 值。

        Raises:
            D1Error: 设置失败。
        """
        c_key = key.encode("utf-8")
        c_val, v_len = self._encode_payload(value)

        ret = _lib.D1_Set(c_uint64(task_id), c_key, c_val, v_len)
        if ret != 0:
            raise D1Error(f"D1_Set failed, error code: {ret}", code=ret)

    # === 17. Get ===

    def get(self, task_id, key):
        """从 D1 内建键值存储获取键对应的值。

        Args:
            task_id (int): 任务标识。
            key (str): 键。

        Returns:
            bytes | None: 值，键不存在返回 None。

        Raises:
            D1Error: 获取失败。
        """
        c_key = key.encode("utf-8")
        result = c_char_p()
        result_len = c_int()

        ret = _lib.D1_Get(c_uint64(task_id), c_key, byref(result), byref(result_len))

        py_result = None
        if result.value is not None and result_len.value > 0:
            py_result = string_at(result.value, result_len.value)

        if result.value is not None:
            _lib.D1_Free(result.value)

        if ret != 0:
            raise D1Error(f"D1_Get failed, error code: {ret}", code=ret)

        return py_result

    # === 辅助方法 ===

    @staticmethod
    def _encode_payload(data):
        """编码载荷为 C 兼容格式。

        Args:
            data: str 或 bytes。

        Returns:
            (bytes, int): 编码后的数据与长度。
        """
        if data is None:
            return c_char_p(), 0
        if isinstance(data, str):
            encoded = data.encode("utf-8")
        elif isinstance(data, bytes):
            encoded = data
        else:
            encoded = str(data).encode("utf-8")
        return ctypes.create_string_buffer(encoded, len(encoded)), len(encoded)