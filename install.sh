#!/bin/bash

#======================================
# Y-UI 一键安装脚本
# https://github.com/CNYuns/yui
#======================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 配置
INSTALL_DIR="/usr/local/y-ui"
SERVICE_NAME="y-ui"
GITHUB_REPO="CNYuns/yui"
YUI_VERSION="${1:-latest}"
XRAY_VERSION="v24.11.30"

# 获取公网 IP
get_public_ip() {
    local ip=""
    # 尝试多个 IP 检测服务
    ip=$(curl -s4 --max-time 5 https://api.ipify.org 2>/dev/null) ||
    ip=$(curl -s4 --max-time 5 https://ifconfig.me 2>/dev/null) ||
    ip=$(curl -s4 --max-time 5 https://ip.sb 2>/dev/null) ||
    ip=$(curl -s4 --max-time 5 https://ipinfo.io/ip 2>/dev/null) ||
    ip=$(hostname -I 2>/dev/null | awk '{print $1}')

    if [[ -z "$ip" || "$ip" == "127.0.0.1" ]]; then
        ip="YOUR_SERVER_IP"
    fi
    echo "$ip"
}

# 检测系统架构
get_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armv7)
            echo "arm"
            ;;
        *)
            echo -e "${RED}不支持的架构: $arch${NC}"
            exit 1
            ;;
    esac
}

# 获取 Xray 架构名
get_xray_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64|amd64)
            echo "64"
            ;;
        aarch64|arm64)
            echo "arm64-v8a"
            ;;
        armv7l|armv7)
            echo "arm32-v7a"
            ;;
        *)
            echo "64"
            ;;
    esac
}

# 检测系统
check_system() {
    if [[ ! -f /etc/os-release ]]; then
        echo -e "${RED}无法检测操作系统${NC}"
        exit 1
    fi
    source /etc/os-release
    echo -e "${GREEN}系统: $PRETTY_NAME${NC}"
}

# 检查 root 权限
check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}请使用 root 用户运行此脚本${NC}"
        exit 1
    fi
}

# 安装依赖
install_deps() {
    echo -e "${CYAN}安装依赖...${NC}"
    if command -v apt-get &> /dev/null; then
        apt-get update -qq
        apt-get install -y -qq curl wget unzip tar
    elif command -v yum &> /dev/null; then
        yum install -y -q curl wget unzip tar
    elif command -v dnf &> /dev/null; then
        dnf install -y -q curl wget unzip tar
    elif command -v pacman &> /dev/null; then
        pacman -Sy --noconfirm curl wget unzip tar
    elif command -v apk &> /dev/null; then
        apk add --no-cache curl wget unzip tar
    fi
}

# 安装 Xray-core
install_xray() {
    echo -e "${CYAN}安装 Xray-core...${NC}"

    local xray_arch=$(get_xray_arch)
    local xray_url="https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/Xray-linux-${xray_arch}.zip"
    local temp_file="/tmp/xray.zip"

    # 创建 Xray 目录
    mkdir -p /usr/local/xray
    mkdir -p /etc/xray
    mkdir -p /var/log/xray

    # 下载 Xray
    echo -e "${CYAN}下载 Xray: $xray_url${NC}"
    if ! curl -L -o "$temp_file" "$xray_url" --progress-bar; then
        echo -e "${RED}下载 Xray 失败${NC}"
        exit 1
    fi

    # 解压
    unzip -o "$temp_file" -d /usr/local/xray
    rm -f "$temp_file"

    # 设置权限
    chmod +x /usr/local/xray/xray

    # 创建符号链接
    ln -sf /usr/local/xray/xray /usr/local/bin/xray

    # 验证安装
    if /usr/local/xray/xray version &>/dev/null; then
        local ver=$(/usr/local/xray/xray version | head -1)
        echo -e "${GREEN}Xray 安装成功: $ver${NC}"
    else
        echo -e "${RED}Xray 安装失败${NC}"
        exit 1
    fi

    # 创建默认配置
    if [[ ! -f /etc/xray/config.json ]]; then
        cat > /etc/xray/config.json << 'XRAYEOF'
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [],
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom"
    },
    {
      "tag": "blocked",
      "protocol": "blackhole"
    }
  ]
}
XRAYEOF
    fi
}

# 获取最新版本
get_latest_version() {
    local latest=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null | grep -m1 '"tag_name"' | cut -d'"' -f4)
    if [[ -z "$latest" || ! "$latest" =~ ^v[0-9] ]]; then
        echo "v1.2.9"
    else
        echo "$latest"
    fi
}

