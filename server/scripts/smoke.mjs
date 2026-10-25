import WebSocket from "ws";

const base = process.env.SMOKE_BASE_URL || "http://localhost:3001";
const stamp = Date.now().toString().slice(-8);

class Client {
  cookie = "";

  async request(path, options = {}) {
    const headers = new Headers(options.headers);
    if (this.cookie) headers.set("cookie", this.cookie);
    if (options.body && !(options.body instanceof FormData)) {
      headers.set("content-type", "application/json");
    }
    const response = await fetch(`${base}${path}`, { ...options, headers });
    const setCookie = response.headers.getSetCookie?.() ?? [];
    if (setCookie[0]) this.cookie = setCookie[0].split(";")[0];
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      throw new Error(
        `${options.method || "GET"} ${path} -> ${response.status}: ${data.message || "unknown error"}`,
      );
    }
    return data;
  }

  post(path, body) {
    return this.request(path, {
      method: "POST",
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  }

  patch(path, body) {
    return this.request(path, {
      method: "PATCH",
      body: JSON.stringify(body),
    });
  }

  delete(path) {
    return this.request(path, { method: "DELETE" });
  }
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function waitForSocketMessage(socket) {
  return new Promise((resolve, reject) => {
    const timeout = setTimeout(
      () => reject(new Error("Timed out waiting for realtime message")),
      5000,
    );
    socket.on("message", (raw) => {
      const payload = JSON.parse(String(raw));
      if (payload.type === "message") {
        clearTimeout(timeout);
        resolve(payload);
      }
    });
    socket.on("error", reject);
  });
}

function openSocket(cookie) {
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(
      `${base.replace(/^http/, "ws")}/ws`,
      { headers: { cookie } },
    );
    socket.once("open", () => resolve(socket));
    socket.once("error", reject);
  });
}

const alice = new Client();
const bob = new Client();
const admin = new Client();
const sockets = [];

