# 搭个伴 · 校园同好社区

一个可直接运行的校园交流与找搭子平台。用户可以分享校园见闻、发布搭子邀请、提交申请，并在申请通过后通过实时私信继续联系；管理员可以发布校园新闻、审核举报、下架内容和管理账号。

## 功能

- 账号注册、登录、退出与个人资料编辑
- 同好分享和校园新闻的发布、搜索、编辑、删除
- 分享帖评论、点赞、收藏
- 搭子邀请发布、搜索、编辑、删除
- 搭子申请与同意/拒绝流程
- 只有申请通过后才会建立的一对一实时私信
- 消息历史、未读数量和已读状态
- 举报分享帖、搭子帖、私信或用户
- 管理员举报审核、内容下架/恢复/删除、账号封禁
- 响应式桌面端和移动端界面
- SQLite 自动建表与演示数据

## 技术栈

- 前端：Vue 3、TypeScript、Vite、Vue Router
- 后端：Node.js、Express 5、WebSocket
- 数据库：Node.js 内置 SQLite（`node:sqlite`）
- 认证：HttpOnly Cookie + JWT

需要 Node.js 22.13 或更新版本。当前使用 Node.js 内置 SQLite，启动时出现一次 `ExperimentalWarning` 属于正常提示。

## 本地运行

```powershell
npm install
npm run dev
```

打开 `http://localhost:5173`。Vite 会将 API 和 WebSocket 请求代理到 `http://localhost:3001`。

也可以构建后以单端口运行：

```powershell
npm run build
npm start
```

打开 `http://localhost:3001`。

## 演示账号

| 角色 | 账号 | 密码 |
| --- | --- | --- |
| 普通用户 | `linzhi` | `Demo123!` |
| 普通用户 | `zhouyu` | `Demo123!` |
| 普通用户 | `chenmu` | `Demo123!` |
| 管理员 | `admin` | `CampusAdmin123!` |

管理员账号由首次启动时自动创建。正式部署前，请复制 `.env.example` 为 `.env`，修改 `JWT_SECRET`、`ADMIN_USERNAME` 和 `ADMIN_PASSWORD`，并删除已有的 `server/data/campus.db` 后重新启动，让新配置生效。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `3001` | 后端监听端口 |
| `JWT_SECRET` | 本地开发密钥 | 生产环境必须替换 |
| `ADMIN_USERNAME` | `admin` | 首次创建的管理员账号 |
| `ADMIN_PASSWORD` | `CampusAdmin123!` | 首次创建的管理员密码 |
| `SEED_DEMO` | `true` | 是否写入演示用户和内容 |

## 验证

先启动服务，再运行完整 API 冒烟测试：

```powershell
npm start
npm run smoke
```

测试覆盖注册登录、内容创建、点赞、收藏、评论、申请审批、WebSocket 实时消息、举报审核、下架和封禁。测试会创建临时数据；用于正式数据的数据库不建议直接运行该脚本。

## 目录

```text
client/
  public/demo/          演示插画
  src/components/       通用组件
  src/views/            页面
  src/stores/           登录状态与提示
server/
  src/routes/           API 路由
  src/db.ts             数据库结构与种子数据
  src/realtime.ts       WebSocket 在线推送
  scripts/smoke.mjs     端到端 API 冒烟测试
```

生产环境建议在前方配置 HTTPS 反向代理，并把 `JWT_SECRET` 设置为足够长的随机字符串。SQLite 数据与上传图片分别保存在 `server/data/` 和 `server/uploads/`，备份时需同时备份这两个目录。
