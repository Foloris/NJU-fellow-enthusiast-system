import { Router } from "express";
import { db, transaction } from "../db.js";
import { optionalAuth, requireAuth } from "../auth.js";
import {
  applicationSchema,
  formatZodError,
  matePostSchema,
  reportSchema,
} from "../validation.js";
import type { AuthRequest } from "../types.js";
import { createReport } from "./reports.js";

export const matesRouter = Router();

type MateRow = {
  id: number;
  author_id: number;
  title: string;
  tags: string;
  details: string;
  image_url: string | null;
  location: string | null;
  contact: string;
  duration: string;
  notes: string | null;
  status: "published" | "removed";
  created_at: string;
  updated_at: string;
  author_username: string;
  author_nickname: string;
  author_avatar: string | null;
  application_count: number;
  my_application_status: string | null;
};

const mateSelect = `
  SELECT p.*, u.username AS author_username, u.nickname AS author_nickname,
         u.avatar AS author_avatar,
         (SELECT COUNT(*) FROM applications a WHERE a.mate_post_id = p.id) AS application_count,
         (SELECT a.status FROM applications a
          WHERE a.mate_post_id = p.id AND a.applicant_id = ? LIMIT 1) AS my_application_status
  FROM mate_posts p JOIN users u ON u.id = p.author_id
`;

function mapMate(
  row: MateRow,
  viewer?: { id: number; role: string },
) {
  let tags: string[] = [];
  try {
    tags = JSON.parse(row.tags) as string[];
  } catch {
    tags = [];
  }
  const ownerOrAdmin =
    viewer?.role === "admin" || viewer?.id === row.author_id;
  const contactVisible =
    ownerOrAdmin || row.my_application_status === "accepted";
  return {
    id: row.id,
    title: row.title,
    tags,
    details: row.details,
    imageUrl: row.image_url,
    location: row.location,
    contact: contactVisible ? row.contact : null,
    contactHint: contactVisible
      ? null
      : row.my_application_status === "accepted"
        ? "请通过私聊与对方联系"
        : "申请通过后可查看联系方式并私聊",
    duration: row.duration,
    notes: row.notes,
    status: row.status,
    createdAt: row.created_at,
    updatedAt: row.updated_at,
    applicationCount: Number(row.application_count),
    myApplicationStatus: row.my_application_status,
    isOwner: viewer?.id === row.author_id,
    author: {
      id: row.author_id,
      username: row.author_username,
      nickname: row.author_nickname,
      avatar: row.author_avatar,
    },
  };
}

matesRouter.get("/", optionalAuth, (req: AuthRequest, res) => {
  const q = String(req.query.q ?? "").trim();
  const tag = String(req.query.tag ?? "").trim();
  const mine = req.query.mine === "true";
  const page = Math.max(1, Number(req.query.page) || 1);
  const pageSize = Math.min(50, Math.max(1, Number(req.query.pageSize) || 12));
  const conditions: string[] = [];
  const filterParams: Array<string | number | null> = [];

  if (mine) {
    if (!req.user) {
      res.status(401).json({ message: "请先登录" });
      return;
    }
    conditions.push("p.author_id = ?");
    filterParams.push(req.user.id);
  } else {
    conditions.push("p.status = 'published'", "u.status = 'active'");
  }
  if (q) {
    conditions.push(
      "(p.title LIKE ? OR p.details LIKE ? OR p.tags LIKE ? OR p.location LIKE ?)",
    );
    filterParams.push(`%${q}%`, `%${q}%`, `%${q}%`, `%${q}%`);
  }
  if (tag) {
    conditions.push("p.tags LIKE ?");
    filterParams.push(`%\"${tag}\"%`);
  }
  const where = `WHERE ${conditions.join(" AND ")}`;
  const viewerId = req.user?.id ?? 0;
  const count = db
    .prepare(
      `SELECT COUNT(*) AS count FROM mate_posts p
       JOIN users u ON u.id = p.author_id ${where}`,
    )
    .get(...filterParams) as { count: number };
  const rows = db
    .prepare(
      `${mateSelect} ${where}
       ORDER BY p.created_at DESC LIMIT ? OFFSET ?`,
    )
    .all(
      viewerId,
      ...filterParams,
      pageSize,
      (page - 1) * pageSize,
    ) as MateRow[];
  res.json({
    items: rows.map((row) => mapMate(row, req.user)),
    total: count.count,
    page,
    pageSize,
  });
});

