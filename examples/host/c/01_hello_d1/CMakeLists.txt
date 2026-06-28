# D1 C 语言示例 —— 01_hello_d1 构建脚本
# 本文件定义 C 示例程序的 CMake 构建规则。
#
# 前置条件：
#   请先将 libd1.so (Linux) / libd1.dylib (macOS) 以及 d1.h 头文件
#   下载至 ${CMAKE_SOURCE_DIR}/../../../deps/ 目录下。
#   也可从 SDK 路径引用头文件。

cmake_minimum_required(VERSION 3.16)
project(hello_d1_c LANGUAGES C)

# C 标准
set(CMAKE_C_STANDARD 11)
set(CMAKE_C_STANDARD_REQUIRED ON)

# ── D1 动态库路径配置 ──
# 从示例目录向上三级到达仓库根目录，然后进入 deps/
set(D1_DEPS_DIR "${CMAKE_SOURCE_DIR}/../../../deps")

# 头文件搜索路径（优先 deps，其次 SDK）
include_directories("${D1_DEPS_DIR}")
include_directories("${CMAKE_SOURCE_DIR}/../../../lang/c")

# 链接目录
link_directories("${D1_DEPS_DIR}")

# ── 源文件 ──
set(SOURCES main.c)

# ── 可执行目标 ──
add_executable(hello_d1_c ${SOURCES})

# 链接 D1 动态库
target_link_libraries(hello_d1_c PRIVATE d1)

# ── 编译选项 ──
if(CMAKE_C_COMPILER_ID MATCHES "GNU|Clang")
    target_compile_options(hello_d1_c PRIVATE -Wall -Wextra -O2)
endif()