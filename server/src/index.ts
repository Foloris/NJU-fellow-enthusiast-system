import "dotenv/config";
import cookieParser from "cookie-parser";
import express, {
  type NextFunction,
  type Request,
  type Response,
} from "express";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import multer from "multer";
import { randomUUID } from "node:crypto";
import { WebSocketServer } from "ws";
import { db, seedDatabase, serverDir, uploadDir } from "./db.js";
import { optionalAuth, requireAuth, verifyToken } from "./auth.js";
import { addRealtimeClient } from "./realtime.js";
import { authRouter } from "./routes/auth.js";
import { exchangeRouter } from "./routes/exchange.js";
import { matesRouter } from "./routes/mates.js";
import { messagesRouter } from "./routes/messages.js";
import { reportsRouter } from "./routes/reports.js";
import { adminRouter } from "./routes/admin.js";
import type { AuthRequest } from "./types.js";

seedDatabase();

const app = express();
const port = Number(process.env.PORT) || 3001;

app.disable("x-powered-by");
app.use(express.json({ limit: "1mb" }));
app.use(express.urlencoded({ extended: false }));
app.use(cookieParser());
app.use(optionalAuth);

app.get("/api/health", (_req, res) => {
  res.json({ ok: true, time: new Date().toISOString() });
});

app.use("/api/auth", authRouter);
app.use("/api/exchange", exchangeRouter);
app.use("/api/mates", matesRouter);
app.use("/api/messages", messagesRouter);
app.use("/api/reports", reportsRouter);
app.use("/api/admin", adminRouter);

const storage = multer.diskStorage({
  destination: (_req, _file, callback) => callback(null, uploadDir),
  filename: (_req, file, callback) => {
    const extension =
      path.extname(file.originalname).toLowerCase().replace(/[^.\w]/g, "") ||
      ".jpg";
    callback(null, `${Date.now()}-${randomUUID()}${extension}`);
  },
});
const upload = multer({
  storage,
  limits: { fileSize: 5 * 1024 * 1024 },
  fileFilter: (_req, file, callback) => {
    if (!["image/jpeg", "image/png", "image/webp", "image/gif"].includes(file.mimetype)) {
      callback(new Error("仅支持 JPG、PNG、WebP 或 GIF 图片"));
      return;
    }
    callback(null, true);
  },
});

app.post(
  "/api/uploads",
  requireAuth,
  upload.single("image"),
  (req: AuthRequest, res) => {
    if (!req.file) {
      res.status(400).json({ message: "请选择图片" });
      return;
    }
    res.status(201).json({ url: `/uploads/${req.file.filename}` });
  },
);

app.use("/uploads", express.static(uploadDir, { maxAge: "7d" }));

app.get("/api/users/:id", (req, res) => {
  const user = db
    .prepare(
      `SELECT id, username, nickname, avatar, bio, role, created_at
       FROM users WHERE id = ? AND status = 'active'`,
    )
    .get(Number(req.params.id)) as Record<string, unknown> | undefined;
  if (!user) {
    res.status(404).json({ message: "用户不存在" });
    return;
  }
  const counts = db
    .prepare(
      `SELECT
        (SELECT COUNT(*) FROM exchange_posts
         WHERE author_id = ? AND status = 'published') AS exchanges,
        (SELECT COUNT(*) FROM mate_posts
         WHERE author_id = ? AND status = 'published') AS mates`,
    )
    .get(Number(user.id), Number(user.id)) as {
    exchanges: number;
    mates: number;
  };
  res.json({
    user: {
      id: user.id,
      username: user.username,
      nickname: user.nickname,
      avatar: user.avatar,
      bio: user.bio,
      role: user.role,
      createdAt: user.created_at,
      exchangeCount: counts.exchanges,
      mateCount: counts.mates,
    },
  });
});

const clientDist = path.resolve(serverDir, "../client/dist");
app.use(express.static(clientDist));
app.get("*splat", (req, res, next) => {
  if (req.path.startsWith("/api/") || req.path.startsWith("/uploads/")) {
    next();
    return;
  }
  res.sendFile(path.join(clientDist, "index.html"), (error) => {
    if (error) next();
  });
});

app.use((_req, res) => {
  res.status(404).json({ message: "接口不存在" });
});

app.use(
  (error: Error, _req: Request, res: Response, _next: NextFunction) => {
    console.error(error);
    const message =
      error instanceof multer.MulterError && error.code === "LIMIT_FILE_SIZE"
        ? "图片不能超过 5MB"
        : error.message || "服务器发生错误";
    res.status(400).json({ message });
  },
);

const server = createServer(app);
const wss = new WebSocketServer({ noServer: true });

server.on("upgrade", (request, socket, head) => {
  if (!request.url?.startsWith("/ws")) {
    socket.destroy();
    return;
  }
  const cookies = request.headers.cookie
    ?.split(";")
    .reduce<Record<string, string>>((acc, raw) => {
      const [key, ...parts] = raw.trim().split("=");
      acc[key] = decodeURIComponent(parts.join("="));
      return acc;
    }, {});
  const user = cookies?.campus_token
    ? verifyToken(cookies.campus_token)
    : null;
  if (!user) {
    socket.write("HTTP/1.1 401 Unauthorized\r\n\r\n");
    socket.destroy();
    return;
  }
  wss.handleUpgrade(request, socket, head, (webSocket) => {
    wss.emit("connection", webSocket, request, user.id);
  });
});

wss.on(
  "connection",
  (socket: import("ws").WebSocket, _request: Request, userId: number) => {
    addRealtimeClient(userId, socket);
    socket.send(JSON.stringify({ type: "connected" }));
    socket.on("error", () => socket.terminate());
  },
);

server.listen(port, () => {
  console.log(`Campus Fellow API running at http://localhost:${port}`);
});
