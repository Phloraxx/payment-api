import type { RawData, WebSocket } from "ws";
import type { Ticket } from "../../types/index.js";
import type { LoggerService } from "../services/logger.service.js";

type SocketSet = Set<WebSocket>;

export class WsManager {
  private readonly ticketRooms = new Map<string, SocketSet>();
  private readonly adminSockets = new Set<WebSocket>();
  private readonly heartbeatTimers = new Map<WebSocket, NodeJS.Timeout>();

  constructor(logger: LoggerService) {
    logger.onLog((entry) => this.broadcastAdmin({ type: "log_entry", entry }));
  }

  addTicketSocket(ticketId: string, socket: WebSocket): void {
    const room = this.ticketRooms.get(ticketId) ?? new Set<WebSocket>();
    room.add(socket);
    this.ticketRooms.set(ticketId, room);
    this.register(socket, () => room.delete(socket));
  }

  addAdminSocket(socket: WebSocket): void {
    this.adminSockets.add(socket);
    this.register(socket, () => this.adminSockets.delete(socket));
  }

  broadcastTicket(ticketId: string, payload: unknown): void {
    this.broadcast(this.ticketRooms.get(ticketId), payload);
  }

  broadcastTicketUpdate(action: string, ticket: Ticket): void {
    this.broadcastAdmin({ type: "ticket_update", action, ticket });
  }

  broadcastExpired(ticket: Ticket): void {
    this.broadcastTicket(ticket.id, { type: "expired", reason: "timeout", ticketId: ticket.id });
    this.broadcastTicketUpdate("expired", ticket);
  }

  shutdown(): void {
    const payload = { type: "shutdown", reason: "restart", reconnectMs: 3000 };
    for (const sockets of this.ticketRooms.values()) this.broadcast(sockets, payload);
    this.broadcast(this.adminSockets, payload);
    for (const timer of this.heartbeatTimers.values()) clearInterval(timer);
    this.heartbeatTimers.clear();
  }

  private broadcastAdmin(payload: unknown): void {
    this.broadcast(this.adminSockets, payload);
  }

  private broadcast(sockets: SocketSet | undefined, payload: unknown): void {
    if (!sockets) return;
    const message = JSON.stringify(payload);
    for (const socket of sockets) {
      if (socket.readyState === socket.OPEN) socket.send(message);
    }
  }

  private register(socket: WebSocket, cleanup: () => void): void {
    let alive = true;
    socket.on("pong", () => {
      alive = true;
    });
    socket.on("message", (_data: RawData) => {
      alive = true;
    });
    socket.on("close", () => {
      cleanup();
      const timer = this.heartbeatTimers.get(socket);
      if (timer) clearInterval(timer);
      this.heartbeatTimers.delete(socket);
    });
    const timer = setInterval(() => {
      if (!alive) {
        socket.terminate();
        return;
      }
      alive = false;
      socket.ping();
    }, 30_000);
    timer.unref();
    this.heartbeatTimers.set(socket, timer);
  }
}
