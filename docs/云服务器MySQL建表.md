# 云服务器 MySQL 建表

适用：在云服务器终端操作，使用 MySQL 8.0 或更高版本，库名为 `bigshot`。本项目使用 MySQL 的 `utf8mb4_0900_*` 排序规则，不适用于 MySQL 5.7 或直接替换为 MariaDB。

建表 SQL 的唯一来源是 `server/internal/repository/migrations/` 中按版本排列的文件。当前为 `001_initial.sql` 和 `002_email_verification.sql`。推荐由 Go 迁移程序执行，它会记录迁移版本、串行执行建表，成功后才登记完成。

## 1. 创建数据库

在云服务器终端登录 MySQL：

```bash
mysql -u root -p
```

输入 MySQL 管理员密码后执行下面的 SQL。如果 Ubuntu 上 root 使用系统账号认证，也可以通过 `sudo mysql` 登录。

```sql
SELECT VERSION();

CREATE DATABASE IF NOT EXISTS bigshot
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

如果 `bigshot` 已存在，这条命令不会改变已有表的定义或清空数据。此流程适合本项目的新库；已有其它系统数据的同名库不要直接混用。

## 2. 创建应用数据库账号

下面按应用和 MySQL 在同一台服务器、应用通过 `127.0.0.1:3306` 连接配置。将示例中的密码文字替换为独立的数据库密码再执行：

```sql
CREATE USER IF NOT EXISTS 'bigshot_app'@'127.0.0.1'
  IDENTIFIED BY '替换为独立的数据库强密码';

GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, REFERENCES
  ON bigshot.* TO 'bigshot_app'@'127.0.0.1';

SHOW GRANTS FOR 'bigshot_app'@'127.0.0.1';
EXIT;
```

账号已存在时 `CREATE USER IF NOT EXISTS` 不会覆盖原密码，配置时应使用它的现有密码。该账号用于运行应用和迁移，不授予全局权限；`npm run smoke` 的临时建库测试需要另外的测试管理权限，不属于此账号的用途。

若数据库是云厂商托管的独立 MySQL，请使用其管理控制台创建账号和授权，并将下一步的 DB_HOST 改成数据库内网地址；`127.0.0.1` 只用于同机 MySQL。

## 3. 配置项目连接

进入已上传的项目根目录（包含 go.mod 的目录），保留已有配置文件：

```bash
[ -f .env ] || cp .env.example .env
nano .env
```

设置下面的字段，其它项目路径配置可沿用示例：

```dotenv
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=bigshot
DB_USER=bigshot_app
DB_PASSWORD=与刚才创建账号时相同的密码
JWT_SECRET=替换为至少32字节的随机密钥
ADMIN_USERNAME=admin
ADMIN_PASSWORD=替换为8到72字节的后台管理员密码
SEED_DEMO=false
```

数据库密码和后台管理员密码是两套不同的凭据。JWT_SECRET 可以使用 `openssl rand -hex 32` 在服务器生成后填入，不要使用示例占位文字。当前程序在迁移时也会校验 JWT_SECRET 和 ADMIN_PASSWORD，所以这两个字段必须同时填写。

如果使用 `APP_ENV=production`，后续网页登录须配置 HTTPS；仅建表不需要开放网站端口，也不需要启动前端。

## 4. 执行建表

在项目根目录运行，服务器需要 Go 1.24 或更新版本：

```bash
go run ./server/cmd/server -migrate-only
```

若已经生成 Linux 可执行文件，则也可以运行：

```bash
./server/bin/server -migrate-only
```

Windows 生成的 `.exe` 不可用于 Linux。上面两条命令择一执行，成功应看到 `MySQL migrations applied`。此命令完成后退出，不监听3001端口，不插入管理员和演示数据；管理员会在正常启动服务时按 `.env` 自动创建。

## 5. 检查结果

```bash
mysql -h 127.0.0.1 -P 3306 -u bigshot_app -p bigshot
```

```sql
SHOW TABLES;
SELECT version, applied_at FROM schema_migrations;
SHOW CREATE TABLE applications\G
EXIT;
```

全新数据库应包含下列14张表，迁移版本应包含 `001_initial.sql` 和 `002_email_verification.sql`：

| 表 | 用途 |
| --- | --- |
| users | 账号、密码哈希、资料、角色与封禁状态 |
| exchange_posts | 同好分享和校园新闻 |
| mate_posts | 搭子邀请、标签和通过申请后可见的联系方式 |
| comments | 分享评论 |
| likes | 点赞，限制同一用户对同一帖子重复记录 |
| favorites | 收藏，限制同一用户对同一帖子重复记录 |
| applications | 搭子申请和审批结果，限制重复申请 |
| conversations | 申请通过后建立的会话，一份申请对应一个会话 |
| messages | 会话消息及已读时间 |
| reports | 举报、证据快照与审核结果 |
| user_emails | 用户与已验证校园邮箱的唯一绑定 |
| email_verifications | 邮箱验证码摘要、有效期和错误次数 |
| email_verification_send_limits | 验证码发送来源频率限制 |
| schema_migrations | 已执行的数据库迁移版本 |

全部业务表使用 InnoDB，通过外键保持关联。删除搭子会级联删除申请、会话和消息；举报快照保留其被举报对象的文本证据。建表成功后不需要手工插入管理员或明文密码。

## 备选：服务器暂时没有 Go

已完成建库和账号授权后，在项目根目录直接导入同一份 SQL：

```bash
mysql --default-character-set=utf8mb4 -h 127.0.0.1 -P 3306 -u bigshot_app -p bigshot < server/internal/repository/migrations/001_initial.sql
mysql --default-character-set=utf8mb4 -h 127.0.0.1 -P 3306 -u bigshot_app -p bigshot < server/internal/repository/migrations/002_email_verification.sql
```

这两步创建13张业务和验证相关表。后续 Go 服务首次启动会创建 `schema_migrations` 并登记版本；SQL 使用 `CREATE TABLE IF NOT EXISTS`，不会重复插入业务数据。不要在 SQL 出错后手工登记迁移成功，也不要使用 `--force` 忽略错误。

## 本次准备与验证

已核对现有建表 SQL、Go 迁移入口和配置校验逻辑。本次新增的是云端操作说明，没有变更表结构；尚未连接云服务器执行，云端是否成功须以第5步的输出为准。
