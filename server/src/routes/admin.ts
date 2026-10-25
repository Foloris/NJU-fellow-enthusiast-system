import { Router } from "express";
import { db, transaction, type UserRow, toSafeUser } from "../db.js";
import { requireAdmin } from "../auth.js";
import type { AuthRequest } from "../types.js";

export const adminRouter = Router();

adminRouter.use(requireAdmin);

adminRouter.get("/summary", (_req, res) => {
  const count = (sql: string) =>
    Number((db.prepare(sql).get() as { count: number }).count);
  res.json({
    users: count("SELECT COUNT(*) AS count FROM users WHERE role = 'user'"),
    bannedUsers: count(
      "SELECT COUNT(*) AS count FROM users WHERE status = 'banned'",
    ),
    exchangePosts: count(
      "SELECT COUNT(*) AS count FROM exchange_posts WHERE status = 'published'",
    ),
    matePosts: count(
      "SELECT COUNT(*) AS count FROM mate_posts WHERE status = 'published'",
    ),
    pendingReports: count(
      "SELECT COUNT(*) AS count FROM reports WHERE status = 'pending'",
    ),
    activeConversations: count("SELECT COUNT(*) AS count FROM conversations"),
  });
});

adminRouter.get("/users", (req, res) => {
  const q = String(req.query.q ?? "").trim();
  const condition = q
    ? "WHERE username LIKE ? OR nickname LIKE ?"
    : "WHERE role = 'user'";
  const params = q ? [`%${q}%`, `%${q}%`] : [];
  const rows = db
    .prepare(
      `SELECT * FROM users ${condition}
       ORDER BY CASE status WHEN 'banned' THEN 0 ELSE 1 END, created_at DESC`,
    )
    .all(...params) as UserRow[];
  res.json({
    items: rows.map((row) => ({
      ...toSafeUser(row, true),
      postCount: Number(
        (
          db
            .prepare(
              `SELECT (
                 (SELECT COUNT(*) FROM exchange_posts WHERE author_id = ?) +
                 (SELECT COUNT(*) FROM mate_posts WHERE author_id = ?)
               ) AS count`,
            )
            .get(row.id, row.id) as { count: number }
        ).count,
      ),
    })),
  });
});

adminRouter.patch("/users/:id", (req: AuthRequest, res) => {
  const userId = Number(req.params.id);
  if (userId === req.user!.id) {
    res.status(400).json({ message: "不能封禁当前管理员账号" });
    return;
  }
  const status = String(req.body.status);
  if (!["active", "banned"].includes(status)) {
    res.status(400).json({ message: "无效的账号状态" });
    return;
  }
  const user = db
    .prepare("SELECT * FROM users WHERE id = ?")
    .get(userId) as UserRow | undefined;
  if (!user) {
    res.status(404).json({ message: "用户不存在" });
    return;
  }
  if (user.role === "admin") {
    res.status(400).json({ message: "不能在这里修改管理员账号" });
    return;
  }
  transaction(() => {
    db.prepare("UPDATE users SET status = ? WHERE id = ?").run(status, userId);
    if (status === "banned") {
      db.prepare(
        "UPDATE exchange_posts SET status = 'removed' WHERE author_id = ?",
      ).run(userId);
      db.prepare(
        "UPDATE mate_posts SET status = 'removed' WHERE author_id = ?",
      ).run(userId);
    }
  });
  res.json({ message: status === "banned" ? "账号已封禁" : "账号已解除封禁" });
});

adminRouter.get("/content", (req, res) => {
  const type = req.query.type === "mate" ? "mate" : "exchange";
  const q = String(req.query.q ?? "").trim();
  const search = q ? "AND (p.title LIKE ? OR p.content LIKE ?)" : "";
  const params = q ? [`%${q}%`, `%${q}%`] : [];
  if (type === "mate") {
    const rows = db
      .prepare(
        `SELECT p.id, p.title, p.details AS content, p.status, p.created_at,
                u.nickname AS author_nickname, u.username AS author_username
         FROM mate_posts p JOIN users u ON u.id = p.author_id
         WHERE 1 = 1 ${search.replaceAll("p.content", "p.details")}
         ORDER BY p.created_at DESC`,
      )
      .all(...params) as Record<string, unknown>[];
    res.json({
      type,
      items: rows.map((row) => ({
        id: row.id,
        title: row.title,
        content: row.content,
        status: row.status,
        createdAt: row.created_at,
        authorNickname: row.author_nickname,
        authorUsername: row.author_username,
      })),
    });
    return;
  }
  const rows = db
    .prepare(
      `SELECT p.id, p.title, p.content, p.kind, p.status, p.created_at,
              u.nickname AS author_nickname, u.username AS author_username
       FROM exchange_posts p JOIN users u ON u.id = p.author_id
       WHERE 1 = 1 ${search}
       ORDER BY p.created_at DESC`,
    )
    .all(...params) as Record<string, unknown>[];
  res.json({
    type,
    items: rows.map((row) => ({
      id: row.id,
      title: row.title,
      content: row.content,
      kind: row.kind,
      status: row.status,
      createdAt: row.created_at,
      authorNickname: row.author_nickname,
      authorUsername: row.author_username,
    })),
  });
});

