# D1 C++ 示例 —— 01_hello_d1 构建脚本
# 本文件定义 C++ 示例程序的 CMake 构建规则。
#
# 前置条件：
#   请先将 libd1.so (Linux) / libd1.dylib (macOS) 以及 d1.h 头文件
#   下载至 ${CMAKE_SOURCE_DIR}/../../../deps/ 目录下。

cmake_minimum_required(VERSION 3.16)
project(hello_d1_cpp LANGUAGES CXX)

# C++ 标准
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ── D1 动态库路径配置 ──
# 从示例目录向上三级到达仓库根目录，然后进入 deps/
set(D1_DEPS_DIR "${CMAKE_SOURCE_DIR}/../../../deps")

# 头文件搜索路径（优先 deps，其次 SDK）
include_directories("${D1_DEPS_DIR}")
include_directories("${CMAKE_SOURCE_DIR}/../../../lang/cpp")
include_directories("${CMAKE_SOURCE_DIR}/../../../lang/c")

# 链接目录
link_directories("${D1_DEPS_DIR}")

# ── 源文件 ──
set(SOURCES main.cpp)

# ── 可执行目标 ──
add_executable(hello_d1_cpp ${SOURCES})

# 链接 D1 动态库
target_link_libraries(hello_d1_cpp PRIVATE d1)

# 链接线程库（Linux 需要 -pthread）
if(UNIX AND NOT APPLE)
    target_link_libraries(hello_d1_cpp PRIVATE pthread)
endif()

# ── 编译选项 ──
if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    target_compile_options(hello_d1_cpp PRIVATE -Wall -Wextra -O2)
endif()