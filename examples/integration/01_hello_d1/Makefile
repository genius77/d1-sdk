# D1 Hello World 示例 (Go) — Makefile
#
# 使用前请确保 deps/ 目录中有:
#   - d1.h（头文件）
#   - libd1.so（Linux）/ libd1.dylib（macOS）/ d1.dll（Windows）
#
# 也可以手动设置环境变量:
#   export CGO_CFLAGS="-I/path/to/deps"
#   export CGO_LDFLAGS="-L/path/to/deps -ld1"
#
# 如果使用 Makefile 中的安装目标，会自动放入 deps/ 目录。

.PHONY: build run clean info setup-deps

# 应用程序名称
APP_NAME := hello_d1

# 默认目标
build:
	CGO_ENABLED=1 go build -o $(APP_NAME) .

# 构建并运行
run: build
	./$(APP_NAME)

# 清理构建产物
clean:
	rm -f $(APP_NAME)

# 显示编译信息和帮助
info:
	@echo "===== D1 Hello World (Go) 编译信息 ====="
	@echo "应用名称: $(APP_NAME)"
	@echo "Go 版本: $$(go version)"
	@echo "CGO 启用: $$(go env CGO_ENABLED)"
	@echo ""
	@echo "当前 CGO 配置:"
	@echo "  CGO_CFLAGS  = $$(go env CGO_CFLAGS)"
	@echo "  CGO_LDFLAGS = $$(go env CGO_LDFLAGS)"
	@echo ""
	@echo "deps/ 目录内容:"
	@ls -la ../../../deps/ 2>/dev/null || echo "  (deps/ 目录不存在或为空)"
	@echo ""
	@echo "可用目标:"
	@echo "  build        编译应用程序"
	@echo "  run          编译并运行"
	@echo "  clean        清理构建产物"
	@echo "  info         显示此帮助信息"
	@echo "  setup-deps   创建 deps/ 目录（需手动放入库文件）"

# 确保 deps/ 目录存在
setup-deps:
	@mkdir -p ../../../deps
	@echo "deps/ 目录已创建: $$(cd ../../../deps && pwd)"
	@echo "请将以下文件放入该目录:"
	@echo "  - d1.h"
	@echo "  - libd1.so   (Linux)"
	@echo "  - libd1.dylib (macOS)"
	@echo "  - d1.dll      (Windows)"