# 下载文件
download() {
    local url=$1
    local output=$2
    echo -e "${CYAN}下载: $url${NC}"
    if command -v curl &> /dev/null; then
        curl -L -o "$output" "$url" --progress-bar
    elif command -v wget &> /dev/null; then
        wget -O "$output" "$url" --show-progress
    else
        echo -e "${RED}请安装 curl 或 wget${NC}"
        exit 1
    fi
}

# 安装 Y-UI
install_yui() {
    local arch=$(get_arch)
    local version=$YUI_VERSION

    if [[ "$version" == "latest" ]]; then
        version=$(get_latest_version)
    fi

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 安装程序 ${version}${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""

    # 创建目录
    mkdir -p "$INSTALL_DIR"
    mkdir -p /var/log/y-ui

    # 下载 Y-UI
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/y-ui-linux-${arch}.tar.gz"
    local temp_file="/tmp/y-ui-linux-${arch}.tar.gz"

    download "$download_url" "$temp_file"

    # 检查下载是否成功
    if [[ ! -f "$temp_file" ]] || [[ $(stat -c%s "$temp_file" 2>/dev/null || stat -f%z "$temp_file" 2>/dev/null) -lt 1000 ]]; then
        echo -e "${RED}下载失败或文件过小${NC}"
        exit 1
    fi

    # 停止旧服务
    if systemctl is-active --quiet $SERVICE_NAME 2>/dev/null; then
        echo -e "${YELLOW}停止旧服务...${NC}"
        systemctl stop $SERVICE_NAME
    fi

    # 解压安装
    echo -e "${CYAN}解压安装包...${NC}"
    tar -xzf "$temp_file" -C "$INSTALL_DIR"
    rm -f "$temp_file"

    # 设置权限
    chmod +x "$INSTALL_DIR/y-ui-server"
    chmod 600 "$INSTALL_DIR/config.yaml" 2>/dev/null || true

    # 创建默认配置（如果不存在）
    create_default_config

    # 安装 y-ui 管理命令
    install_cli

    # 安装 systemd 服务
    install_service

    # 获取公网 IP
    local public_ip=$(get_public_ip)

    # 获取配置的端口
    local port=$(grep -E "^\s*addr:" "$INSTALL_DIR/config.yaml" 2>/dev/null | grep -oE '[0-9]+' || echo "8080")

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 安装完成!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "管理命令: ${CYAN}y-ui${NC}"
    echo -e "服务状态: ${CYAN}systemctl status y-ui${NC}"
    echo ""
    echo -e "${GREEN}访问地址: ${CYAN}http://${public_ip}:${port}${NC}"
    echo ""
    echo -e "${YELLOW}首次访问请初始化管理员账号${NC}"
    echo ""

    # 显示服务状态
    echo -e "${CYAN}服务状态:${NC}"
    systemctl status $SERVICE_NAME --no-pager -l || true
}

# 创建默认配置文件
create_default_config() {
    if [[ ! -f "$INSTALL_DIR/config.yaml" ]]; then
        echo -e "${CYAN}创建默认配置...${NC}"
        cat > "$INSTALL_DIR/config.yaml" << 'EOF'
server:
  addr: ":8080"
  mode: "release"

database:
  type: "sqlite"
  path: "./y-ui.db"

xray:
  binary_path: "/usr/local/xray/xray"
  config_path: "/etc/xray/config.json"
  assets_path: "/usr/local/xray"

log:
  level: "info"
  output: "stdout"

jwt:
  secret: ""
  expire: 24
EOF
        chmod 600 "$INSTALL_DIR/config.yaml"
    fi
}

# 安装 CLI 管理工具
install_cli() {
    echo -e "${CYAN}安装管理命令...${NC}"

    cat > /usr/local/bin/y-ui << 'EOFCLI'
#!/bin/bash

#======================================
# Y-UI 管理工具
#======================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

INSTALL_DIR="/usr/local/y-ui"
SERVICE_NAME="y-ui"

# 获取公网 IP
get_ip() {
    curl -s4 --max-time 3 https://api.ipify.org 2>/dev/null ||
    curl -s4 --max-time 3 https://ifconfig.me 2>/dev/null ||
    hostname -I 2>/dev/null | awk '{print $1}'
}

# 显示菜单
show_menu() {
    clear
    local ip=$(get_ip)
    local port=$(grep -E "^\s*addr:" "$INSTALL_DIR/config.yaml" 2>/dev/null | grep -oE '[0-9]+' || echo "8080")

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 管理面板${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "  访问地址: ${CYAN}http://${ip}:${port}${NC}"
    echo ""
    echo -e "${CYAN}--- 服务管理 ---${NC}"
    echo -e "  ${GREEN}1.${NC} 启动 Y-UI"
    echo -e "  ${GREEN}2.${NC} 停止 Y-UI"
    echo -e "  ${GREEN}3.${NC} 重启 Y-UI"
    echo -e "  ${GREEN}4.${NC} 查看状态"
    echo -e "  ${GREEN}5.${NC} 查看日志"
    echo ""
    echo -e "${CYAN}--- 配置管理 ---${NC}"
    echo -e "  ${GREEN}6.${NC} 修改端口"
    echo -e "  ${GREEN}7.${NC} 重置管理员"
    echo -e "  ${GREEN}8.${NC} 查看配置"
    echo ""
    echo -e "${CYAN}--- 系统管理 ---${NC}"
    echo -e "  ${GREEN}9.${NC} 更新 Y-UI"
    echo -e "  ${GREEN}10.${NC} 卸载 Y-UI"
    echo ""
    echo -e "${CYAN}--- Xray ---${NC}"
    echo -e "  ${GREEN}11.${NC} Xray 状态"
    echo -e "  ${GREEN}12.${NC} 重启 Xray"
    echo ""
    echo -e "  ${GREEN}0.${NC} 退出"
    echo ""
    echo -e "${GREEN}========================================${NC}"
}

start_service() {
    echo -e "${CYAN}启动 Y-UI...${NC}"
    systemctl start $SERVICE_NAME
    sleep 2
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${GREEN}Y-UI 启动成功${NC}"
        local ip=$(get_ip)
        local port=$(grep -E "^\s*addr:" "$INSTALL_DIR/config.yaml" 2>/dev/null | grep -oE '[0-9]+' || echo "8080")
        echo -e "访问: ${CYAN}http://${ip}:${port}${NC}"
    else
        echo -e "${RED}Y-UI 启动失败${NC}"
        journalctl -u $SERVICE_NAME --no-pager -n 20
    fi
}

stop_service() {
    echo -e "${CYAN}停止 Y-UI...${NC}"
    systemctl stop $SERVICE_NAME
    echo -e "${GREEN}Y-UI 已停止${NC}"
}

restart_service() {
    echo -e "${CYAN}重启 Y-UI...${NC}"
    systemctl restart $SERVICE_NAME
    sleep 2
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${GREEN}Y-UI 重启成功${NC}"
    else
        echo -e "${RED}Y-UI 重启失败${NC}"
        journalctl -u $SERVICE_NAME --no-pager -n 20
    fi
}

show_status() {
    echo -e "${CYAN}Y-UI 服务状态:${NC}"
    systemctl status $SERVICE_NAME --no-pager
    echo ""
    local ip=$(get_ip)
    local port=$(grep -E "^\s*addr:" "$INSTALL_DIR/config.yaml" 2>/dev/null | grep -oE '[0-9]+' || echo "8080")
    echo -e "访问地址: ${CYAN}http://${ip}:${port}${NC}"
}

show_logs() {
    echo -e "${CYAN}Y-UI 日志:${NC}"
    journalctl -u $SERVICE_NAME --no-pager -n 50 -f
}

change_port() {
    read -p "请输入新端口 [默认 8080]: " new_port
    new_port=${new_port:-8080}

    if ! [[ "$new_port" =~ ^[0-9]+$ ]] || [ "$new_port" -lt 1 ] || [ "$new_port" -gt 65535 ]; then
        echo -e "${RED}无效的端口号${NC}"
        return
    fi

    sed -i "s/addr: \":[0-9]*\"/addr: \":$new_port\"/" "$INSTALL_DIR/config.yaml"
    echo -e "${GREEN}端口已修改为 $new_port${NC}"
    restart_service
}

reset_admin() {
    echo -e "${YELLOW}警告: 这将删除数据库，需要重新初始化!${NC}"
    read -p "确定要重置吗? [y/N]: " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        stop_service
        rm -f "$INSTALL_DIR/y-ui.db"
        echo -e "${GREEN}已重置，请重新初始化管理员${NC}"
        start_service
    fi
}

show_config() {
    echo -e "${CYAN}当前配置:${NC}"
    cat "$INSTALL_DIR/config.yaml"
}

update_yui() {
    echo -e "${CYAN}更新 Y-UI...${NC}"
    curl -L -o /tmp/install.sh https://raw.githubusercontent.com/CNYuns/yui/main/install.sh
    bash /tmp/install.sh
}

uninstall_yui() {
    echo -e "${YELLOW}警告: 这将删除 Y-UI 及其所有数据!${NC}"
    read -p "确定要卸载吗? [y/N]: " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        systemctl stop $SERVICE_NAME 2>/dev/null
        systemctl disable $SERVICE_NAME 2>/dev/null
        rm -f /etc/systemd/system/y-ui.service
        rm -rf "$INSTALL_DIR"
        rm -f /usr/local/bin/y-ui
        systemctl daemon-reload
        echo -e "${GREEN}Y-UI 已卸载${NC}"
        exit 0
    fi
}

xray_status() {
    echo -e "${CYAN}Xray 状态:${NC}"
    if command -v xray &>/dev/null; then
        xray version
        echo ""
        local pid=$(pgrep -f "xray run")
        if [[ -n "$pid" ]]; then
            echo -e "${GREEN}Xray 运行中 (PID: $pid)${NC}"
        else
            echo -e "${YELLOW}Xray 未运行${NC}"
        fi
    else
        echo -e "${RED}Xray 未安装${NC}"
    fi
}

restart_xray() {
    echo -e "${CYAN}重启 Xray...${NC}"
    pkill -f "xray run" 2>/dev/null || true
    sleep 1
    if [[ -f /etc/xray/config.json ]]; then
        nohup /usr/local/xray/xray run -c /etc/xray/config.json > /var/log/xray/access.log 2>&1 &
        echo -e "${GREEN}Xray 已重启${NC}"
    else
        echo -e "${RED}Xray 配置文件不存在${NC}"
    fi
}

main() {
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}请使用 root 用户运行${NC}"
        exit 1
    fi

    case "$1" in
        start) start_service; exit 0 ;;
        stop) stop_service; exit 0 ;;
        restart) restart_service; exit 0 ;;
        status) show_status; exit 0 ;;
        log|logs) show_logs; exit 0 ;;
        update) update_yui; exit 0 ;;
        uninstall) uninstall_yui; exit 0 ;;
        help|--help|-h)
            echo "用法: y-ui [命令]"
            echo "命令: start|stop|restart|status|log|update|uninstall"
            exit 0
            ;;
    esac

    while true; do
        show_menu
        read -p "请选择 [0-12]: " choice
        case $choice in
            1) start_service ;;
            2) stop_service ;;
            3) restart_service ;;
            4) show_status ;;
            5) show_logs ;;
            6) change_port ;;
            7) reset_admin ;;
            8) show_config ;;
            9) update_yui ;;
            10) uninstall_yui ;;
            11) xray_status ;;
            12) restart_xray ;;
            0) echo -e "${GREEN}再见!${NC}"; exit 0 ;;
            *) echo -e "${RED}无效选项${NC}" ;;
        esac
        echo ""
        read -p "按回车键继续..."
    done
}

