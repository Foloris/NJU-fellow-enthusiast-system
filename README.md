# 搭个伴 · 校园同好社区

Vue 3 + TypeScript 前端、Go 后端、MySQL 数据库。支持同好分享、校园新闻、搭子申请、审批后实时私信、举报及管理员审核，沿用原有页面设计和业务流程。

网页需要登录后访问；未登录打开首页或业务链接会先进入登录/注册页。新账号注册需要确认密码、校园邮箱及邮件验证码，成功后自动登录并进入原目标页面。

## 环境

- Go 1.24 或更新版本
- Node.js 22.13 或更新版本，仅用于前端工具链
- MySQL 8.0 或更新版本，使用 InnoDB、utf8mb4

## 配置和启动

先在 MySQL 中创建独立数据库，例如：

```sql
CREATE DATABASE bigshot CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

复制根目录 `.env.example` 为 `.env`，填写 `DB_*`、至少32字节的随机 `JWT_SECRET`、8～72字节的 `ADMIN_PASSWORD`。不要提交真实密码。

```powershell
npm ci
go mod download
npm run dev
```

访问 http://localhost:5173 。Vite 将 `/api`、`/uploads`、`/ws` 代理至3001端口的 Go 服务。

首次启动自动执行版本化 SQL 迁移并初始化管理员。`SEED_DEMO=true` 时，在只有管理员的空库中插入演示数据；重复启动不覆盖管理员密码、不重复写入演示数据。

| 变量 | 默认值 / 要求 |
| --- | --- |
| `PORT` | `3001` |
| `APP_ENV` | `development`；生产设置为 `production`，启用 Secure Cookie |
| `DB_HOST` / `DB_PORT` | `127.0.0.1` / `3306` |
| `DB_NAME` | `campus_fellow`，需提前创建 |
| `DB_USER` / `DB_PASSWORD` | 数据库账号及密码，需要指定库内建表和读写权限 |
| `JWT_SECRET` | 必填，至少32字节随机值，拒绝示例占位值 |
| `ADMIN_USERNAME` | `admin`，仅用于首次创建 |
| `ADMIN_PASSWORD` | 必填，8～72字节，仅用于首次创建 |
| `SEED_DEMO` | 默认 `false`，示例为 `true`；生产必须 `false` |
| `UPLOAD_DIR` | `server/uploads` |
| `CLIENT_DIST_DIR` | `client/dist` |
| `CAMPUS_EMAIL_DOMAIN` | `smail.nju.edu.cn`，唯一允许注册的邮箱域名，不带@ |
| `SMTP_HOST` / `SMTP_PORT` | 发信SMTP主机 / `465`；主机留空时无法发送注册验证码 |
| `SMTP_TLS_MODE` | `tls`（通常465），或 `starttls`（通常587） |
| `SMTP_USERNAME` / `SMTP_PASSWORD` | 发信账号和客户端专用密码/授权码 |
| `SMTP_FROM` | 发件邮箱，默认同SMTP_USERNAME |

命令默认在项目根目录运行。进程环境变量优先于 `.env`。配置文件支持 `KEY=value`、整行注释、单/双引号包围的字面值，不执行变量替换。

## 构建与部署

```powershell
npm run build
.\server\bin\server.exe
```

Linux/macOS 执行 `./server/bin/server`。二进制同时提供前端静态文件、API、上传文件和 WebSocket，运行时无需 Node.js。`npm start` 使用 `go run`，适合已安装 Go 的环境。从其它目录启动时通过 `-root` 指定项目根目录。

仅执行数据库迁移：

```powershell
go run ./server/cmd/server -migrate-only
```

生产环境配置 HTTPS 反向代理，转发 WebSocket Upgrade，持久化上传目录并同时备份 MySQL 和图片。数据库连接使用 UTC，接口时间采用 RFC3339。终止服务时关闭实时连接并等待 HTTP 请求完成。

已有管理员不会因修改 `.env` 自动改密。不要删除数据库来修改密码或升级结构。历史 SQLite 数据不自动导入；如果需要保留，先备份并另行安排数据迁移。

## 验证

```powershell
npm run typecheck
go test ./server/...
go vet ./server/...
npm run build
npm run smoke
```

普通 Go 测试不依赖 MySQL。`npm run smoke` 使用 `.env` 中的连接账号创建随机的 `campus_it*_test` 临时库，在独立 HTTP 服务上验证业务后删除该临时库；账号需额外具备创建和删除数据库权限。它不向应用库 `DB_NAME` 写入测试数据，也无需事先启动后端。

详细准备、手动验收和排错见 [测试说明](docs/测试说明.md)。

云服务器建库、授权和建表见 [MySQL建表步骤](docs/云服务器MySQL建表.md)。

暂不使用域名时，按 [云服务器IP访问](docs/云服务器IP访问.md) 构建并开放HTTP测试访问。

校园邮箱允许后缀、南大邮箱SMTP开通及配置步骤见 [校园邮箱与SMTP配置](docs/校园邮箱与SMTP配置.md)。验证码60秒有效，注册成功后单次消耗，邮箱不能重复绑定。

## 分层结构

```text
client/                         Vue + TypeScript 页面、组件、样式
server/cmd/server/              启动入口、依赖组装、优雅退出
server/internal/config/         环境配置与校验
server/internal/mailer/         使用TLS/STARTTLS认证的SMTP验证码发信
server/internal/model/          业务模型与接口数据类型
server/internal/httpapi/        路由、JSON、Cookie、上传及 HTTP 测试
server/internal/service/        业务校验、权限、认证、流程和事件
server/internal/repository/     MySQL 查询、事务、初始化数据
server/internal/repository/migrations/  版本化 SQL 结构迁移
server/internal/realtime/       WebSocket 连接、并发写入和心跳
docs/                          项目约束与测试说明
go.mod / go.sum                 Go 依赖
package.json / package-lock.json 前端工具链依赖
```

接口层通过业务层访问数据；SQL 只出现在 repository。HTTP 和数据库访问使用标准库 `net/http`、`database/sql`。四个直接 Go 依赖负责 MySQL、JWT、WebSocket 和 bcrypt，不引入 Web 框架或 ORM。后端已移除 Node.js 和 SQLite 运行路径。

## 演示账号

仅在 `SEED_DEMO=true` 且库中只有管理员时创建 `linzhi`、`zhouyu`、`chenmu`，密码均为 `Demo123!`。管理员使用首次初始化时 `.env` 配置的账号密码。
