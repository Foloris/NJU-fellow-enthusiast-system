import { Router } from "express";
import { db } from "../db.js";
import { requireAuth } from "../auth.js";
import { broadcastToUsers } from "../realtime.js";
import type { AuthRequest } from "../types.js";

export const messagesRouter = Router();

function conversationForUser(conversationId: number, userId: number) {
  return db
    .prepare(
      `SELECT c.*,
              CASE WHEN c.user_a_id = ? THEN c.user_b_id ELSE c.user_a_id END AS other_id
       FROM conversations c
       WHERE c.id = ? AND (c.user_a_id = ? OR c.user_b_id = ?)`,
    )
    .get(userId, conversationId, userId, userId) as
    | {
        id: number;
        application_id: number;
        user_a_id: number;
        user_b_id: number;
        created_at: string;
        other_id: number;
      }
    | undefined;
}

messagesRouter.get("/", requireAuth, (req: AuthRequest, res) => {
  const rows = db
    .prepare(
      `SELECT c.*,
              CASE WHEN c.user_a_id = ? THEN c.user_b_id ELSE c.user_a_id END AS other_id,
              u.username AS other_username,
              u.nickname AS other_nickname,
              u.avatar AS other_avatar,
              u.status AS other_status,
              (SELECT content FROM messages m
               WHERE m.conversation_id = c.id ORDER BY m.id DESC LIMIT 1) AS last_message,
              (SELECT created_at FROM messages m
               WHERE m.conversation_id = c.id ORDER BY m.id DESC LIMIT 1) AS last_message_at,
              (SELECT COUNT(*) FROM messages m
               WHERE m.conversation_id = c.id
                 AND m.sender_id <> ? AND m.read_at IS NULL) AS unread_count,
              p.title AS post_title
       FROM conversations c
       JOIN users u ON u.id =
         (CASE WHEN c.user_a_id = ? THEN c.user_b_id ELSE c.user_a_id END)
       JOIN applications a ON a.id = c.application_id
       JOIN mate_posts p ON p.id = a.mate_post_id
       WHERE c.user_a_id = ? OR c.user_b_id = ?
       ORDER BY COALESCE(last_message_at, c.created_at) DESC`,
    )
    .all(
      req.user!.id,
      req.user!.id,
      req.user!.id,
      req.user!.id,
      req.user!.id,
    ) as Record<string, unknown>[];
  res.json({
    items: rows.map((row) => ({
      id: row.id,
      otherUser: {
        id: row.other_id,
        username: row.other_username,
        nickname: row.other_nickname,
        avatar: row.other_avatar,
        status: row.other_status,
      },
      lastMessage: row.last_message ?? "申请通过，发句问候吧",
      lastMessageAt: row.last_message_at ?? row.created_at,
      unreadCount: Number(row.unread_count),
      postTitle: row.post_title,
    })),
  });
});

messagesRouter.get(
  "/:id/messages",
  requireAuth,
  (req: AuthRequest, res) => {
    const conversation = conversationForUser(
      Number(req.params.id),
      req.user!.id,
    );
    if (!conversation) {
      res.status(404).json({ message: "会话不存在" });
      return;
    }
    const before = Number(req.query.before) || 0;
    const limit = Math.min(100, Math.max(1, Number(req.query.limit) || 50));
    const condition = before ? "AND m.id < ?" : "";
    const params: Array<string | number | null> = [conversation.id];
    if (before) params.push(before);
    params.push(limit);
    const rows = db
      .prepare(
        `SELECT m.*, u.nickname AS sender_nickname, u.avatar AS sender_avatar
         FROM messages m JOIN users u ON u.id = m.sender_id
         WHERE m.conversation_id = ? ${condition}
         ORDER BY m.id DESC LIMIT ?`,
      )
      .all(...params) as Record<string, unknown>[];
    res.json({
      items: rows.reverse().map((row) => ({
        id: row.id,
        content: row.content,
        createdAt: row.created_at,
        readAt: row.read_at,
        sender: {
          id: row.sender_id,
          nickname: row.sender_nickname,
          avatar: row.sender_avatar,
        },
      })),
      hasMore: rows.length === limit,
    });
  },
);

messagesRouter.post(
  "/:id/messages",
  requireAuth,
  (req: AuthRequest, res) => {
    const conversation = conversationForUser(
      Number(req.params.id),
      req.user!.id,
    );
    if (!conversation) {
      res.status(404).json({ message: "会话不存在" });
      return;
    }
    const content = String(req.body.content ?? "").trim();
    if (!content || content.length > 2000) {
      res.status(400).json({ message: "消息内容应为 1—2000 个字符" });
      return;
    }
    const result = db
      .prepare(
        "INSERT INTO messages (conversation_id, sender_id, content) VALUES (?, ?, ?)",
      )
      .run(conversation.id, req.user!.id, content);
    const row = db
      .prepare(
        `SELECT m.*, u.nickname AS sender_nickname, u.avatar AS sender_avatar
         FROM messages m JOIN users u ON u.id = m.sender_id WHERE m.id = ?`,
      )
      .get(result.lastInsertRowid) as Record<string, unknown>;
    const message = {
      id: row.id,
      content: row.content,
      createdAt: row.created_at,
      readAt: row.read_at,
      sender: {
        id: row.sender_id,
        nickname: row.sender_nickname,
        avatar: row.sender_avatar,
      },
    };
    broadcastToUsers([conversation.user_a_id, conversation.user_b_id], {
      type: "message",
      conversationId: conversation.id,
      message,
    });
    res.status(201).json({ message });
  },
);

messagesRouter.post("/:id/read", requireAuth, (req: AuthRequest, res) => {
  const conversation = conversationForUser(
    Number(req.params.id),
    req.user!.id,
  );
  if (!conversation) {
    res.status(404).json({ message: "会话不存在" });
    return;
  }
  const result = db
    .prepare(
      `UPDATE messages SET read_at = CURRENT_TIMESTAMP
       WHERE conversation_id = ? AND sender_id <> ? AND read_at IS NULL`,
    )
    .run(conversation.id, req.user!.id);
  broadcastToUsers([conversation.user_a_id, conversation.user_b_id], {
    type: "read",
    conversationId: conversation.id,
    readerId: req.user!.id,
  });
  res.json({ updated: result.changes });
});