main "$@"
EOFCLI

    chmod +x /usr/local/bin/y-ui
    echo -e "${GREEN}管理命令已安装: y-ui${NC}"
}

# 安装 systemd 服务
install_service() {
    echo -e "${CYAN}安装系统服务...${NC}"

    cat > /etc/systemd/system/y-ui.service << 'EOF'
[Unit]
Description=Y-UI - Xray Web Management Panel
Documentation=https://github.com/CNYuns/yui
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=/usr/local/y-ui
ExecStart=/usr/local/y-ui/y-ui-server --config /usr/local/y-ui/config.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65535
Environment=GIN_MODE=release
StandardOutput=journal
StandardError=journal
SyslogIdentifier=y-ui

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable $SERVICE_NAME
    systemctl start $SERVICE_NAME

    echo -e "${GREEN}服务已安装并启动${NC}"
}

# 卸载
uninstall() {
    echo -e "${YELLOW}警告: 这将删除 Y-UI 及其所有数据!${NC}"
    read -p "确定要卸载吗? [y/N]: " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        systemctl stop $SERVICE_NAME 2>/dev/null
        systemctl disable $SERVICE_NAME 2>/dev/null
        rm -f /etc/systemd/system/y-ui.service
        rm -rf "$INSTALL_DIR"
        rm -f /usr/local/bin/y-ui
        rm -rf /var/log/y-ui
        systemctl daemon-reload
        echo -e "${GREEN}Y-UI 已卸载${NC}"
    fi
}

# 主函数
main() {
    check_root
    check_system

    case "${1:-install}" in
        install)
            install_deps
            install_xray
            install_yui
            ;;
        uninstall)
            uninstall
            ;;
        *)
            echo "用法: $0 [install|uninstall]"
            exit 1
            ;;
    esac
}

main "$@"
