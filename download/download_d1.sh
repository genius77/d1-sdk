#!/bin/bash
# ============================================================
# D1 动态库下载脚本 (Unix/Linux/macOS)
# 用法: ./download_d1.sh [version]
#   - 不指定版本: 下载最新版本
#   - ./download_d1.sh v1.1.0 : 下载指定版本
# ============================================================

set -euo pipefail

D1_REPO="genius77/D1"
DEPS_DIR="deps"

# 颜色输出
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

# ─── 平台检测 ──────────────────────────────────────────────
detect_platform() {
    local os arch
    case "$(uname -s)" in
        Linux)  os="linux" ;;
        Darwin) os="macos" ;;
        *)      echo -e "${RED}✗ 不支持的操作系统: $(uname -s)${NC}"; exit 1 ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch="x64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo -e "${RED}✗ 不支持的架构: $(uname -m)${NC}"; exit 1 ;;
    esac
    PLATFORM="${os}-${arch}"
}

# ─── 版本解析 ──────────────────────────────────────────────
resolve_version() {
    if [ "${1:-latest}" = "latest" ]; then
        echo -e "${CYAN}正在获取最新版本...${NC}"
        VERSION=$(curl -sL "https://api.github.com/repos/${D1_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        [ -z "$VERSION" ] && { echo -e "${RED}✗ 无法获取版本${NC}"; exit 1; }
    else
        VERSION="$1"
    fi
    echo -e "${GREEN}✓ 版本: ${VERSION}${NC}"
}

# ─── 下载并解压 ────────────────────────────────────────────
download() {
    local asset_name="libd1-${PLATFORM}-${VERSION}"
    local ext="tar.gz"

    echo -e "${CYAN}正在查找 ${asset_name}...${NC}"
    local json=$(curl -sL "https://api.github.com/repos/${D1_REPO}/releases/tags/${VERSION}")
    local url=$(echo "$json" | grep '"browser_download_url"' | grep "${asset_name}" | sed -E 's/.*"([^"]+)".*/\1/' | head -1)

    [ -z "$url" ] && { echo -e "${RED}✗ 未找到该平台产物${NC}"; exit 1; }

    mkdir -p "${DEPS_DIR}"
    echo -e "${CYAN}下载中...${NC}"
    curl -sL -o "${DEPS_DIR}/${asset_name}.${ext}" "$url"
    tar -xzf "${DEPS_DIR}/${asset_name}.${ext}" -C "${DEPS_DIR}/"
    rm "${DEPS_DIR}/${asset_name}.${ext}"
}

# ─── 验证 ──────────────────────────────────────────────────
verify() {
    echo ""
    # 确保 d1.h 在 deps/ 根目录
    if [ -f "${DEPS_DIR}/d1.h" ]; then
        echo -e "${GREEN}✓ d1.h${NC}"
    elif [ -f "${DEPS_DIR}/include/d1.h" ]; then
        cp "${DEPS_DIR}/include/d1.h" "${DEPS_DIR}/d1.h"
        echo -e "${GREEN}✓ d1.h (已复制)${NC}"
    fi
    local lib=$(find "${DEPS_DIR}" -maxdepth 1 \( -name "libd1.so" -o -name "libd1*.dylib" \) | head -1)
    [ -n "$lib" ] && echo -e "${GREEN}✓ $(basename "$lib")${NC}"
}

# ─── 主流程 ────────────────────────────────────────────────
echo -e "\n${GREEN}╔══════════════════════════════════╗${NC}"
echo -e "${GREEN}║  D1 动态库下载工具              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════╝${NC}\n"

detect_platform
resolve_version "${1:-latest}"
download
verify

echo -e "\n${GREEN}✅ D1 下载完成，文件位于 ./${DEPS_DIR}/${NC}"
echo -e "  → 接下来: cd examples/host/<语言>/01_hello_d1 && 开始运行"