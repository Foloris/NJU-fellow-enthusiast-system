import { Router } from "express";
import bcrypt from "bcryptjs";
import { db, toSafeUser, type UserRow } from "../db.js";
import {
  clearAuthCookie,
  optionalAuth,
  requireAuth,
  setAuthCookie,
} from "../auth.js";
import {
  formatZodError,
  loginSchema,
  profileSchema,
  registerSchema,
} from "../validation.js";
import type { AuthRequest } from "../types.js";

export const authRouter = Router();

authRouter.post("/register", (req, res) => {
  const parsed = registerSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }

  const { username, password, nickname } = parsed.data;
  const duplicate = db
    .prepare("SELECT id FROM users WHERE lower(username) = lower(?)")
    .get(username);
  if (duplicate) {
    res.status(409).json({ message: "这个账号已经被使用" });
    return;
  }

  const result = db
    .prepare(
      `INSERT INTO users (username, password_hash, nickname)
       VALUES (?, ?, ?)`,
    )
    .run(username, bcrypt.hashSync(password, 12), nickname);
  const user = db
    .prepare("SELECT * FROM users WHERE id = ?")
    .get(result.lastInsertRowid) as UserRow;
  const safeUser = toSafeUser(user);
  setAuthCookie(res, safeUser);
  res.status(201).json({ user: safeUser });
});

authRouter.post("/login", (req, res) => {
  const parsed = loginSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }

  const user = db
    .prepare("SELECT * FROM users WHERE lower(username) = lower(?)")
    .get(parsed.data.username) as UserRow | undefined;
  if (!user || !bcrypt.compareSync(parsed.data.password, user.password_hash)) {
    res.status(401).json({ message: "账号或密码不正确" });
    return;
  }
  if (user.status === "banned") {
    res.status(403).json({ message: "该账号已被封禁，请联系管理员申诉" });
    return;
  }

  const safeUser = toSafeUser(user);
  setAuthCookie(res, safeUser);
  res.json({ user: safeUser });
});

authRouter.post("/logout", (_req, res) => {
  clearAuthCookie(res);
  res.json({ message: "已退出登录" });
});

authRouter.get("/me", optionalAuth, (req: AuthRequest, res) => {
  res.json({ user: req.user ?? null });
});

authRouter.patch("/me", requireAuth, (req: AuthRequest, res) => {
  const parsed = profileSchema.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ message: formatZodError(parsed.error) });
    return;
  }
  const data = parsed.data;
  db.prepare(
    "UPDATE users SET nickname = ?, bio = ?, avatar = ? WHERE id = ?",
  ).run(
    data.nickname,
    data.bio,
    data.avatar ?? req.user!.avatar ?? null,
    req.user!.id,
  );
  const user = db
    .prepare("SELECT * FROM users WHERE id = ?")
    .get(req.user!.id) as UserRow;
  res.json({ user: toSafeUser(user) });
});

