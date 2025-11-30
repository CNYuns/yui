#!/bin/bash

#======================================
# Y-UI 一键安装脚本
# https://github.com/CNYuns/Yun
#======================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
INSTALL_DIR="/usr/local/y-ui"
SERVICE_NAME="y-ui"
GITHUB_REPO="CNYuns/Yun"
VERSION="${1:-latest}"

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

# 检测系统
check_system() {
    if [[ ! -f /etc/os-release ]]; then
        echo -e "${RED}无法检测操作系统${NC}"
        exit 1
    fi

    source /etc/os-release

    if [[ "$ID" != "ubuntu" && "$ID" != "debian" && "$ID" != "centos" && "$ID" != "rhel" && "$ID" != "fedora" && "$ID" != "arch" && "$ID" != "alpine" ]]; then
        echo -e "${YELLOW}警告: 未经测试的系统 $ID，继续安装...${NC}"
    fi
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
        apt-get install -y -qq curl wget unzip
    elif command -v yum &> /dev/null; then
        yum install -y -q curl wget unzip
    elif command -v dnf &> /dev/null; then
        dnf install -y -q curl wget unzip
    elif command -v pacman &> /dev/null; then
        pacman -Sy --noconfirm curl wget unzip
    elif command -v apk &> /dev/null; then
        apk add --no-cache curl wget unzip
    fi
}