try {
  await alice.post("/api/auth/register", {
    username: `smoke_a_${stamp}`,
    password: "Smoke123!",
    nickname: "冒烟测试甲",
  });
  await bob.post("/api/auth/register", {
    username: `smoke_b_${stamp}`,
    password: "Smoke123!",
    nickname: "冒烟测试乙",
  });
  const adminLogin = await admin.post("/api/auth/login", {
    username: process.env.ADMIN_USERNAME || "admin",
    password: process.env.ADMIN_PASSWORD || "CampusAdmin123!",
  });
  assert(adminLogin.user.role === "admin", "Admin login did not return admin role");

  const share = await alice.post("/api/exchange/posts", {
    kind: "share",
    title: "自动化验收分享帖",
    content: "这是一条用于验证点赞、收藏、评论、举报和管理员下架流程的测试内容。",
    imageUrl: null,
  });
  assert(share.id, "Exchange post was not created");

  const like = await bob.post(`/api/exchange/posts/${share.id}/like`);
  assert(like.active && like.count === 1, "Like toggle failed");
  const favorite = await bob.post(`/api/exchange/posts/${share.id}/favorite`);
  assert(favorite.active && favorite.count === 1, "Favorite toggle failed");
  await bob.post(`/api/exchange/posts/${share.id}/comments`, {
    content: "评论流程正常。",
  });
  const shareDetail = await bob.request(`/api/exchange/posts/${share.id}`);
  assert(shareDetail.comments.length === 1, "Comment was not returned");

  const mate = await alice.post("/api/mates", {
    title: "自动化验收搭子邀请",
    tags: ["测试", "周末"],
    details: "用于验证申请、同意和实时私信链路的测试邀请，活动地点在校园公共区域。",
    imageUrl: null,
    location: "中心广场",
    contact: "仅测试使用",
    duration: "本周六下午",
    notes: "自动化测试",
  });
  assert(mate.id, "Mate post was not created");

  await bob.post(`/api/mates/${mate.id}/apply`, {
    message: "我希望参加这次测试活动。",
  });
  const applications = await alice.request(`/api/mates/${mate.id}/applications`);
  assert(applications.items.length === 1, "Application was not visible to owner");
  const application = applications.items[0];
  const accepted = await alice.patch(
    `/api/mates/applications/${application.id}`,
    { action: "accept" },
  );
  assert(accepted.conversationId, "Accepting application did not create conversation");

  const realtimeTarget = await openSocket(alice.cookie);
  sockets.push(realtimeTarget);
  const realtimeEvent = waitForSocketMessage(realtimeTarget);
  const sent = await bob.post(
    `/api/messages/${accepted.conversationId}/messages`,
    { content: "这是一条实时私信测试。" },
  );
  const pushed = await realtimeEvent;
  assert(
    pushed.message?.id === sent.message.id,
    "WebSocket did not push the new message",
  );
  const messages = await alice.request(
    `/api/messages/${accepted.conversationId}/messages`,
  );
  assert(messages.items.length === 1, "Message history was not returned");
  await alice.post(`/api/messages/${accepted.conversationId}/read`);

  await bob.post(`/api/exchange/posts/${share.id}/report`, {
    reason: "自动化举报测试",
    details: "验证管理员审核流程。",
  });
  await bob.post("/api/reports", {
    targetType: "message",
    targetId: sent.message.id,
    reason: "私信举报测试",
    details: "验证私信上下文审核。",
  });
  let reportList = await admin.request("/api/reports/admin?status=pending");
  const postReport = reportList.items.find(
    (item) => item.targetType === "post" && item.targetId === share.id,
  );
  const messageReport = reportList.items.find(
    (item) => item.targetType === "message" && item.targetId === sent.message.id,
  );
  assert(postReport, "Post report was not visible to admin");
  assert(messageReport?.messageContext?.length, "Message report context is missing");

  await admin.patch(`/api/admin/reports/${postReport.id}`, {
    status: "resolved",
    action: "remove_content",
    adminNote: "自动化验收下架",
  });
  const removed = await fetch(`${base}/api/exchange/posts/${share.id}`);
  assert(removed.status === 404, "Removed post is still publicly visible");
  await admin.patch(`/api/admin/content/exchange/${share.id}`, {
    status: "published",
  });

  await admin.patch(`/api/admin/reports/${messageReport.id}`, {
    status: "resolved",
    action: "delete_message",
    adminNote: "自动化验收删除私信",
  });

  const users = await admin.request("/api/admin/users");
  const aliceUser = users.items.find(
    (item) => item.username === `smoke_a_${stamp}`,
  );
  await admin.patch(`/api/admin/users/${aliceUser.id}`, { status: "banned" });
  const blocked = await fetch(`${base}/api/auth/me`, {
    headers: { cookie: alice.cookie },
  });
  const blockedSession = await blocked.json();
  assert(
    blockedSession.user === null,
    "Banned account still resolves as an active session",
  );
  const blockedAction = await fetch(`${base}/api/exchange/posts`, {
    method: "POST",
    headers: {
      cookie: alice.cookie,
      "content-type": "application/json",
    },
    body: JSON.stringify({
      kind: "share",
      title: "不应发布",
      content: "被封禁用户不应能够继续发布内容。",
    }),
  });
  assert(blockedAction.status === 401, "Banned account can still write data");
  await admin.patch(`/api/admin/users/${aliceUser.id}`, { status: "active" });

  await admin.delete(`/api/admin/content/exchange/${share.id}`);
  await admin.delete(`/api/admin/content/mate/${mate.id}`);

  console.log(
    JSON.stringify(
      {
        ok: true,
        checked: [
          "registration",
          "login",
          "exchange CRUD",
          "likes",
          "favorites",
          "comments",
          "mate applications",
          "accept application",
          "realtime messaging",
          "read receipts",
          "post reports",
          "message reports",
          "moderation",
          "content removal",
          "account banning",
        ],
      },
      null,
      2,
    ),
  );
} finally {
  for (const socket of sockets) socket.close();
}
