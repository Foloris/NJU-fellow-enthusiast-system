import { Router } from "express";
import { db } from "../db.js";
import { requireAdmin, requireAuth } from "../auth.js";
import { formatZodError, reportSchema } from "../validation.js";
import type { AuthRequest } from "../types.js";

type ReportInput = {
  targetType: "post" | "mate" | "user" | "message";
  targetId: number;
  reason: string;
  details: string;
};

export function createReport(reporterId: number, input: ReportInput) {
  let snapshot: Record<string, unknown> | null = null;

  if (input.targetType === "post") {
    snapshot = db
      .prepare(
        `SELECT p.id, p.title, p.content, p.status, p.author_id,
                u.nickname AS author_nickname
         FROM exchange_posts p JOIN users u ON u.id = p.author_id
         WHERE p.id = ?`,
      )
      .get(input.targetId) as Record<string, unknown> | undefined ?? null;
  }
  if (input.targetType === "mate") {
    snapshot = db
      .prepare(
        `SELECT p.id, p.title, p.details, p.status, p.author_id,
                u.nickname AS author_nickname
         FROM mate_posts p JOIN users u ON u.id = p.author_id
         WHERE p.id = ?`,
      )
      .get(input.targetId) as Record<string, unknown> | undefined ?? null;
  }
  if (input.targetType === "user") {
    if (input.targetId === reporterId) {
      return { ok: false, status: 400, error: "不能举报自己的账号" } as const;
    }
    snapshot = db
      .prepare(
        "SELECT id, username, nickname, bio, status FROM users WHERE id = ?",
      )
      .get(input.targetId) as Record<string, unknown> | undefined ?? null;
  }
  if (input.targetType === "message") {
    const message = db
      .prepare(
        `SELECT m.*, c.user_a_id, c.user_b_id, u.nickname AS sender_nickname
         FROM messages m
         JOIN conversations c ON c.id = m.conversation_id
         JOIN users u ON u.id = m.sender_id
         WHERE m.id = ?`,
      )
      .get(input.targetId) as
      | (Record<string, unknown> & {
          user_a_id: number;
          user_b_id: number;
        })
      | undefined;
    if (
      message &&
      message.user_a_id !== reporterId &&
      message.user_b_id !== reporterId
    ) {
      return {
        ok: false,
        status: 403,
        error: "不能举报不属于你的会话中的消息",
      } as const;
    }
    snapshot = message ?? null;
  }

  if (!snapshot) {
    return { ok: false, status: 404, error: "举报对象不存在" } as const;
  }

  const duplicate = db
    .prepare(
      `SELECT id FROM reports
       WHERE reporter_id = ? AND target_type = ? AND target_id = ?
         AND status = 'pending'`,
    )
    .get(reporterId, input.targetType, input.targetId);
  if (duplicate) {
    return {
      ok: false,
      status: 409,
      error: "你已举报过该内容，管理员正在处理中",
    } as const;
  }

  const result = db
    .prepare(
      `INSERT INTO reports
       (reporter_id, target_type, target_id, reason, details, snapshot)
       VALUES (?, ?, ?, ?, ?, ?)`,
    )
    .run(
      reporterId,
      input.targetType,
      input.targetId,
      input.reason,
      input.details,
      JSON.stringify(snapshot),
    );
  return { ok: true, id: Number(result.lastInsertRowid) } as const;
}

export const reportsRouter = Router();

reportsRouter.post("/", requireAuth, (req: AuthRequest, res) => {
  const parsed = reportSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  if (parsed.data.targetType === "post" || parsed.data.targetType === "mate") {
    res.status(400).json({ message: "请从对应的帖子页面提交举报" });
    return;
  }
  const created = createReport(req.user!.id, parsed.data);
  if (!created.ok) {
    res.status(created.status).json({ message: created.error });
    return;
  }
  res.status(201).json({ message: "举报已提交，管理员会尽快处理" });
});

function messageContext(messageId: number) {
  const message = db
    .prepare("SELECT conversation_id FROM messages WHERE id = ?")
    .get(messageId) as { conversation_id: number } | undefined;
  if (!message) return [];
  return db
    .prepare(
      `SELECT m.id, m.content, m.created_at, m.sender_id, u.nickname AS sender_nickname
       FROM messages m JOIN users u ON u.id = m.sender_id
       WHERE m.conversation_id = ?
         AND m.id >= ? AND m.id <= ?
       ORDER BY m.id ASC`,
    )
    .all(
      message.conversation_id,
      Math.max(1, messageId - 2),
      messageId + 2,
    ) as Record<string, unknown>[];
}

reportsRouter.get("/admin", requireAdmin, (req, res) => {
  const status = String(req.query.status ?? "pending");
  const validStatus = ["pending", "resolved", "dismissed"].includes(status)
    ? status
    : "pending";
  const rows = db
    .prepare(
      `SELECT r.*, u.nickname AS reporter_nickname, u.username AS reporter_username,
              a.nickname AS resolver_nickname
       FROM reports r
       JOIN users u ON u.id = r.reporter_id
       LEFT JOIN users a ON a.id = r.resolved_by
       WHERE r.status = ?
       ORDER BY r.created_at DESC`,
    )
    .all(validStatus) as Record<string, unknown>[];
  res.json({
    items: rows.map((row) => ({
      id: row.id,
      targetType: row.target_type,
      targetId: row.target_id,
      reason: row.reason,
      details: row.details,
      snapshot: JSON.parse(String(row.snapshot || "{}")),
      status: row.status,
      action: row.action,
      adminNote: row.admin_note,
      createdAt: row.created_at,
      resolvedAt: row.resolved_at,
      reporter: {
        id: row.reporter_id,
        nickname: row.reporter_nickname,
        username: row.reporter_username,
      },
      resolverNickname: row.resolver_nickname,
      messageContext:
        row.target_type === "message"
          ? messageContext(Number(row.target_id))
          : [],
    })),
  });
});