# 获取最新版本
get_latest_version() {
    local latest=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" 2>/dev/null | grep -m1 '"tag_name"' | cut -d'"' -f4)
    if [[ -z "$latest" || ! "$latest" =~ ^v[0-9] ]]; then
        echo "v1.2.2"
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
    local version=$VERSION

    if [[ "$version" == "latest" ]]; then
        version=$(get_latest_version)
    fi

    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 安装程序 ${version}${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""

    # 创建安装目录
    mkdir -p "$INSTALL_DIR"
    mkdir -p /var/log/y-ui
    mkdir -p /etc/xray

    # 下载压缩包
    local download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/y-ui-linux-${arch}.tar.gz"
    local temp_file="/tmp/y-ui-linux-${arch}.tar.gz"

    download "$download_url" "$temp_file"

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

    # 安装 y-ui 管理命令
    install_cli

    # 安装 systemd 服务
    install_service

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 安装完成!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "管理命令: ${CYAN}y-ui${NC}"
    echo -e "服务状态: ${CYAN}systemctl status y-ui${NC}"
    echo -e "访问地址: ${CYAN}http://YOUR_IP:8080${NC}"
    echo ""
    echo -e "${YELLOW}首次访问请初始化管理员账号${NC}"
    echo ""
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
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

INSTALL_DIR="/usr/local/y-ui"
SERVICE_NAME="y-ui"

# 显示菜单
show_menu() {
    clear
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}       Y-UI 管理面板 v1.2.2${NC}"
    echo -e "${GREEN}========================================${NC}"
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
    echo -e "  ${GREEN}9.${NC} 编辑配置"
    echo ""
    echo -e "${CYAN}--- 系统管理 ---${NC}"
    echo -e "  ${GREEN}10.${NC} 更新 Y-UI"
    echo -e "  ${GREEN}11.${NC} 卸载 Y-UI"
    echo -e "  ${GREEN}12.${NC} 设置开机自启"
    echo -e "  ${GREEN}13.${NC} 取消开机自启"
    echo ""
    echo -e "${CYAN}--- Xray 管理 ---${NC}"
    echo -e "  ${GREEN}14.${NC} 重载 Xray 配置"
    echo -e "  ${GREEN}15.${NC} 重启 Xray"
    echo -e "  ${GREEN}16.${NC} 查看 Xray 状态"
    echo ""
    echo -e "  ${GREEN}0.${NC} 退出"
    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# 启动服务
start_service() {
    echo -e "${CYAN}启动 Y-UI...${NC}"
    systemctl start $SERVICE_NAME
    sleep 2
    if systemctl is-active --quiet $SERVICE_NAME; then
        echo -e "${GREEN}Y-UI 启动成功${NC}"
    else
        echo -e "${RED}Y-UI 启动失败${NC}"
        journalctl -u $SERVICE_NAME --no-pager -n 20
    fi
}

# 停止服务
stop_service() {
    echo -e "${CYAN}停止 Y-UI...${NC}"
    systemctl stop $SERVICE_NAME
    echo -e "${GREEN}Y-UI 已停止${NC}"
}

# 重启服务
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

# 查看状态
show_status() {
    echo -e "${CYAN}Y-UI 服务状态:${NC}"
    echo ""
    systemctl status $SERVICE_NAME --no-pager
    echo ""

    # 显示进程信息
    local pid=$(pgrep -f "y-ui-server")
    if [[ -n "$pid" ]]; then
        echo -e "${GREEN}进程 PID: $pid${NC}"
        echo -e "${GREEN}内存使用: $(ps -o rss= -p $pid | awk '{printf "%.2f MB", $1/1024}')${NC}"
        echo -e "${GREEN}CPU 使用: $(ps -o %cpu= -p $pid)%${NC}"
    fi
}

# 查看日志
show_logs() {
    echo -e "${CYAN}Y-UI 日志 (最近 50 行):${NC}"
    echo ""
    journalctl -u $SERVICE_NAME --no-pager -n 50
    echo ""
    echo -e "${YELLOW}按 Ctrl+C 退出实时日志${NC}"
    read -p "是否查看实时日志? [y/N]: " choice
    if [[ "$choice" == "y" || "$choice" == "Y" ]]; then
        journalctl -u $SERVICE_NAME -f
    fi
}

# 修改端口
change_port() {
    read -p "请输入新端口 [默认 8080]: " new_port
    new_port=${new_port:-8080}

    if ! [[ "$new_port" =~ ^[0-9]+$ ]] || [ "$new_port" -lt 1 ] || [ "$new_port" -gt 65535 ]; then
        echo -e "${RED}无效的端口号${NC}"
        return
    fi

    sed -i "s/addr: \":[0-9]*\"/addr: \":$new_port\"/" "$INSTALL_DIR/config.yaml"
    echo -e "${GREEN}端口已修改为 $new_port${NC}"

    read -p "是否重启服务? [Y/n]: " restart
    if [[ "$restart" != "n" && "$restart" != "N" ]]; then
        restart_service
    fi
}

# 重置管理员
reset_admin() {
    echo -e "${YELLOW}警告: 这将删除所有用户数据!${NC}"
    read -p "确定要重置吗? [y/N]: " confirm

    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        stop_service
        rm -f "$INSTALL_DIR/y-ui.db"
        echo -e "${GREEN}管理员已重置，请重新初始化${NC}"
        start_service
    fi
}

# 查看配置
show_config() {
    echo -e "${CYAN}当前配置:${NC}"
    echo ""
    cat "$INSTALL_DIR/config.yaml"
}

# 编辑配置
edit_config() {
    local editor=${EDITOR:-vi}
    $editor "$INSTALL_DIR/config.yaml"

    read -p "是否重启服务? [Y/n]: " restart
    if [[ "$restart" != "n" && "$restart" != "N" ]]; then
        restart_service
    fi
}

# 更新
update_yui() {
    echo -e "${CYAN}检查更新...${NC}"
    curl -L -o /tmp/install.sh https://raw.githubusercontent.com/CNYuns/Yun/main/install.sh
    bash /tmp/install.sh
}

# 卸载
uninstall_yui() {
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

# 设置开机自启
enable_autostart() {
    systemctl enable $SERVICE_NAME
    echo -e "${GREEN}已设置开机自启${NC}"
}

# 取消开机自启
disable_autostart() {
    systemctl disable $SERVICE_NAME
    echo -e "${GREEN}已取消开机自启${NC}"
}

# 重载 Xray 配置
reload_xray() {
    echo -e "${CYAN}重载 Xray 配置...${NC}"
    curl -s -X POST "http://localhost:8080/api/v1/system/reload" -H "Authorization: Bearer $(cat $INSTALL_DIR/.token 2>/dev/null)" || echo -e "${YELLOW}请通过 Web 界面操作${NC}"
}

# 重启 Xray
restart_xray() {
    echo -e "${CYAN}重启 Xray...${NC}"
    curl -s -X POST "http://localhost:8080/api/v1/system/restart" -H "Authorization: Bearer $(cat $INSTALL_DIR/.token 2>/dev/null)" || echo -e "${YELLOW}请通过 Web 界面操作${NC}"
}

# 查看 Xray 状态
xray_status() {
    echo -e "${CYAN}Xray 状态:${NC}"
    local xray_pid=$(pgrep -f "xray run")
    if [[ -n "$xray_pid" ]]; then
        echo -e "${GREEN}Xray 运行中 (PID: $xray_pid)${NC}"
        xray version 2>/dev/null || echo "Xray 版本: 未知"
    else
        echo -e "${RED}Xray 未运行${NC}"
    fi
}

# 主函数
main() {
    # 检查 root
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}请使用 root 用户运行${NC}"
        exit 1
    fi

    # 处理命令行参数
    case "$1" in
        start)
            start_service
            exit 0
            ;;
        stop)
            stop_service
            exit 0
            ;;
        restart)
            restart_service
            exit 0
            ;;
        status)
            show_status
            exit 0
            ;;
        log|logs)
            show_logs
            exit 0
            ;;
        update)
            update_yui
            exit 0
            ;;
        uninstall)
            uninstall_yui
            exit 0
            ;;
        help|--help|-h)
            echo "用法: y-ui [命令]"
            echo ""
            echo "命令:"
            echo "  start     启动服务"
            echo "  stop      停止服务"
            echo "  restart   重启服务"
            echo "  status    查看状态"
            echo "  log       查看日志"
            echo "  update    更新 Y-UI"
            echo "  uninstall 卸载 Y-UI"
            echo ""
            echo "不带参数运行显示交互式菜单"
            exit 0
            ;;
    esac

    # 交互式菜单
    while true; do
        show_menu
        read -p "请选择 [0-16]: " choice

        case $choice in
            1) start_service ;;
            2) stop_service ;;
            3) restart_service ;;
            4) show_status ;;
            5) show_logs ;;
            6) change_port ;;
            7) reset_admin ;;
            8) show_config ;;
            9) edit_config ;;
            10) update_yui ;;
            11) uninstall_yui; exit 0 ;;
            12) enable_autostart ;;
            13) disable_autostart ;;
            14) reload_xray ;;
            15) restart_xray ;;
            16) xray_status ;;
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
Documentation=https://github.com/CNYuns/Yun
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
RestartPreventExitStatus=23
StartLimitIntervalSec=60
StartLimitBurst=5
LimitNOFILE=65535
LimitNPROC=4096
MemoryMax=512M
CPUQuota=80%
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/usr/local/y-ui /etc/xray /var/log/y-ui
Environment=GIN_MODE=release
StandardOutput=journal
StandardError=journal
SyslogIdentifier=y-ui
WatchdogSec=30

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
