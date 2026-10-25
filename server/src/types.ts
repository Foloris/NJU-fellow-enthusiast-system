import type { Request } from "express";
import type { SafeUser } from "./db.js";

export type AuthUser = SafeUser;

export type AuthRequest = Request & {
  user?: AuthUser;
};