adminRouter.patch("/content/:type/:id", (req, res) => {
  const type = req.params.type;
  const id = Number(req.params.id);
  const status = String(req.body.status);
  if (!["published", "removed"].includes(status)) {
    res.status(400).json({ message: "无效的内容状态" });
    return;
  }
  if (type !== "exchange" && type !== "mate") {
    res.status(400).json({ message: "无效的内容类型" });
    return;
  }
  const table = type === "exchange" ? "exchange_posts" : "mate_posts";
  const result = db
    .prepare(`UPDATE ${table} SET status = ? WHERE id = ?`)
    .run(status, id);
  if (result.changes === 0) {
    res.status(404).json({ message: "内容不存在" });
    return;
  }
  res.json({ message: status === "removed" ? "内容已下架" : "内容已恢复" });
});

adminRouter.delete("/content/:type/:id", (req, res) => {
  const type = req.params.type;
  if (type !== "exchange" && type !== "mate") {
    res.status(400).json({ message: "无效的内容类型" });
    return;
  }
  const table = type === "exchange" ? "exchange_posts" : "mate_posts";
  const result = db
    .prepare(`DELETE FROM ${table} WHERE id = ?`)
    .run(Number(req.params.id));
  if (result.changes === 0) {
    res.status(404).json({ message: "内容不存在" });
    return;
  }
  res.json({ message: "内容已删除" });
});

adminRouter.patch("/reports/:id", (req: AuthRequest, res) => {
  const reportId = Number(req.params.id);
  const status = String(req.body.status);
  const action = String(req.body.action ?? "none");
  const adminNote = String(req.body.adminNote ?? "").slice(0, 1000);
  if (!["resolved", "dismissed"].includes(status)) {
    res.status(400).json({ message: "无效的处理状态" });
    return;
  }
  const report = db
    .prepare("SELECT * FROM reports WHERE id = ?")
    .get(reportId) as
    | {
        id: number;
        target_type: "post" | "mate" | "user" | "message";
        target_id: number;
        status: string;
        snapshot: string;
      }
    | undefined;
  if (!report) {
    res.status(404).json({ message: "举报记录不存在" });
    return;
  }
  if (report.status !== "pending") {
    res.status(409).json({ message: "这条举报已经处理过了" });
    return;
  }
  if (
    ![
      "none",
      "remove_content",
      "delete_content",
      "ban_user",
      "delete_message",
    ].includes(action)
  ) {
    res.status(400).json({ message: "无效的处理方式" });
    return;
  }

  let targetOwnerId: number | null = null;
  if (report.target_type === "post" || report.target_type === "mate") {
    const table = report.target_type === "post" ? "exchange_posts" : "mate_posts";
    const target = db
      .prepare(`SELECT author_id FROM ${table} WHERE id = ?`)
      .get(report.target_id) as { author_id: number } | undefined;
    targetOwnerId = target?.author_id ?? null;
  }
  if (report.target_type === "user") targetOwnerId = report.target_id;
  if (report.target_type === "message") {
    const target = db
      .prepare(
        `SELECT c.user_a_id, c.user_b_id, m.sender_id
         FROM messages m JOIN conversations c ON c.id = m.conversation_id
         WHERE m.id = ?`,
      )
      .get(report.target_id) as
      | { user_a_id: number; user_b_id: number; sender_id: number }
      | undefined;
    targetOwnerId = target?.sender_id ?? null;
  }

  transaction(() => {
    if (action === "remove_content") {
      if (report.target_type === "post" || report.target_type === "mate") {
        const table =
          report.target_type === "post" ? "exchange_posts" : "mate_posts";
        db.prepare(`UPDATE ${table} SET status = 'removed' WHERE id = ?`).run(
          report.target_id,
        );
      }
    }
    if (action === "delete_content") {
      if (report.target_type === "post") {
        db.prepare("DELETE FROM exchange_posts WHERE id = ?").run(
          report.target_id,
        );
      }
      if (report.target_type === "mate") {
        db.prepare("DELETE FROM mate_posts WHERE id = ?").run(report.target_id);
      }
    }
    if (action === "delete_message" && report.target_type === "message") {
      db.prepare("DELETE FROM messages WHERE id = ?").run(report.target_id);
    }
    if (action === "ban_user" && targetOwnerId) {
      db.prepare("UPDATE users SET status = 'banned' WHERE id = ?").run(
        targetOwnerId,
      );
      db.prepare(
        "UPDATE exchange_posts SET status = 'removed' WHERE author_id = ?",
      ).run(targetOwnerId);
      db.prepare(
        "UPDATE mate_posts SET status = 'removed' WHERE author_id = ?",
      ).run(targetOwnerId);
    }
    db.prepare(
      `UPDATE reports
       SET status = ?, action = ?, admin_note = ?, resolved_by = ?,
           resolved_at = CURRENT_TIMESTAMP
       WHERE id = ?`,
    ).run(status, action, adminNote, req.user!.id, report.id);
  });
  res.json({ message: status === "dismissed" ? "举报已驳回" : "举报已处理" });
});
