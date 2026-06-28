// ============================================================
// data_transform.js — 数据转换管道
// 演示 call(script) 内部调用 + 数据库操作
// ============================================================

function main(input) {
    // input = { method: "xxx", params: { ... } }
    d1.info("data_transform 收到: " + input.method);

    switch (input.method) {
        case "transform.csv_to_json":
            return handleCsvToJson(input.params);

        case "transform.normalize":
            return handleNormalize(input.params);

        case "transform.batch":
            return handleBatch(input.params);

        default:
            return { result: { transformed: false } };
    }
}

// ── 示例 1: CSV → JSON 转换 ──
function handleCsvToJson(params) {
    var csv = params.csv || "";
    var delimiter = params.delimiter || ",";

    var lines = csv.split("\n").filter(function(l) { return l.trim() !== ""; });
    if (lines.length < 2) {
        return { error: { code: -1, message: "CSV has no data rows" } };
    }

    var headers = lines[0].split(delimiter);
    var rows = [];

    for (var i = 1; i < lines.length; i++) {
        var values = lines[i].split(delimiter);
        var row = {};
        for (var j = 0; j < headers.length; j++) {
            row[headers[j].trim()] = (values[j] || "").trim();
        }
        rows.push(row);
    }

    d1.info("转换完成: " + rows.length + " 行");
    return { result: { rows: rows, count: rows.length } };
}

// ── 示例 2: 数据标准化（chain call） ──
function handleNormalize(params) {
    var data = params.data;

    // 第一步：用 d1.call(script) 调用另一个脚本做预处理
    var resp = d1.call("script", "data_transform", "transform.csv_to_json", {
        csv: data,
        delimiter: "|"
    }, 10);

    if (resp.error) {
        d1.error("预处理失败: " + JSON.stringify(resp.error));
        return { error: { code: -1, message: "preprocess failed" } };
    }

    d1.info("预处理完成: " + JSON.stringify(resp.result));

    // 第二步：存入数据库
    var dbResp = d1.db_exec(
        "INSERT INTO transform_log (method, input_len, output_count, created_at) VALUES (?, ?, ?, datetime('now'))",
        ["transform.normalize", (data || "").length, resp.result.count]
    );

    if (dbResp.error) {
        d1.error("数据库写入失败: " + JSON.stringify(dbResp.error));
    }

    return { result: { normalized: true, count: resp.result.count } };
}

// ── 示例 3: 批量处理 ──
function handleBatch(params) {
    var items = params.items || [];
    var results = [];

    for (var i = 0; i < items.length; i++) {
        // d1.call(script) 调用自身做逐条处理
        var resp = d1.call("script", "data_transform", "transform.normalize", {
            data: items[i]
        }, 5);

        if (resp.error) {
            d1.warn("第 " + i + " 条处理失败: " + JSON.stringify(resp.error));
            results.push({ index: i, error: resp.error });
        } else {
            results.push({ index: i, result: resp.result });
        }
    }

    d1.info("批量处理完成: " + results.length + " 条");
    return { result: { total: items.length, processed: results.length, results: results } };
}