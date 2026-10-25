import { Router } from "express";
import { db } from "../db.js";
import { optionalAuth, requireAuth } from "../auth.js";
import {
  exchangePostSchema,
  formatZodError,
  reportSchema,
} from "../validation.js";
import type { AuthRequest } from "../types.js";
import { createReport } from "./reports.js";

export const exchangeRouter = Router();

const postSelect = `
  SELECT
    p.*,
    u.username AS author_username,
    u.nickname AS author_nickname,
    u.avatar AS author_avatar,
    (SELECT COUNT(*) FROM likes l WHERE l.post_id = p.id) AS like_count,
    (SELECT COUNT(*) FROM favorites f WHERE f.post_id = p.id) AS favorite_count,
    (SELECT COUNT(*) FROM comments c WHERE c.post_id = p.id) AS comment_count,
    EXISTS(SELECT 1 FROM likes l WHERE l.post_id = p.id AND l.user_id = ?) AS liked,
    EXISTS(SELECT 1 FROM favorites f WHERE f.post_id = p.id AND f.user_id = ?) AS favorited
  FROM exchange_posts p
  JOIN users u ON u.id = p.author_id
`;

function mapPost(row: Record<string, unknown>) {
  return {
    id: row.id,
    kind: row.kind,
    title: row.title,
    content: row.content,
    imageUrl: row.image_url,
    status: row.status,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    likeCount: Number(row.like_count),
    favoriteCount: Number(row.favorite_count),
    commentCount: Number(row.comment_count),
    liked: Boolean(row.liked),
    favorited: Boolean(row.favorited),
    author: {
      id: row.author_id,
      username: row.author_username,
      nickname: row.author_nickname,
      avatar: row.author_avatar,
    },
  };
}

exchangeRouter.get("/posts", optionalAuth, (req: AuthRequest, res) => {
  const viewerId = req.user?.id ?? 0;
  const q = String(req.query.q ?? "").trim();
  const kind = String(req.query.kind ?? "").trim();
  const mine = req.query.mine === "true";
  const page = Math.max(1, Number(req.query.page) || 1);
  const pageSize = Math.min(50, Math.max(1, Number(req.query.pageSize) || 12));
  const conditions: string[] = [];
  const params: Array<string | number | null> = [viewerId, viewerId];

  if (!mine) conditions.push("p.status = 'published'");
  if (mine) {
    if (!req.user) {
      res.status(401).json({ message: "请先登录" });
      return;
    }
    conditions.push("p.author_id = ?");
    params.push(req.user.id);
    if (req.user.role !== "admin") conditions.push("u.status = 'active'");
  } else {
    conditions.push("u.status = 'active'");
  }
  if (q) {
    conditions.push("(p.title LIKE ? OR p.content LIKE ?)");
    params.push(`%${q}%`, `%${q}%`);
  }
  if (kind === "share" || kind === "news") {
    conditions.push("p.kind = ?");
    params.push(kind);
  }

  const where = conditions.length ? `WHERE ${conditions.join(" AND ")}` : "";
  const count = db
    .prepare(
      `SELECT COUNT(*) AS count FROM exchange_posts p
       JOIN users u ON u.id = p.author_id ${where}`,
    )
    .get(...params.slice(2)) as { count: number };
  const rows = db
    .prepare(
      `${postSelect} ${where}
       ORDER BY CASE p.kind WHEN 'news' THEN 0 ELSE 1 END, p.created_at DESC
       LIMIT ? OFFSET ?`,
    )
    .all(...params, pageSize, (page - 1) * pageSize) as Record<string, unknown>[];

  res.json({
    items: rows.map(mapPost),
    total: count.count,
    page,
    pageSize,
  });
});

