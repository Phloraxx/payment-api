import type { LogEntry, PoolSnapshot, Settings, Ticket } from "../types";
import { api } from "./client";

export function listTickets(params = "") {
  return api<{ tickets: Ticket[] }>(`/api/admin/tickets${params}`);
}

export function getStats() {
  return api<Record<string, number>>("/api/admin/stats");
}

export function getPool() {
  return api<{ pools: PoolSnapshot[] }>("/api/admin/pool");
}

export function getLogs(params = "") {
  return api<{ logs: LogEntry[] }>(`/api/admin/logs${params}`);
}

export function markPaid(ticketId: string) {
  return api<Ticket>(`/api/admin/tickets/${ticketId}/mark-paid`, { method: "POST" });
}

export function cancelTicket(ticketId: string) {
  return api<Ticket>(`/api/admin/tickets/${ticketId}/cancel`, { method: "POST" });
}

export function createTestTicket(amount: number | string) {
  return api<Ticket>("/api/admin/test/ticket", { method: "POST", body: JSON.stringify({ amount }) });
}

export function simulateWebhook(sms: string) {
  return api<{ status: string; ticketId: string; action: string }>("/api/admin/test/webhook", { method: "POST", body: JSON.stringify({ sms }) });
}

export function fullSync() {
  return api<{ attempted: number; failed: number }>("/api/admin/sync/full", { method: "POST" });
}

export function getSettings() {
  return api<Settings>("/api/admin/settings");
}
