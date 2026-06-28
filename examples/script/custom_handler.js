// ============================================================
// custom_handler.js — 自定义业务处理器
// 演示 request 异步调用 + 重试逻辑 + 临时存储
// ============================================================

function main(input) {
    // input = { method: "xxx", params: { ... } }
    d1.info("custom_handler 收到: " + input.method);

    switch (input.method) {
        case "order.create":
            return handleOrderCreate(input.params);

        case "order.status":
            return handleOrderStatus(input.params);

        default:
            return { result: { handled: false, method: input.method } };
    }
}

// ── 示例 1: request 异步调用 + 临时存储 ──
function handleOrderCreate(params) {
    var orderId = params.order_id;
    var amount = params.amount;

    // 存储订单信息到临时存储
    d1.set("order:" + orderId, { id: orderId, amount: amount, status: "pending" });

    // d1.request(connName, method, [data], [timeout], callback)
    // 异步调用支付服务
    var err = d1.request("payment", "process_payment", {
        order_id: orderId,
        amount: amount
    }, 30, function(resp) {
        if (resp.error) {
            d1.error("支付失败 order=" + orderId + ": " + JSON.stringify(resp.error));
            d1.set("order:" + orderId, { id: orderId, status: "failed" });
            return;
        }

        d1.info("支付成功 order=" + orderId);
        d1.set("order:" + orderId, { id: orderId, status: "paid" });

        // 支付成功后通知物流
        d1.publish("mqtt", "logistics.notify", { order_id: orderId, action: "prepare" });
    });

    if (err) {
        d1.error("request 发起失败: " + err);
        return { error: { code: -1, message: "request failed" } };
    }

    return { result: { order_id: orderId, status: "processing" } };
}

// ── 示例 2: 查询 + 缓存 ──
function handleOrderStatus(params) {
    var orderId = params.order_id;

    // 先查临时存储
    var order = d1.get("order:" + orderId);
    if (order) {
        return { result: order };
    }

    // 查缓存
    var cached = d1.cch_get("order_status:" + orderId);
    if (cached) {
        return { result: cached };
    }

    // 查数据库
    var resp = d1.db_query("SELECT status, amount FROM orders WHERE id = ?", [orderId]);
    if (resp.error) {
        d1.error("数据库查询失败: " + JSON.stringify(resp.error));
        return { error: { code: -1, message: "query failed" } };
    }

    d1.debug("查询结果: " + JSON.stringify(resp.result));
    return { result: resp.result };
}