matesRouter.get("/:id", optionalAuth, (req: AuthRequest, res) => {
  const row = db
    .prepare(`${mateSelect} WHERE p.id = ?`)
    .get(req.user?.id ?? 0, Number(req.params.id)) as MateRow | undefined;
  if (!row) {
    res.status(404).json({ message: "搭子帖子不存在" });
    return;
  }
  const ownerOrAdmin =
    req.user?.role === "admin" || req.user?.id === row.author_id;
  if (row.status !== "published" && !ownerOrAdmin) {
    res.status(404).json({ message: "搭子帖子不存在或已被下架" });
    return;
  }
  res.json({ post: mapMate(row, req.user) });
});

matesRouter.post("/", requireAuth, (req: AuthRequest, res) => {
  const parsed = matePostSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  const data = parsed.data;
  const result = db
    .prepare(
      `INSERT INTO mate_posts
       (author_id, title, tags, details, image_url, location, contact, duration, notes)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    )
    .run(
      req.user!.id,
      data.title,
      JSON.stringify(data.tags),
      data.details,
      data.imageUrl || null,
      data.location || null,
      data.contact,
      data.duration,
      data.notes || null,
    );
  res.status(201).json({ id: Number(result.lastInsertRowid) });
});

matesRouter.patch("/:id", requireAuth, (req: AuthRequest, res) => {
  const post = db
    .prepare("SELECT id, author_id FROM mate_posts WHERE id = ?")
    .get(Number(req.params.id)) as
    | { id: number; author_id: number }
    | undefined;
  if (!post) {
    res.status(404).json({ message: "搭子帖子不存在" });
    return;
  }
  if (post.author_id !== req.user!.id && req.user!.role !== "admin") {
    res.status(403).json({ message: "只能修改自己发布的搭子" });
    return;
  }
  const parsed = matePostSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  const data = parsed.data;
  db.prepare(
    `UPDATE mate_posts SET
       title = ?, tags = ?, details = ?, image_url = ?, location = ?,
       contact = ?, duration = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
     WHERE id = ?`,
  ).run(
    data.title,
    JSON.stringify(data.tags),
    data.details,
    data.imageUrl || null,
    data.location || null,
    data.contact,
    data.duration,
    data.notes || null,
    post.id,
  );
  res.json({ message: "搭子帖子已更新" });
});

matesRouter.delete("/:id", requireAuth, (req: AuthRequest, res) => {
  const post = db
    .prepare("SELECT author_id FROM mate_posts WHERE id = ?")
    .get(Number(req.params.id)) as { author_id: number } | undefined;
  if (!post) {
    res.status(404).json({ message: "搭子帖子不存在" });
    return;
  }
  if (post.author_id !== req.user!.id && req.user!.role !== "admin") {
    res.status(403).json({ message: "只能删除自己发布的搭子" });
    return;
  }
  db.prepare("DELETE FROM mate_posts WHERE id = ?").run(Number(req.params.id));
  res.json({ message: "搭子帖子已删除" });
});

matesRouter.post("/:id/apply", requireAuth, (req: AuthRequest, res) => {
  const parsed = applicationSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  const post = db
    .prepare(
      "SELECT id, author_id, status FROM mate_posts WHERE id = ?",
    )
    .get(Number(req.params.id)) as
    | { id: number; author_id: number; status: string }
    | undefined;
  if (!post || post.status !== "published") {
    res.status(404).json({ message: "搭子帖子不存在或已停止招募" });
    return;
  }
  if (post.author_id === req.user!.id) {
    res.status(400).json({ message: "不能申请自己发布的搭子" });
    return;
  }
  const existing = db
    .prepare(
      "SELECT id, status FROM applications WHERE mate_post_id = ? AND applicant_id = ?",
    )
    .get(post.id, req.user!.id) as
    | { id: number; status: string }
    | undefined;
  if (existing) {
    res.status(409).json({ message: "你已经申请过这个搭子" });
    return;
  }
  const result = db
    .prepare(
      `INSERT INTO applications (mate_post_id, applicant_id, message)
       VALUES (?, ?, ?)`,
    )
    .run(post.id, req.user!.id, parsed.data.message);
  res.status(201).json({
    id: Number(result.lastInsertRowid),
    message: "申请已发送，等待发布者回应",
  });
});

matesRouter.get(
  "/:id/applications",
  requireAuth,
  (req: AuthRequest, res) => {
    const post = db
      .prepare("SELECT author_id FROM mate_posts WHERE id = ?")
      .get(Number(req.params.id)) as { author_id: number } | undefined;
    if (!post) {
      res.status(404).json({ message: "搭子帖子不存在" });
      return;
    }
    if (post.author_id !== req.user!.id && req.user!.role !== "admin") {
      res.status(403).json({ message: "只有发布者可以查看申请" });
      return;
    }
    const rows = db
      .prepare(
        `SELECT a.*, u.username, u.nickname, u.avatar, u.bio,
                c.id AS conversation_id
         FROM applications a
         JOIN users u ON u.id = a.applicant_id
         LEFT JOIN conversations c ON c.application_id = a.id
         WHERE a.mate_post_id = ?
         ORDER BY CASE a.status WHEN 'pending' THEN 0 ELSE 1 END, a.created_at DESC`,
      )
      .all(Number(req.params.id)) as Record<string, unknown>[];
    res.json({
      items: rows.map((row) => ({
        id: row.id,
        message: row.message,
        status: row.status,
        createdAt: row.created_at,
        respondedAt: row.responded_at,
        conversationId: row.conversation_id,
        applicant: {
          id: row.applicant_id,
          username: row.username,
          nickname: row.nickname,
          avatar: row.avatar,
          bio: row.bio,
        },
      })),
    });
  },
);

matesRouter.get("/applications/mine", requireAuth, (req: AuthRequest, res) => {
  const scope = req.query.scope === "received" ? "received" : "sent";
  const condition =
    scope === "received" ? "p.author_id = ?" : "a.applicant_id = ?";
  const rows = db
    .prepare(
      `SELECT a.*, p.title AS post_title, p.status AS post_status,
              p.author_id, requester.nickname AS applicant_nickname,
              owner.nickname AS owner_nickname,
              c.id AS conversation_id
       FROM applications a
       JOIN mate_posts p ON p.id = a.mate_post_id
       JOIN users requester ON requester.id = a.applicant_id
       JOIN users owner ON owner.id = p.author_id
       LEFT JOIN conversations c ON c.application_id = a.id
       WHERE ${condition}
       ORDER BY a.created_at DESC`,
    )
    .all(req.user!.id) as Record<string, unknown>[];
  res.json({
    items: rows.map((row) => ({
      id: row.id,
      message: row.message,
      status: row.status,
      createdAt: row.created_at,
      post: {
        id: row.mate_post_id,
        title: row.post_title,
        status: row.post_status,
      },
      applicantNickname: row.applicant_nickname,
      ownerNickname: row.owner_nickname,
      conversationId: row.conversation_id,
    })),
  });
});

matesRouter.patch(
  "/applications/:id",
  requireAuth,
  (req: AuthRequest, res) => {
    const action = req.body.action as "accept" | "reject";
    if (!["accept", "reject"].includes(action)) {
      res.status(400).json({ message: "无效的处理方式" });
      return;
    }
    const application = db
      .prepare(
        `SELECT a.*, p.author_id
         FROM applications a JOIN mate_posts p ON p.id = a.mate_post_id
         WHERE a.id = ?`,
      )
      .get(Number(req.params.id)) as
      | {
          id: number;
          mate_post_id: number;
          applicant_id: number;
          author_id: number;
          status: string;
        }
      | undefined;
    if (!application) {
      res.status(404).json({ message: "申请不存在" });
      return;
    }
    if (
      application.author_id !== req.user!.id &&
      req.user!.role !== "admin"
    ) {
      res.status(403).json({ message: "只有发布者可以处理申请" });
      return;
    }
    if (application.status !== "pending") {
      res.status(409).json({ message: "这条申请已经处理过了" });
      return;
    }

    let conversationId: number | null = null;
    transaction(() => {
      db.prepare(
        `UPDATE applications
         SET status = ?, responded_at = CURRENT_TIMESTAMP WHERE id = ?`,
      ).run(action === "accept" ? "accepted" : "rejected", application.id);
      if (action === "accept") {
        const existing = db
          .prepare("SELECT id FROM conversations WHERE application_id = ?")
          .get(application.id) as { id: number } | undefined;
        if (existing) {
          conversationId = existing.id;
        } else {
          const created = db
            .prepare(
              `INSERT INTO conversations (application_id, user_a_id, user_b_id)
               VALUES (?, ?, ?)`,
            )
            .run(
              application.id,
              application.author_id,
              application.applicant_id,
            );
          conversationId = Number(created.lastInsertRowid);
        }
      }
    });
    res.json({
      message: action === "accept" ? "已同意申请，可以开始私聊" : "已拒绝申请",
      conversationId,
    });
  },
);

matesRouter.post(
  "/:id/report",
  requireAuth,
  (req: AuthRequest, res) => {
    const parsed = reportSchema.safeParse({
      ...req.body,
      targetType: "mate",
      targetId: Number(req.params.id),
    });
    if (!parsed.success) {
      res.status(400).json({ message: formatZodError(parsed.error) });
      return;
    }
    const created = createReport(req.user!.id, parsed.data);
    if (!created.ok) {
      res.status(created.status).json({ message: created.error });
      return;
    }
    res.status(201).json({ message: "举报已提交，管理员会尽快处理" });
  },
);
