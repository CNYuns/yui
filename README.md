# Y-UI

基于 Xray-core 的 Web 管理面板，提供可视化的代理服务管理界面。

> **重要提示**：本软件仅供个人学习和研究使用，请务必遵守当地法律法规。详见 [LICENSE](LICENSE)。

## 一键安装

```bash
bash <(curl -Ls https://raw.githubusercontent.com/CNYuns/Yun/main/install.sh)
```

安装完成后使用 `y-ui` 命令管理。

## 功能特性

- **多协议支持**：VMess、VLESS、Trojan、Shadowsocks 等
- **可视化管理**：无需手动编辑 JSON 配置文件
- **用户管理**：支持多用户、流量限制、到期时间设置
- **证书管理**：自动申请和续签 Let's Encrypt 证书
- **流量统计**：实时查看流量使用情况
- **系统监控**：CPU、内存、磁盘使用率监控
- **审计日志**：记录所有操作日志
- **RBAC 权限**：管理员、操作员、查看者三级权限
- **安全特性**：自动生成 JWT 密钥、登录速率限制、CORS 白名单

## 系统特性

- **自启动**：systemd 服务，开机自动启动
- **保活机制**：进程异常退出自动重启
- **资源限制**：内存 512M，CPU 80% 上限
- **安全加固**：最小权限原则运行
- **看门狗**：30 秒自检周期

## 管理命令

```bash
# 交互式菜单
y-ui

# 快捷命令
y-ui start      # 启动服务
y-ui stop       # 停止服务
y-ui restart    # 重启服务
y-ui status     # 查看状态
y-ui log        # 查看日志
y-ui update     # 更新 Y-UI
y-ui uninstall  # 卸载 Y-UI
```

### 交互式菜单

```
========================================
       Y-UI 管理面板 v1.2.1
========================================

--- 服务管理 ---
  1. 启动 Y-UI
  2. 停止 Y-UI
  3. 重启 Y-UI
  4. 查看状态
  5. 查看日志

--- 配置管理 ---
  6. 修改端口
  7. 重置管理员
  8. 查看配置
  9. 编辑配置

--- 系统管理 ---
  10. 更新 Y-UI
  11. 卸载 Y-UI
  12. 设置开机自启
  13. 取消开机自启

--- Xray 管理 ---
  14. 重载 Xray 配置
  15. 重启 Xray
  16. 查看 Xray 状态

  0. 退出
```

## 技术栈

### 后端
- Go 1.22+
- Gin Web 框架
- GORM ORM
- SQLite 数据库
- JWT 认证
- ACME 证书管理

### 前端
- Vue 3
- TypeScript
- Element Plus
- Vite
- ECharts

## 手动安装

### 下载

从 [Releases](../../releases) 页面下载对应平台的二进制文件：

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Linux | amd64 | y-ui-linux-amd64 |
| Linux | arm64 | y-ui-linux-arm64 |
| Linux | armv7 | y-ui-linux-arm |
| Windows | amd64 | y-ui-windows-amd64.exe |
| Windows | arm64 | y-ui-windows-arm64.exe |
| macOS | amd64 | y-ui-darwin-amd64 |
| macOS | arm64 | y-ui-darwin-arm64 |

### 配置

首次启动会自动生成配置文件 `/usr/local/y-ui/config.yaml`：

```yaml
server:
  addr: ":8080"          # 监听地址
  mode: "release"        # gin 模式

auth:
  # jwt_secret 会自动生成，无需手动配置
  token_ttl_hours: 24    # Token 有效期

database:
  driver: "sqlite"
  dsn: "y-ui.db"

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/etc/xray/config.json"

log:
  level: "info"
  output: "/var/log/y-ui/y-ui.log"
```

## Docker 部署

```bash
docker run -d \
  --name y-ui \
  --restart=always \
  -p 8080:8080 \
  -v /etc/y-ui:/usr/local/y-ui \
  -v /etc/xray:/etc/xray \
  cnyuns/y-ui:latest
```

## 从源码构建

```bash
# 克隆仓库
git clone https://github.com/CNYuns/Yun.git
cd Yun

# 构建
make build

# 运行
./y-ui-server --config config.yaml
```

## API 文档

### 认证

所有 API 请求（除登录外）需要在 Header 中携带 Token：

```
Authorization: Bearer <token>
```

### 主要接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/clients | 获取用户列表 |
| POST | /api/v1/clients | 创建用户 |
| GET | /api/v1/inbounds | 获取入站列表 |
| POST | /api/v1/inbounds | 创建入站 |
| GET | /api/v1/stats/summary | 获取流量汇总 |
| GET | /api/v1/system/status | 获取系统状态 |
| POST | /api/v1/system/reload | 重载 Xray 配置 |

## 安全特性

- **JWT 密钥**：首次启动自动生成 64 位安全随机密钥
- **登录保护**：每分钟最多 5 次登录尝试，超限封禁 15 分钟
- **CORS 安全**：基于白名单的跨域策略
- **文件权限**：配置文件和证书使用 0600 权限
- **安全头部**：自动添加 CSP、HSTS 等安全头
- **进程隔离**：systemd 安全加固配置

## 目录结构

```
/usr/local/y-ui/           # 安装目录
├── y-ui-server            # 主程序
├── config.yaml            # 配置文件
├── y-ui.db                # 数据库
└── dist/                  # 前端文件

/var/log/y-ui/             # 日志目录
/etc/xray/                 # Xray 配置
```

## 常见问题

### Q: 如何重置管理员密码？

```bash
y-ui
# 选择 7. 重置管理员
```

### Q: 如何修改端口？

```bash
y-ui
# 选择 6. 修改端口
```

### Q: 如何查看日志？

```bash
y-ui log
# 或
journalctl -u y-ui -f
```

### Q: 如何更新？

```bash
y-ui update
# 或
bash <(curl -Ls https://raw.githubusercontent.com/CNYuns/Yun/main/install.sh)
```

## 许可证

本项目采用自定义许可证，详见 [LICENSE](LICENSE)。

**重要限制**：
- 禁止用于非法用途
- 禁止任何形式的商业用途
- 禁止安装在非个人电脑上
- 违规使用将追究法律责任

## 联系方式

- 邮箱：391475293@qq.com
- GitHub：[CNYuns/Yun](https://github.com/CNYuns/Yun)

## 致谢

- [Xray-core](https://github.com/XTLS/Xray-core)
- [Gin](https://github.com/gin-gonic/gin)
- [Vue.js](https://vuejs.org/)
- [Element Plus](https://element-plus.org/)
