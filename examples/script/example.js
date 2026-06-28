// ============================================================
// example.js — D1 Script 扩展示例
// 覆盖所有 VM API：publish / request / call / reply / 日志 / 缓存 / 数据库 / 临时存储
// ============================================================

function main(input) {
    // input = { method: "xxx", params: { ... } }
    d1.info("收到消息: " + input.method);

    switch (input.method) {
        case "sensor.alert":
            return handleSensorAlert(input.params);

        case "user.login":
            return handleUserLogin(input.params);

        case "data.sync":
            return handleDataSync(input.params);

        case "cache.ops":
            return handleCacheOps(input.params);

        default:
            return { result: { echo: true, method: input.method } };
    }
}

// ── 示例 1: publish 单向消息 ──
function handleSensorAlert(params) {
    var temp = params.temperature || 0;

    if (temp > 30) {
        d1.warn("高温报警: " + temp + "°C");

        // d1.publish(connName, method, [data])
        var err = d1.publish("mqtt", "alert.temp", { temperature: temp, level: "high" });
        if (err) {
            d1.error("publish 失败: " + err);
        }
    }

    return { result: { published: true, temperature: temp } };
}

// ── 示例 2: call 同步调用 ──
function handleUserLogin(params) {
    var token = params.token || "";

    // d1.call(kind, target, method, [data], [timeout])
    var resp = d1.call("service", "auth.service", "validate_token", { token: token }, 5);

    if (resp.error) {
        d1.warn("token 验证失败: " + JSON.stringify(resp.error));
        return { error: { code: -1, message: "auth failed" } };
    }

    d1.info("用户登录成功: " + JSON.stringify(resp.result));
    return { result: { login: true, user: resp.result } };
}

// ── 示例 3: reply 回复外部请求 ──
function handleDataSync(params) {
    var data = params.data;

    // 处理完成后回复
    var processed = transform(data);

    // d1.reply(data)
    var err = d1.reply({ result: "synced", count: processed.length });
    if (err) {
        d1.error("reply 失败: " + err);
    }

    return { result: { synced: true, count: processed.length } };
}

// ── 示例 4: 缓存操作 ──
function handleCacheOps(params) {
    var key = params.key;

    // d1.cch_set(key, value, ttl)
    var err = d1.cch_set("user:" + key, { name: "test", ts: Date.now() }, 3600);
    if (err) {
        d1.error("缓存写入失败: " + err);
    }

    // d1.cch_get(key)
    var cached = d1.cch_get("user:" + key);
    d1.info("缓存读取: " + JSON.stringify(cached));

    return { result: { cached: cached } };
}

// ── 辅助函数 ──
function transform(data) {
    return [1, 2, 3];
}