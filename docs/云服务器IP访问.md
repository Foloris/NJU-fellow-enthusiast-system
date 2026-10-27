# 云服务器通过 IP 访问

适用：Linux 云服务器已有项目源码、Go 1.24+、Node.js 22.13+ 和 MySQL 8.0+，`bigshot` 已创建。先通过 `http://公网IPv4:3001` 测试，不涉及域名、Nginx 或证书。

## 1. 检查配置

进入服务器上的项目根目录（有 `go.mod`、`package.json` 的目录），编辑已有 `.env`，保留已经设置的 DB_*、JWT_SECRET 和管理员凭据：

```bash
nano .env
```

核对：

```dotenv
PORT=3001
APP_ENV=development
SEED_DEMO=false
DB_NAME=bigshot
CLIENT_DIST_DIR=client/dist
UPLOAD_DIR=server/uploads
```

此阶段使用 HTTP，所以必须使用 `APP_ENV=development`。当前代码在 production 下发送 Secure Cookie，浏览器通过公网 HTTP 不会携带该登录 Cookie，会表现为登录后又回到登录页。使用 development 不会关闭后端登录和权限检查；仍需有效的 JWT_SECRET、管理员密码和数据库账号。

此访问方式用于测试，HTTP 不加密传输，请使用测试账号和测试数据。后续配置 HTTPS 时再改为 production。

## 2. 构建

在 Linux 的项目根目录执行：

```bash
node --version
go version
npm ci
npm run build
```

成功后应存在：

```text
client/dist/index.html
server/bin/server
```

不要上传 Windows 编译的 server.exe 作为 Linux 服务。npm run build 在 Linux 上会自动生成 Linux 可执行文件；构建完成后的服务运行不需要 Node.js。

## 3. 前台启动并检查服务器内部访问

```bash
./server/bin/server
```

保持该终端运行，另开一个 SSH 终端执行：

```bash
curl -i http://127.0.0.1:3001/api/health
curl -I http://127.0.0.1:3001/
```

健康检查应为200且包含 `"ok":true`，网页请求也应为200。当前 Go 服务会在3001端口同时提供网页、API、上传图片和 WebSocket，前端使用相对路径和当前网页主机连接，无需修改代码里的IP。

仅看到 `Go API listening` 日志还不能证明启动成功：如果紧接着出现 `bind` 错误，说明已有进程占用端口。使用下面的命令确认监听者，不要重复启动第二个后端：

```bash
sudo ss -lntp '( sport = :3001 )'
```

监听地址应为 `0.0.0.0:3001`、`*:3001` 或 `[::]:3001` 等通配地址，而非只监听127.0.0.1。

## 4. 放行云端端口

在云厂商控制台，给这台实例实际关联的安全组/防火墙添加入站规则：

| 项目 | 值 |
| --- | --- |
| 协议 | TCP |
| 端口 | 3001 |
| 来源 | 先使用你测试设备的公网IPv4/32；需要任意IPv4设备访问时使用0.0.0.0/0 |
| 动作 | 允许 |

使用实例的公网IPv4，不是服务器上的127.0.0.1或内网地址。本步骤只需开放3001，不需要为网站开放MySQL的3306或Vite的5173。

Ubuntu 使用UFW时先检查：

```bash
sudo ufw status
```

若为 `active`，添加允许规则：

```bash
sudo ufw allow 3001/tcp
```

若为 `inactive`，当前没有UFW规则拦截，不需要为这一步专门启用UFW。其它Linux发行版若使用firewalld，应按其规则放行3001/TCP。

## 5. 从你的电脑访问

在电脑浏览器输入完整地址，明确使用http：

```text
http://你的服务器公网IPv4:3001
```

也可先检查：

```text
http://你的服务器公网IPv4:3001/api/health
```

网页未登录时应跳至登录/注册页。验证注册（含确认密码）、登录、刷新保持会话、发帖、上传图片、申请审批和两台设备实时聊天。第二台设备也必须使用同一个公网IP和端口，且安全组来源覆盖它的公网IP。

## 6. 退出 SSH 后继续运行

先在原来运行服务的终端按 Ctrl+C 停止前台服务，再在项目根目录执行一次：

```bash
nohup ./server/bin/server > server/server.log 2>&1 < /dev/null &
```

查看进程和日志：

```bash
sudo ss -lntp '( sport = :3001 )'
tail -n 80 server/server.log
```

nohup适合这一轮临时测试，不提供崩溃后自动重启或开机自启。后续正式运行可配置systemd。不要同时保留前台、nohup和npm启动的多个后端。

## 排查顺序

| 现象 | 优先检查 |
| --- | --- |
| 内部health请求失败 | 服务日志、MySQL连接、密钥配置、端口是否被占用 |
| 内部health成功，公网超时 | 安全组是否绑定当前实例、来源IP、系统防火墙、公网IP和TCP3001 |
| 公网connection refused | 进程是否退出、端口是否正确、是否监听通配地址 |
| API正常，网页503 | client/dist是否已构建、是否在项目根目录启动 |
| 登录后回到登录页 | HTTP阶段的APP_ENV是否为development；进程环境变量会覆盖.env |
| 注册页正常，注册失败 | API响应和后端日志、数据库账号权限 |
| 上传失败 | server/uploads写权限、文件实际格式和5MB限制 |
| SSH断开后无法访问 | 是否仍为前台进程；检查nohup日志和监听端口 |

本次已核对监听地址、静态资源、Cookie和WebSocket代码。尚未连接云服务器，公网是否可达应以上述服务器内部和外部两轮检查为准。
