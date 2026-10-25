import type { NextFunction, Response } from "express";
import jwt from "jsonwebtoken";
import { db, toSafeUser, type UserRow } from "./db.js";
import type { AuthRequest, AuthUser } from "./types.js";

const JWT_SECRET =
  process.env.JWT_SECRET || "campus-fellow-local-development-secret-change-me";

export function signToken(user: AuthUser) {
  return jwt.sign(
    { id: user.id, username: user.username, role: user.role },
    JWT_SECRET,
    { expiresIn: "7d" },
  );
}

export function setAuthCookie(res: Response, user: AuthUser) {
  res.cookie("campus_token", signToken(user), {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    maxAge: 7 * 24 * 60 * 60 * 1000,
  });
}

export function clearAuthCookie(res: Response) {
  res.clearCookie("campus_token", {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

function tokenFromRequest(req: AuthRequest) {
  const cookieToken = req.cookies?.campus_token as string | undefined;
  if (cookieToken) return cookieToken;
  const header = req.headers.authorization;
  if (header?.startsWith("Bearer ")) return header.slice(7);
  return null;
}

export function optionalAuth(req: AuthRequest, _res: Response, next: NextFunction) {
  const token = tokenFromRequest(req);
  if (!token) return next();

  try {
    const payload = jwt.verify(token, JWT_SECRET) as { id: number };
    const user = db
      .prepare("SELECT * FROM users WHERE id = ? AND status = 'active'")
      .get(payload.id) as UserRow | undefined;
    if (user) req.user = toSafeUser(user);
  } catch {
    // Invalid or expired tokens are treated as unauthenticated.
  }
  next();
}

export function requireAuth(req: AuthRequest, res: Response, next: NextFunction) {
  optionalAuth(req, res, () => {
    if (!req.user) {
      res.status(401).json({ message: "请先登录后再继续" });
      return;
    }
    next();
  });
}

export function requireAdmin(req: AuthRequest, res: Response, next: NextFunction) {
  requireAuth(req, res, () => {
    if (req.user?.role !== "admin") {
      res.status(403).json({ message: "仅管理员可以访问" });
      return;
    }
    next();
  });
}

export function verifyToken(token: string): AuthUser | null {
  try {
    const payload = jwt.verify(token, JWT_SECRET) as { id: number };
    const user = db
      .prepare("SELECT * FROM users WHERE id = ? AND status = 'active'")
      .get(payload.id) as UserRow | undefined;
    return user ? toSafeUser(user) : null;
  } catch {
    return null;
  }
}

