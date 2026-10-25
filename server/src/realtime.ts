import type { WebSocket } from "ws";

const clients = new Map<number, Set<WebSocket>>();

export function addRealtimeClient(userId: number, socket: WebSocket) {
  const set = clients.get(userId) ?? new Set<WebSocket>();
  set.add(socket);
  clients.set(userId, set);
  socket.on("close", () => {
    const current = clients.get(userId);
    current?.delete(socket);
    if (current?.size === 0) clients.delete(userId);
  });
}

export function broadcastToUsers(userIds: number[], payload: unknown) {
  const data = JSON.stringify(payload);
  for (const userId of userIds) {
    for (const socket of clients.get(userId) ?? []) {
      if (socket.readyState === socket.OPEN) socket.send(data);
    }
  }
}

