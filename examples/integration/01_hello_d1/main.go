// 示例: D1 Hello World (Go 直接集成)
//
// 本示例演示 Go 项目直接通过 import "d1" 集成 D1 运行时:
//   1. 获取版本信息
//   2. 设置生命周期钩子（OnInit/OnStart/OnStop）
//   3. 设置默认消息处理器（OnRequest，演示 API：Publish、Call、CacheSet/CacheGet、DBQuery、Set/Get、Info/Warn）
//   4. 初始化 D1 → 启动 D1 → 阻塞等待退出（Ctrl+C）
//
// 编译运行: go build && ./01_hello_d1
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"d1"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("===== D1 Hello World (Go 直接集成) =====")

	// 1. 创建 D1 运行时实例
	d := d1.New()

	// 输出版本信息
	log.Printf("D1 version: %s", d1.Version)

	// 2. 设置生命周期钩子
	d.OnInit(func(ctx *d1.Context) error {
		ctx.Info("宿主预初始化：校验配置、预热连接...")
		return nil
	})

	d.OnStart(func(ctx *d1.Context) error {
		ctx.Info("D1 已启动，注册服务发现...")
		return nil
	})

	d.OnStop(func(ctx *d1.Context) error {
		ctx.Info("收到停止信号，宿主优先清理...")
		return nil
	})

	// 3. 设置默认消息处理器
	//    演示所有 D1 API 的调用方式
	d.OnRequest(func(ctx *d1.Context, req *d1.Request) (*d1.Response, error) {
		log.Printf("[handler] taskID=%d method=%s params=%s", ctx.TaskID(), req.Method, string(req.Params))

		// ── 1. Publish — 发送单向消息（无回复） ──
		pubData, _ := json.Marshal(map[string]interface{}{"temp": 25.5, "unit": "celsius"})
		if err := ctx.Publish("mqtt_client", "sensor.data", pubData); err != nil {
			ctx.Warn("Publish failed: " + err.Error())
		} else {
			ctx.Info("Publish OK")
		}

		// ── 2. CacheSet — 写入缓存 ──
		cacheVal, _ := json.Marshal(map[string]interface{}{"name": "Alice", "role": "admin"})
		if err := ctx.CacheSet("user:42", cacheVal, 3600); err != nil {
			ctx.Warn("CacheSet failed: " + err.Error())
		}

		// ── 3. CacheGet — 读取缓存 ──
		if cached, err := ctx.CacheGet("user:42"); err == nil && cached != nil {
			ctx.Info("CacheGet: " + string(cached))
		}

		// ── 4. Call — 同步调用（阻塞等待结果） ──
		ctx.Info("Calling script:api_handler.get_user")
		callParams, _ := json.Marshal(map[string]interface{}{"id": 123})
		resp, err := ctx.Call("script", "api_handler", "get_user", callParams, 5*time.Second)
		if err != nil {
			ctx.Warn("Call failed: " + err.Error())
		} else if resp.Error != nil {
			ctx.Warn("Call error: " + resp.Error.Message)
		} else {
			ctx.Info("Call result: " + string(resp.Result))
		}

		// ── 5. DBQuery — 数据库查询 ──
		rows, err := ctx.DBQuery("SELECT * FROM users LIMIT 1")
		if err != nil {
			ctx.Warn("DBQuery failed: " + err.Error())
		} else {
			data, _ := json.Marshal(rows)
			ctx.Info("DBQuery: " + string(data))
		}

		// ── 6. Set/Get — 临时存储 ──
		ctx.Set("temp_key", "hello from handler")
		val := ctx.Get("temp_key")
		ctx.Info(fmt.Sprintf("Get('temp_key'): %v", val))

		// ── 7. 返回响应 ──
		result, _ := json.Marshal(map[string]interface{}{
			"status": "ok",
			"msg":    "hello from Go handler",
		})
		return d1.NewResponse(result), nil
	})

	// 4. 初始化 D1（configPath 为空使用默认配置）
	log.Println("Initializing D1...")
	if err := d.Init(""); err != nil {
		log.Fatalf("D1 Init failed: %v", err)
	}
	log.Println("D1 initialized")

	// 5. 启动 D1
	log.Println("Starting D1...")
	if err := d.Start(); err != nil {
		log.Fatalf("D1 Start failed: %v", err)
	}
	log.Println("D1 running (press Ctrl+C to exit)")

	// 6. 阻塞等待退出
	if err := d.WaitStop(); err != nil {
		log.Printf("D1 WaitStop error: %v", err)
	}
	log.Println("D1 exited")
}