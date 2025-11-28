# XPanel

基于 Xray-core 的 Web 管理面板，提供可视化的代理服务管理界面。

> **重要提示**：本软件仅供个人学习和研究使用，请务必遵守当地法律法规。详见 [LICENSE](LICENSE)。

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

## 快速开始

### 环境要求

- Go 1.22 或更高版本
- Node.js 18 或更高版本
- Xray-core（需预先安装）

### 下载安装

从 [Releases](../../releases) 页面下载对应平台的二进制文件：

| 平台 | 架构 | 文件名 |
|------|------|--------|
| Linux | amd64 | xpanel-linux-amd64 |
| Linux | arm64 | xpanel-linux-arm64 |
| Linux | armv7 | xpanel-linux-arm |
| Windows | amd64 | xpanel-windows-amd64.exe |
| Windows | arm64 | xpanel-windows-arm64.exe |
| macOS | amd64 | xpanel-darwin-amd64 |
| macOS | arm64 | xpanel-darwin-arm64 |

### 配置文件

首次启动会自动生成配置文件，也可手动创建 `config.yaml`：

```yaml
server:
  addr: ":8080"          # 监听地址
  mode: "release"        # gin 模式

auth:
  # jwt_secret 会自动生成，无需手动配置
  token_ttl_hours: 24    # Token 有效期

database:
  driver: "sqlite"
  dsn: "xpanel.db"

xray:
  binary_path: "/usr/local/bin/xray"
  config_path: "/etc/xray/config.json"

tls:
  auto_acme: false
  cert_path: "/etc/xpanel/certs"
  email: ""

log:
  level: "info"
  output: "stdout"
```

### 启动服务

```bash
# 添加执行权限
chmod +x xpanel-linux-amd64

# 启动（首次启动自动生成安全配置）
./xpanel-linux-amd64 --config config.yaml
```

### 首次使用

1. 访问 `http://your-server:8080`
2. 系统会提示初始化管理员账号
3. 设置邮箱和密码后即可登录

## Docker 部署

### 使用 Docker Compose

```bash
# 克隆仓库
git clone https://github.com/CNYuns/Yun.git
cd Yun/xpanel

# 启动
docker-compose up -d
```

### 手动构建

```bash
# 构建镜像
docker build -t xpanel .

# 运行容器
docker run -d \
  --name xpanel \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -v /etc/xray:/etc/xray \
  xpanel
```

## 从源码构建

```bash
# 克隆仓库
git clone https://github.com/CNYuns/Yun.git
cd Yun/xpanel

# 安装前端依赖并构建
cd frontend
npm install
npm run build
cd ..

# 构建后端
cd backend
go build -o ../xpanel ./cmd/xpanel
cd ..

# 运行
./xpanel --config config.yaml
```

或使用 Makefile：

```bash
make build
./xpanel --config config.yaml
```

## API 文档

### 认证

所有 API 请求（除登录外）需要在 Header 中携带 Token：

```
Authorization: Bearer <token>
```

### 响应格式

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

### 主要接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/auth/login | 登录 |
| POST | /api/v1/auth/logout | 登出 |
| GET | /api/v1/auth/profile | 获取用户信息 |
| GET | /api/v1/clients | 获取用户列表 |
| POST | /api/v1/clients | 创建用户 |
| GET | /api/v1/inbounds | 获取入站列表 |
| POST | /api/v1/inbounds | 创建入站 |
| GET | /api/v1/stats/summary | 获取流量汇总 |
| GET | /api/v1/system/status | 获取系统状态 |
| POST | /api/v1/system/reload | 重载 Xray 配置 |

## 权限说明

| 角色 | 说明 | 权限范围 |
|------|------|----------|
| admin | 管理员 | 全部功能 |
| operator | 操作员 | 用户管理、入站管理 |
| viewer | 查看者 | 仅查看 |

## 安全特性

- **JWT 密钥**：首次启动自动生成 64 位安全随机密钥
- **登录保护**：每分钟最多 5 次登录尝试，超限封禁 15 分钟
- **CORS 安全**：基于白名单的跨域策略
- **文件权限**：配置文件和证书使用 0600 权限
- **安全头部**：自动添加 CSP、HSTS 等安全头

## 开发指南

### 目录结构

```
xpanel/
├── backend/                # Go 后端
│   ├── cmd/xpanel/        # 入口文件
│   ├── internal/          # 内部包
│   │   ├── config/        # 配置管理
│   │   ├── database/      # 数据库
│   │   ├── handlers/      # API 处理器
│   │   ├── middleware/    # 中间件
│   │   ├── models/        # 数据模型
│   │   ├── scheduler/     # 调度任务
│   │   ├── services/      # 业务逻辑
│   │   └── xray/          # Xray 管理
│   └── pkg/               # 公共包
├── frontend/              # Vue3 前端
│   ├── src/
│   │   ├── api/           # API 调用
│   │   ├── components/    # 组件
│   │   ├── router/        # 路由
│   │   ├── stores/        # 状态管理
│   │   └── views/         # 页面
│   └── ...
├── .github/workflows/     # CI/CD
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

### 本地开发

```bash
# 启动后端（开发模式）
make dev-backend

# 启动前端（开发模式）
make dev-frontend
```

## 常见问题

### Q: 如何重置管理员密码？

删除数据库文件 `xpanel.db`，重新启动后会提示创建新管理员。

### Q: Xray 配置在哪里？

配置文件默认路径为 `/etc/xray/config.json`，可在 `config.yaml` 中修改。

### Q: 如何查看日志？

日志默认输出到标准输出，可在 `config.yaml` 中配置输出到文件。

### Q: JWT 密钥在哪里？

首次启动时自动生成并保存到 `config.yaml` 文件中。

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