exchangeRouter.get("/posts/:id", optionalAuth, (req: AuthRequest, res) => {
  const viewerId = req.user?.id ?? 0;
  const row = db
    .prepare(`${postSelect} WHERE p.id = ?`)
    .get(viewerId, viewerId, Number(req.params.id)) as
    | Record<string, unknown>
    | undefined;
  if (!row) {
    res.status(404).json({ message: "帖子不存在" });
    return;
  }
  const isOwner = req.user?.id === Number(row.author_id);
  const isAdmin = req.user?.role === "admin";
  if (row.status !== "published" && !isOwner && !isAdmin) {
    res.status(404).json({ message: "帖子不存在或已被下架" });
    return;
  }
  const comments = db
    .prepare(
      `SELECT c.*, u.username, u.nickname, u.avatar
       FROM comments c JOIN users u ON u.id = c.user_id
       WHERE c.post_id = ? ORDER BY c.created_at ASC`,
    )
    .all(Number(req.params.id)) as Record<string, unknown>[];

  res.json({
    post: mapPost(row),
    comments: comments.map((comment) => ({
      id: comment.id,
      content: comment.content,
      createdAt: comment.created_at,
      author: {
        id: comment.user_id,
        username: comment.username,
        nickname: comment.nickname,
        avatar: comment.avatar,
      },
    })),
  });
});

exchangeRouter.post("/posts", requireAuth, (req: AuthRequest, res) => {
  const parsed = exchangePostSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  if (parsed.data.kind === "news" && req.user!.role !== "admin") {
    res.status(403).json({ message: "只有管理员可以发布校园新闻" });
    return;
  }
  const result = db
    .prepare(
      `INSERT INTO exchange_posts
       (author_id, kind, title, content, image_url)
       VALUES (?, ?, ?, ?, ?)`,
    )
    .run(
      req.user!.id,
      parsed.data.kind,
      parsed.data.title,
      parsed.data.content,
      parsed.data.imageUrl || null,
    );
  res.status(201).json({ id: Number(result.lastInsertRowid) });
});

exchangeRouter.patch("/posts/:id", requireAuth, (req: AuthRequest, res) => {
  const post = db
    .prepare("SELECT * FROM exchange_posts WHERE id = ?")
    .get(Number(req.params.id)) as
    | { id: number; author_id: number; kind: "share" | "news" }
    | undefined;
  if (!post) {
    res.status(404).json({ message: "帖子不存在" });
    return;
  }
  if (post.author_id !== req.user!.id && req.user!.role !== "admin") {
    res.status(403).json({ message: "只能修改自己发布的帖子" });
    return;
  }
  const parsed = exchangePostSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  if (parsed.data.kind === "news" && req.user!.role !== "admin") {
    res.status(403).json({ message: "只有管理员可以发布校园新闻" });
    return;
  }
  db.prepare(
    `UPDATE exchange_posts
     SET kind = ?, title = ?, content = ?, image_url = ?, updated_at = CURRENT_TIMESTAMP
     WHERE id = ?`,
  ).run(
    parsed.data.kind,
    parsed.data.title,
    parsed.data.content,
    parsed.data.imageUrl || null,
    post.id,
  );
  res.json({ message: "帖子已更新" });
});

exchangeRouter.delete("/posts/:id", requireAuth, (req: AuthRequest, res) => {
  const post = db
    .prepare("SELECT author_id FROM exchange_posts WHERE id = ?")
    .get(Number(req.params.id)) as { author_id: number } | undefined;
  if (!post) {
    res.status(404).json({ message: "帖子不存在" });
    return;
  }
  if (post.author_id !== req.user!.id && req.user!.role !== "admin") {
    res.status(403).json({ message: "只能删除自己发布的帖子" });
    return;
  }
  db.prepare("DELETE FROM exchange_posts WHERE id = ?").run(
    Number(req.params.id),
  );
  res.json({ message: "帖子已删除" });
});

