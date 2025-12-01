#!/bin/bash

#======================================
# Y-UI 构建脚本
#======================================

set -e

VERSION=${1:-"1.3.0"}
BUILD_DIR="build"
DIST_DIR="dist"

# 颜色
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Y-UI 构建脚本 v${VERSION}${NC}"
echo -e "${GREEN}========================================${NC}"

# 清理
rm -rf $BUILD_DIR $DIST_DIR
mkdir -p $BUILD_DIR $DIST_DIR

# 构建前端
build_frontend() {
    echo -e "${CYAN}构建前端...${NC}"
    cd frontend
    npm install
    npm run build
    cd ..
    cp -r frontend/dist $BUILD_DIR/
}

# 构建后端
build_backend() {
    local os=$1
    local arch=$2
    local output="y-ui-server"

    if [[ "$os" == "windows" ]]; then
        output="y-ui-server.exe"
    fi

    echo -e "${CYAN}构建后端: ${os}-${arch}${NC}"

    cd backend
    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
        -ldflags="-s -w -X main.Version=${VERSION}" \
        -o "../$BUILD_DIR/$output" \
        ./cmd/y-ui/
    cd ..
}

# 打包
package() {
    local os=$1
    local arch=$2
    local name="y-ui-${os}-${arch}"

    echo -e "${CYAN}打包: ${name}${NC}"

    mkdir -p "$BUILD_DIR/$name"

    # 复制文件
    if [[ "$os" == "windows" ]]; then
        cp "$BUILD_DIR/y-ui-server.exe" "$BUILD_DIR/$name/"
    else
        cp "$BUILD_DIR/y-ui-server" "$BUILD_DIR/$name/"
    fi

    cp -r "$BUILD_DIR/dist" "$BUILD_DIR/$name/"
    cp backend/config.example.yaml "$BUILD_DIR/$name/config.yaml"

    # 打包
    cd $BUILD_DIR
    if [[ "$os" == "windows" ]]; then
        zip -r "../$DIST_DIR/${name}.zip" "$name"
    else
        tar -czf "../$DIST_DIR/${name}.tar.gz" "$name"
    fi
    cd ..

    rm -rf "$BUILD_DIR/$name"
}

# 主构建
main() {
    # 构建前端
    if [[ -d "frontend" ]]; then
        build_frontend
    else
        echo -e "${CYAN}跳过前端构建（目录不存在）${NC}"
        mkdir -p $BUILD_DIR/dist
    fi

    # 构建多平台
    platforms=(
        "linux:amd64"
        "linux:arm64"
        "linux:arm"
        "darwin:amd64"
        "darwin:arm64"
        "windows:amd64"
    )

    for platform in "${platforms[@]}"; do
        os="${platform%%:*}"
        arch="${platform##*:}"

        build_backend "$os" "$arch"
        package "$os" "$arch"

        # 清理
        rm -f "$BUILD_DIR/y-ui-server" "$BUILD_DIR/y-ui-server.exe"
    done

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  构建完成！${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    ls -la $DIST_DIR/
}

main
