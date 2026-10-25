import { DatabaseSync } from "node:sqlite";
import bcrypt from "bcryptjs";
import { mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const currentDir = path.dirname(fileURLToPath(import.meta.url));
export const serverDir = path.resolve(currentDir, "..");
const dataDir = path.join(serverDir, "data");
export const uploadDir = path.join(serverDir, "uploads");

mkdirSync(dataDir, { recursive: true });
mkdirSync(uploadDir, { recursive: true });

export const db = new DatabaseSync(path.join(dataDir, "campus.db"));
db.exec("PRAGMA journal_mode = WAL");
db.exec("PRAGMA foreign_keys = ON");

db.exec(`
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    nickname TEXT NOT NULL,
    avatar TEXT,
    bio TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'banned')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS exchange_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('share', 'news')),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    image_url TEXT,
    status TEXT NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'removed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS mate_posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    tags TEXT NOT NULL DEFAULT '[]',
    details TEXT NOT NULL,
    image_url TEXT,
    location TEXT,
    contact TEXT NOT NULL,
    duration TEXT NOT NULL,
    notes TEXT,
    status TEXT NOT NULL DEFAULT 'published' CHECK (status IN ('published', 'removed')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL REFERENCES exchange_posts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS likes (
    post_id INTEGER NOT NULL REFERENCES exchange_posts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, user_id)
  );

  CREATE TABLE IF NOT EXISTS favorites (
    post_id INTEGER NOT NULL REFERENCES exchange_posts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (post_id, user_id)
  );

  CREATE TABLE IF NOT EXISTS applications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mate_post_id INTEGER NOT NULL REFERENCES mate_posts(id) ON DELETE CASCADE,
    applicant_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at TEXT,
    UNIQUE (mate_post_id, applicant_id)
  );

  CREATE TABLE IF NOT EXISTS conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    application_id INTEGER NOT NULL UNIQUE REFERENCES applications(id) ON DELETE CASCADE,
    user_a_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
  );

  CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    read_at TEXT
  );

  CREATE TABLE IF NOT EXISTS reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL CHECK (target_type IN ('post', 'mate', 'user', 'message')),
    target_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    snapshot TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'resolved', 'dismissed')),
    action TEXT,
    admin_note TEXT,
    resolved_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TEXT
  );

  CREATE INDEX IF NOT EXISTS idx_exchange_status_created
    ON exchange_posts(status, created_at DESC);
  CREATE INDEX IF NOT EXISTS idx_mate_status_created
    ON mate_posts(status, created_at DESC);
  CREATE INDEX IF NOT EXISTS idx_applications_mate
    ON applications(mate_post_id, status);
  CREATE INDEX IF NOT EXISTS idx_messages_conversation
    ON messages(conversation_id, id DESC);
  CREATE INDEX IF NOT EXISTS idx_reports_status
    ON reports(status, created_at DESC);
`);

export type UserRow = {
  id: number;
  username: string;
  password_hash: string;
  nickname: string;
  avatar: string | null;
  bio: string;
  role: "user" | "admin";
  status: "active" | "banned";
  created_at: string;
};

export type SafeUser = Omit<UserRow, "password_hash" | "status"> & {
  status?: "active" | "banned";
};

export function transaction<T>(work: () => T): T {
  db.exec("BEGIN IMMEDIATE");
  try {
    const result = work();
    db.exec("COMMIT");
    return result;
  } catch (error) {
    db.exec("ROLLBACK");
    throw error;
  }
}

export function toSafeUser(user: UserRow, includeStatus = false): SafeUser {
  const safe = {
    id: user.id,
    username: user.username,
    nickname: user.nickname,
    avatar: user.avatar,
    bio: user.bio,
    role: user.role,
    created_at: user.created_at,
  };
  return includeStatus ? { ...safe, status: user.status } : safe;
}

export function seedDatabase() {
  const adminUsername = process.env.ADMIN_USERNAME || "admin";
  const adminPassword = process.env.ADMIN_PASSWORD || "CampusAdmin123!";
  const existingAdmin = db
    .prepare("SELECT id FROM users WHERE role = 'admin' LIMIT 1")
    .get() as { id: number } | undefined;

  if (!existingAdmin) {
    db.prepare(
      `INSERT INTO users
       (username, password_hash, nickname, avatar, bio, role)
       VALUES (?, ?, ?, ?, ?, 'admin')`,
    ).run(
      adminUsername,
      bcrypt.hashSync(adminPassword, 12),
      "校园管理站",
      null,
      "维护社区秩序，也分享校园里的新鲜事。",
    );
  }

  if (process.env.SEED_DEMO === "false") return;

  const userCount = db.prepare("SELECT COUNT(*) AS count FROM users").get() as {
    count: number;
  };
  if (userCount.count > 1) return;

  const insertUser = db.prepare(
    `INSERT INTO users
     (username, password_hash, nickname, avatar, bio, role)
     VALUES (?, ?, ?, ?, ?, 'user')`,
  );
  const passwordHash = bcrypt.hashSync("Demo123!", 10);
  const linId = Number(
    insertUser.run(
      "linzhi",
      passwordHash,
      "林知夏",
      null,
      "爱散步、摄影和一切有风的下午。",
    ).lastInsertRowid,
  );
  const zhouId = Number(
    insertUser.run(
      "zhouyu",
      passwordHash,
      "周屿",
      null,
      "计算机系大三，正在努力把生活过得松弛一点。",
    ).lastInsertRowid,
  );
  const chenId = Number(
    insertUser.run(
      "chenmu",
      passwordHash,
      "陈木",
      null,
      "羽毛球新手，图书馆常驻选手。",
    ).lastInsertRowid,
  );
  const admin = db
    .prepare("SELECT id FROM users WHERE role = 'admin' LIMIT 1")
    .get() as { id: number };

  const insertExchange = db.prepare(
    `INSERT INTO exchange_posts
     (author_id, kind, title, content, image_url)
     VALUES (?, ?, ?, ?, ?)`,
  );
  const share1 = Number(
    insertExchange.run(
      linId,
      "share",
      "在仙林，把秋天装进周末的六个小时",
      "从北大楼出发，沿梧桐道慢慢走。没有赶路的任务，只带了一台相机和一杯还温热的拿铁。下午在草坪上坐了很久，看社团排练，也听陌生人弹吉他。\n\n如果你也喜欢这样不设目的地的散步，欢迎在评论区交换路线。",
      "/demo/autumn-walk.svg",
    ).lastInsertRowid,
  );
  const share2 = Number(
    insertExchange.run(
      zhouId,
      "share",
      "一个人自习也能有节奏：我的图书馆续命歌单",
      "最近在准备考试，摸索出一套自己的自习节奏：前 50 分钟专注，休息时去接水，再用 10 分钟整理刚才没想明白的地方。\n\n歌单以纯音乐为主，不会抢走注意力。也想听听大家压箱底的自习好物。",
      "/demo/library.svg",
    ).lastInsertRowid,
  );
  insertExchange.run(
    admin.id,
    "news",
    "校园社团开放日：本周六，来和 40+ 社团碰面",
    "本周六 10:00—16:00，中心广场将举行秋季社团开放日。现场包括社团展演、互动体验和招新咨询。参与活动无需报名，请根据现场指引有序排队。",
    "/demo/clubs.svg",
  );

  db.prepare(
    "INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
  ).run(share1, chenId, "北大楼到天文台的路线也很好走，傍晚光特别好看。");
  db.prepare(
    "INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
  ).run(share2, linId, "推荐试试图书馆四楼靠窗的位置，下午很安静。");
  db.prepare("INSERT INTO likes (post_id, user_id) VALUES (?, ?)").run(
    share1,
    zhouId,
  );
  db.prepare("INSERT INTO likes (post_id, user_id) VALUES (?, ?)").run(
    share1,
    chenId,
  );
  db.prepare("INSERT INTO likes (post_id, user_id) VALUES (?, ?)").run(
    share2,
    linId,
  );
  db.prepare("INSERT INTO favorites (post_id, user_id) VALUES (?, ?)").run(
    share1,
    zhouId,
  );

  const insertMate = db.prepare(
    `INSERT INTO mate_posts
     (author_id, title, tags, details, image_url, location, contact, duration, notes)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
  );
  insertMate.run(
    linId,
    "周末去紫金山看秋天，想找一位慢慢走路的同伴",
    JSON.stringify(["户外", "周末", "摄影"]),
    "计划周六早上八点从学校出发，走蒋王庙线路，全程不赶时间。我带了相机，可以在山顶一起拍拍照。路线有一定体力要求，希望你有基础运动习惯。",
    "/demo/hiking.svg",
    "紫金山 · 蒋王庙入口",
    "同意申请后私聊交换",
    "周六 08:00—15:00",
    "雨天顺延。",
  );
  insertMate.run(
    zhouId,
    "长期英语口语搭子，每周两次图书馆练习",
    JSON.stringify(["学习", "英语", "长期"]),
    "目标是摆脱哑巴英语，主题包括校园生活、技术学习和旅行。希望每周能在图书馆见面两次，每次四十分钟，互相纠正表达。水平不限，能坚持最重要。",
    null,
    "仙林校区图书馆",
    "同意申请后私聊交换",
    "持续四周",
    "如果时间合适，也可以在线上进行。",
  );
  insertMate.run(
    chenId,
    "今晚七点打羽毛球，缺一位双打搭档",
    JSON.stringify(["运动", "今晚", "羽毛球"]),
    "已经订好 19:00—20:30 的场地，想找一位能一起打双打的同学。我是新手，主要是轻松运动，不追求输赢。",
    "/demo/badminton.svg",
    "东区体育馆 3 号场",
    "同意申请后私聊交换",
    "今晚 19:00—20:30",
    "自带球拍，场地费 AA。",
  );
}