exchangeRouter.post(
  "/posts/:id/like",
  requireAuth,
  (req: AuthRequest, res) => {
    const postId = Number(req.params.id);
    const post = db
      .prepare("SELECT id FROM exchange_posts WHERE id = ? AND status = 'published'")
      .get(postId);
    if (!post) {
      res.status(404).json({ message: "帖子不存在或已被下架" });
      return;
    }
    const existing = db
      .prepare("SELECT 1 FROM likes WHERE post_id = ? AND user_id = ?")
      .get(postId, req.user!.id);
    if (existing) {
      db.prepare("DELETE FROM likes WHERE post_id = ? AND user_id = ?").run(
        postId,
        req.user!.id,
      );
    } else {
      db.prepare("INSERT INTO likes (post_id, user_id) VALUES (?, ?)").run(
        postId,
        req.user!.id,
      );
    }
    const count = db
      .prepare("SELECT COUNT(*) AS count FROM likes WHERE post_id = ?")
      .get(postId) as { count: number };
    res.json({ active: !existing, count: count.count });
  },
);

exchangeRouter.post(
  "/posts/:id/favorite",
  requireAuth,
  (req: AuthRequest, res) => {
    const postId = Number(req.params.id);
    const post = db
      .prepare("SELECT id FROM exchange_posts WHERE id = ? AND status = 'published'")
      .get(postId);
    if (!post) {
      res.status(404).json({ message: "帖子不存在或已被下架" });
      return;
    }
    const existing = db
      .prepare("SELECT 1 FROM favorites WHERE post_id = ? AND user_id = ?")
      .get(postId, req.user!.id);
    if (existing) {
      db.prepare(
        "DELETE FROM favorites WHERE post_id = ? AND user_id = ?",
      ).run(postId, req.user!.id);
    } else {
      db.prepare(
        "INSERT INTO favorites (post_id, user_id) VALUES (?, ?)",
      ).run(postId, req.user!.id);
    }
    const count = db
      .prepare("SELECT COUNT(*) AS count FROM favorites WHERE post_id = ?")
      .get(postId) as { count: number };
    res.json({ active: !existing, count: count.count });
  },
);

exchangeRouter.post(
  "/posts/:id/comments",
  requireAuth,
  (req: AuthRequest, res) => {
    const content = String(req.body.content ?? "").trim();
    if (!content || content.length > 500) {
      res.status(400).json({ message: "评论内容应为 1—500 个字符" });
      return;
    }
    const postId = Number(req.params.id);
    const post = db
      .prepare("SELECT id FROM exchange_posts WHERE id = ? AND status = 'published'")
      .get(postId);
    if (!post) {
      res.status(404).json({ message: "帖子不存在或已被下架" });
      return;
    }
    const result = db
      .prepare(
        "INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)",
      )
      .run(postId, req.user!.id, content);
    res.status(201).json({
      comment: {
        id: Number(result.lastInsertRowid),
        content,
        createdAt: new Date().toISOString(),
        author: req.user!,
      },
    });
  },
);

exchangeRouter.delete(
  "/comments/:id",
  requireAuth,
  (req: AuthRequest, res) => {
    const comment = db
      .prepare("SELECT user_id FROM comments WHERE id = ?")
      .get(Number(req.params.id)) as { user_id: number } | undefined;
    if (!comment) {
      res.status(404).json({ message: "评论不存在" });
      return;
    }
    if (comment.user_id !== req.user!.id && req.user!.role !== "admin") {
      res.status(403).json({ message: "不能删除这条评论" });
      return;
    }
    db.prepare("DELETE FROM comments WHERE id = ?").run(Number(req.params.id));
    res.json({ message: "评论已删除" });
  },
);

exchangeRouter.post(
  "/posts/:id/report",
  requireAuth,
  (req: AuthRequest, res) => {
    const body = reportSchema.safeParse({
      ...req.body,
      targetType: "post",
      targetId: Number(req.params.id),
    });
    if (!body.success) {
      res.status(400).json({ message: formatZodError(body.error) });
      return;
    }
    const created = createReport(req.user!.id, body.data);
    if (!created.ok) {
      res.status(created.status).json({ message: created.error });
      return;
    }
    res.status(201).json({ message: "举报已提交，管理员会尽快处理" });
  },
);
