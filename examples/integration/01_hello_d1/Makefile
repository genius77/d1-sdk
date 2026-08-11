# D1 Hello World 示例 (Go 直接集成) — Makefile
#
# 本示例通过 import "d1" 直接集成 D1 运行时，无需 CGO 和动态库。
# 前提: go.mod 中的 replace 指令已指向本地 D1 源码目录。

.PHONY: build run clean info

# 应用程序名称
APP_NAME := hello_d1

# 默认目标
build:
	go build -o $(APP_NAME) .

# 构建并运行
run: build
	./$(APP_NAME)

# 清理构建产物
clean:
	rm -f $(APP_NAME)

# 显示编译信息和帮助
info:
	@echo "===== D1 Hello World (Go 直接集成) 编译信息 ====="
	@echo "应用名称: $(APP_NAME)"
	@echo "Go 版本: $$(go version)"
	@echo ""
	@echo "集成方式: import \"d1\"（无需 CGO 和动态库）"
	@echo "D1 源码位置: go.mod 中 replace 指令指定"
	@echo ""
	@echo "可用目标:"
	@echo "  build   编译应用程序"
	@echo "  run     编译并运行"
	@echo "  clean   清理构建产物"
	@echo "  info    显示此帮助信息"
