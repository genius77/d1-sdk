// Package d1 提供 D1 动态库的 Go SDK 封装
//
// D1 SDK Go 封装 | 对应 D1 动态库版本: >= v1.5.0
//
// 使用前请将 libd1.so / libd1.dylib / d1.dll 及 d1.h 放入 deps/ 目录，
// 或通过 CGO_CFLAGS / CGO_LDFLAGS 环境变量指定头文件与库文件路径。
//
// 快速开始:
//
//	import d1 "github.com/genius77/d1-sdk/lang/go"
//
//	d := d1.Default()
//	if err := d.Init("config.json"); err != nil { ... }
//	if err := d.Start(); err != nil { ... }
//	d.SetOnRequest(func(taskID uint64, method string, params []byte) ([]byte, error) {
//	    // 处理消息
//	    return []byte("ok"), nil
//	})
//	d.WaitStop()
package d1

/*
#cgo CFLAGS: -I${SRCDIR}/../../deps
#cgo LDFLAGS: -L${SRCDIR}/../../deps -ld1
#cgo linux LDFLAGS: -Wl,-rpath,${SRCDIR}/../../deps
#cgo darwin LDFLAGS: -Wl,-rpath,${SRCDIR}/../../deps

#include "d1.h"
#include <stdlib.h>

// --- 回调函数类型定义 ---
// 默认消息处理回调（SetOnRequest）
typedef int (*d1_handler_cb_t)(uint64_t, const char*, const char*, int, char**, int*, char**);
// 异步请求回调（Request）
typedef void (*d1_request_cb_t)(uint64_t, const char*, int, const char*);
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

// ---------------------------------------------------------------------------
// 回调类型定义（面向 Go 用户）
// ---------------------------------------------------------------------------

// HandlerFunc 默认消息处理器回调签名。
//
// 当 D1 收到未匹配到特定处理器的消息时回调此函数。
//   - taskID:  任务标识
//   - method:  消息名称（UTF-8 字符串）
//   - params:  方法参数（原始字节，可能为 nil）
//
// 返回值:
//   - []byte: 响应载荷（nil 表示无需返回载荷）
//   - error:  处理过程中出现的错误（将被传回调用方）
type HandlerFunc func(taskID uint64, method string, params []byte) ([]byte, error)

// RequestCallback D1_Request 异步请求的回调函数签名。
//
// 当远程目标返回响应或超时时被调用。
//   - taskID:  原始请求的任务标识
//   - params: 响应参数（可能为 nil）
//   - err:     错误信息（成功时为 nil）
type RequestCallback func(taskID uint64, params []byte, err error)

// ---------------------------------------------------------------------------
// 内部状态管理
// ---------------------------------------------------------------------------

// D1 表示 D1 运行时实例。
//
// 本 SDK 采用全局单例模式，推荐使用 Default() 获取实例。
// 多个 goroutine 并发调用方法是安全的。
type D1 struct {
	mu          sync.Mutex
	handler     HandlerFunc
	initialized bool
	started     bool
}

// 全局单例
var _instance = &D1{}

// 供 C 回调使用的 Go 函数指针（全局，有且仅有一个处理器）
var _defaultHandler HandlerFunc

// Request 回调注册表 — 以 taskID 为键
var (
	_requestCallbacks   = make(map[uint64]RequestCallback)
	_requestCallbacksMu sync.Mutex
)

// Default 返回全局 D1 单例实例。
func Default() *D1 {
	return _instance
}

// ---------------------------------------------------------------------------
// 导出给 C 的回调桥接函数
// ---------------------------------------------------------------------------

//export goDefaultHandler
func goDefaultHandler(
	taskID C.uint64_t,
	method *C.char,
	params *C.char,
	payloadLen C.int,
	outResult **C.char,
	outLen *C.int,
	outError **C.char,
) C.int {
	handler := _defaultHandler
	if handler == nil {
		errStr := C.CString("no default handler set (SetOnRequest)")
		*outError = errStr
		return -1
	}

	// 将 C 数据转换为 Go 类型
	goMethod := C.GoString(method)
	var goParams []byte
	if params != nil && payloadLen > 0 {
		goParams = C.GoBytes(unsafe.Pointer(params), payloadLen)
	}

	resp, err := handler(uint64(taskID), goMethod, goParams)
	if err != nil {
		cErr := C.CString(err.Error())
		*outError = cErr
		return -1
	}

	if len(resp) > 0 {
		*outResult = (*C.char)(C.CBytes(resp))
		*outLen = C.int(len(resp))
	}
	return 0
}

//export goRequestCallback
func goRequestCallback(
	taskID C.uint64_t,
	payload *C.char,
	payloadLen C.int,
	errStr *C.char,
) {
	id := uint64(taskID)

	_requestCallbacksMu.Lock()
	cb := _requestCallbacks[id]
	delete(_requestCallbacks, id)
	_requestCallbacksMu.Unlock()

	if cb == nil {
		return
	}

	var goPayload []byte
	if payload != nil && payloadLen > 0 {
		goPayload = C.GoBytes(unsafe.Pointer(payload), payloadLen)
	}

	var goErr error
	if errStr != nil {
		goErr = errors.New(C.GoString(errStr))
	}

	cb(id, goPayload, goErr)
}

// ---------------------------------------------------------------------------
// D1 方法 — 封装全部 17 个 C API
// ---------------------------------------------------------------------------

// Version 获取 D1 动态库的版本号字符串。
//
// 示例:
//
//	v := d1.Default().Version()
//	fmt.Println("D1 版本:", v)
func (d *D1) Version() string {
	return C.GoString(C.D1_Version())
}

// Init 初始化 D1 运行时。
//
// configPath: 配置文件路径；传空字符串 "" 表示使用内置默认配置。
// 应在 Start 之前调用，且只能调用一次。
//
// 示例:
//
//	err := d1.Default().Init("config.yaml")
func (d *D1) Init(configPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.initialized {
		return errors.New("D1 already initialized")
	}

	var cConfig *C.char
	if configPath != "" {
		cConfig = C.CString(configPath)
		defer C.free(unsafe.Pointer(cConfig))
	}

	ret := C.D1_Init(cConfig)
	if ret != 0 {
		return fmt.Errorf("D1_Init failed, error code: %d", ret)
	}
	d.initialized = true
	return nil
}

// Start 启动 D1 运行时。
//
// 调用后将进入事件循环，应在 Init 之后、WaitStop 之前调用。
//
// 示例:
//
//	if err := d1.Default().Start(); err != nil { ... }
func (d *D1) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.initialized {
		return errors.New("D1 not initialized, call Init() first")
	}
	if d.started {
		return errors.New("D1 already started")
	}

	ret := C.D1_Start()
	if ret != 0 {
		return fmt.Errorf("D1_Start failed, error code: %d", int(ret))
	}
	d.started = true
	return nil
}

// Stop 停止 D1 运行时并释放相关资源。
//
// 通常推荐使用 WaitStop() 替代手动调用 Stop + Wait。
// 重复调用是安全的。
//
// 示例:
//
//	d1.Default().Stop()
func (d *D1) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.started {
		return nil
	}

	ret := C.D1_Stop()
	d.started = false
	if ret != 0 {
		return fmt.Errorf("D1_Stop failed, error code: %d", int(ret))
	}
	return nil
}

// WaitStop 阻塞等待退出信号，收到信号后自动停止 D1。
//
// 推荐用法: Init() → Start() → WaitStop() → 进程退出
//
// D1_WaitStop 内部监听 SIGINT/SIGTERM (Ctrl+C)，
// 收到信号后自动调用 D1_Stop()，无需用户手动处理信号。
//
// 示例:
//
//	d1.Default().WaitStop()
func (d *D1) WaitStop() int {
	return int(C.D1_WaitStop())
}

// SetOnRequest 设置默认消息处理器。
//
// 当 D1 收到未匹配到特定路由的消息时，将调用此处理器。
// 只需设置一次，通常在 Start 之前调用。
//
// 示例:
//
//	d1.Default().SetOnRequest(func(taskID uint64, method string, params []byte) ([]byte, error) {
//	    log.Printf("收到消息: %s", method)
//	    return []byte("ack"), nil
//	})
func (d *D1) SetOnRequest(handler HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handler = handler
	_defaultHandler = handler
}

// Publish 发布消息到指定目标（单向，不等待响应）。
//
//   - taskID:  任务标识，用于追踪
//   - target:  目标地址
//   - method:  消息名称
//   - params:  方法参数

// 示例:
//
//	err := d1.Default().Publish(1, "node/node1", "event.hello", `{"msg":"hi"}`)
func (d *D1) Publish(taskID uint64, target, method, params string) error {
	cTarget := C.CString(target)
	defer C.free(unsafe.Pointer(cTarget))
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	ret := C.D1_Publish(C.uint64_t(taskID), cTarget, cMethod, cParams, C.int(len(params)))
	if ret != 0 {
		return fmt.Errorf("D1_Publish failed, error code: %d", int(ret))
	}
	return nil
}

// Call 同步调用远程目标并等待响应。
//
//   - taskID:     任务标识
//   - kind:       调用类型（由 D1 协议定义）
//   - target:     目标地址
//   - method:     消息名称
//   - params:     方法参数
//   - timeoutSec: 超时秒数（<=0 表示使用默认超时）
//
// 返回值:
//   - string: 响应载荷
//   - error:  调用失败或超时
//
// 示例:
//
//	resp, err := d1.Default().Call(1, 0, "node/node1", "rpc.query", `{"sql":"SELECT 1"}`, 10)
func (d *D1) Call(taskID uint64, kind int, target, method, params string, timeoutSec int) (string, error) {
	cTarget := C.CString(target)
	defer C.free(unsafe.Pointer(cTarget))
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	var outResult *C.char
	var outLen C.int
	var outError *C.char

	ret := C.D1_Call(
		C.uint64_t(taskID),
		C.int(kind),
		cTarget,
		cMethod,
		cParams,
		C.int(len(params)),
		C.int(timeoutSec),
		&outResult,
		&outLen,
		&outError,
	)

	// 释放 D1 分配的输出内存（无论成功失败均需释放）
	defer func() {
		if outResult != nil {
			C.D1_Free(unsafe.Pointer(outResult))
		}
		if outError != nil {
			C.D1_Free(unsafe.Pointer(outError))
		}
	}()

	if ret != 0 {
		errMsg := ""
		if outError != nil {
			errMsg = C.GoString(outError)
		}
		return "", fmt.Errorf("D1_Call failed, error code: %d: %s", int(ret), errMsg)
	}

	if outResult == nil {
		return "", nil
	}
	return C.GoStringN(outResult, outLen), nil
}

// Request 异步请求远程目标（通过回调接收响应）。
//
//   - taskID:     任务标识
//   - target:     目标地址
//   - method:     消息名称
//   - params:     方法参数
//   - timeoutSec: 超时秒数
//   - callback:   收到响应或超时时回调
//
// 注意: callback 可能在任意 goroutine 中被调用，注意并发安全。
//
// 示例:
//
//	err := d1.Default().Request(1, "node/node1", "async.query", `{"q":"data"}`, 10,
//	    func(taskID uint64, params []byte, err error) {
//	        if err != nil {
//	            log.Printf("请求失败: %v", err)
//	            return
//	        }
//	        log.Printf("响应: %s", string(params))
//	    })
func (d *D1) Request(taskID uint64, target, method, params string, timeoutSec int, callback RequestCallback) error {
	// 注册回调 —— 在调用 C 函数前完成，避免竞态
	_requestCallbacksMu.Lock()
	_requestCallbacks[taskID] = callback
	_requestCallbacksMu.Unlock()

	cTarget := C.CString(target)
	defer C.free(unsafe.Pointer(cTarget))
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	ret := C.D1_Request(
		C.uint64_t(taskID),
		cTarget,
		cMethod,
		cParams,
		C.int(len(params)),
		C.int(timeoutSec),
		(*C.d1_request_cb_t)(unsafe.Pointer(C.goRequestCallback)),
	)

	if ret != 0 {
		// 清理已注册的回调
		_requestCallbacksMu.Lock()
		delete(_requestCallbacks, taskID)
		_requestCallbacksMu.Unlock()
		return fmt.Errorf("D1_Request failed, error code: %d", int(ret))
	}
	return nil
}

// Reply 回复消息（在 HandlerFunc 内部使用）。
//
//   - taskID:  原始请求的任务标识
//   - method:  回复消息名称
//   - params:  回复参数
//
// 示例:
//
//	err := d1.Default().Reply(taskID, "response.ok", `{"status":"done"}`)
func (d *D1) Reply(taskID uint64, method, params string) error {
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	cParams := C.CString(params)
	defer C.free(unsafe.Pointer(cParams))

	ret := C.D1_Reply(C.uint64_t(taskID), cMethod, cParams, C.int(len(params)))
	if ret != 0 {
		return fmt.Errorf("D1_Reply failed, error code: %d", int(ret))
	}
	return nil
}

// CacheGet 从 D1 缓存中获取键对应的值。
//
//   - taskID: 任务标识
//   - key:    缓存键
//
// 返回值:
//   - string: 缓存值
//   - error:  键不存在或获取失败
//
// 示例:
//
//	val, err := d1.Default().CacheGet(1, "mykey")
func (d *D1) CacheGet(taskID uint64, key string) (string, error) {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var result *C.char
	var resultLen C.int

	ret := C.D1_CacheGet(C.uint64_t(taskID), cKey, &result, &resultLen)

	defer func() {
		if result != nil {
			C.D1_Free(unsafe.Pointer(result))
		}
	}()

	if ret != 0 {
		return "", fmt.Errorf("D1_CacheGet failed, error code: %d", int(ret))
	}

	if result == nil {
		return "", nil
	}
	return C.GoStringN(result, resultLen), nil
}

// CacheSet 向 D1 缓存中设置键值对。
//
//   - taskID:     任务标识
//   - key:        缓存键
//   - value:      缓存值
//   - ttlSeconds: 过期时间（秒），<=0 表示永不过期
//
// 示例:
//
//	err := d1.Default().CacheSet(1, "mykey", "myvalue", 3600)
func (d *D1) CacheSet(taskID uint64, key, value string, ttlSeconds int) error {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	ret := C.D1_CacheSet(C.uint64_t(taskID), cKey, cValue, C.int(len(value)), C.int(ttlSeconds))
	if ret != 0 {
		return fmt.Errorf("D1_CacheSet failed, error code: %d", int(ret))
	}
	return nil
}

// CacheDelete 从 D1 缓存中删除指定键。
//
//   - taskID: 任务标识
//   - key:    缓存键
//
// 示例:
//
//	err := d1.Default().CacheDelete(1, "mykey")
func (d *D1) CacheDelete(taskID uint64, key string) error {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	ret := C.D1_CacheDelete(C.uint64_t(taskID), cKey)
	if ret != 0 {
		return fmt.Errorf("D1_CacheDelete failed, error code: %d", int(ret))
	}
	return nil
}

// DBQuery 执行数据库查询并返回 JSON 格式结果集。
//
//   - taskID: 任务标识
//   - query:  SQL 查询语句
//
// 返回值:
//   - string: JSON 格式的查询结果
//   - error:  查询失败
//
// 示例:
//
//	json, err := d1.Default().DBQuery(1, "SELECT id, name FROM users LIMIT 10")
func (d *D1) DBQuery(taskID uint64, query string) (string, error) {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	var result *C.char
	var resultLen C.int

	ret := C.D1_DBQuery(C.uint64_t(taskID), cQuery, C.int(len(query)), &result, &resultLen)

	defer func() {
		if result != nil {
			C.D1_Free(unsafe.Pointer(result))
		}
	}()

	if ret != 0 {
		return "", fmt.Errorf("D1_DBQuery failed, error code: %d", int(ret))
	}

	if result == nil {
		return "", nil
	}
	return C.GoStringN(result, resultLen), nil
}

// DBExec 执行数据库写操作（INSERT/UPDATE/DELETE 等）。
//
//   - taskID: 任务标识
//   - query:  SQL 写操作语句
//
// 返回值:
//   - int64: 受影响的行数
//   - error: 执行失败
//
// 示例:
//
//	rows, err := d1.Default().DBExec(1, "UPDATE users SET active=1 WHERE id=42")
func (d *D1) DBExec(taskID uint64, query string) (int64, error) {
	cQuery := C.CString(query)
	defer C.free(unsafe.Pointer(cQuery))

	var affectedRows C.int64_t

	ret := C.D1_DBExec(C.uint64_t(taskID), cQuery, C.int(len(query)), &affectedRows)
	if ret != 0 {
		return 0, fmt.Errorf("D1_DBExec failed, error code: %d", int(ret))
	}

	return int64(affectedRows), nil
}

// Set 向 D1 内建键值存储设置键值对。
//
//   - taskID: 任务标识
//   - key:    键
//   - value:  值
//
// 示例:
//
//	err := d1.Default().Set(1, "config.app_name", "myapp")
func (d *D1) Set(taskID uint64, key, value string) error {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))
	cValue := C.CString(value)
	defer C.free(unsafe.Pointer(cValue))

	ret := C.D1_Set(C.uint64_t(taskID), cKey, cValue, C.int(len(value)))
	if ret != 0 {
		return fmt.Errorf("D1_Set failed, error code: %d", int(ret))
	}
	return nil
}

// Get 从 D1 内建键值存储获取键对应的值。
//
//   - taskID: 任务标识
//   - key:    键
//
// 返回值:
//   - string: 值
//   - error:  键不存在或获取失败
//
// 示例:
//
//	val, err := d1.Default().Get(1, "config.app_name")
func (d *D1) Get(taskID uint64, key string) (string, error) {
	cKey := C.CString(key)
	defer C.free(unsafe.Pointer(cKey))

	var result *C.char
	var resultLen C.int

	ret := C.D1_Get(C.uint64_t(taskID), cKey, &result, &resultLen)

	defer func() {
		if result != nil {
			C.D1_Free(unsafe.Pointer(result))
		}
	}()

	if ret != 0 {
		return "", fmt.Errorf("D1_Get failed, error code: %d", int(ret))
	}

	if result == nil {
		return "", nil
	}
	return C.GoStringN(result, resultLen), nil
}

// Free 释放 D1 分配的内存（通常由 SDK 内部自动调用，用户无需手动调用）。
//
// 仅在特殊场景下（如直接使用 cgo 调用 D1 C API）才需要手动释放。
func (d *D1) Free(ptr unsafe.Pointer) {
	if ptr != nil {
		C.D1_Free(ptr)
	}
}