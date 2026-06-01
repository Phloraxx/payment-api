export interface Ticket {
  ticketId: string;
  amount: number;
  amountPaisa: number;
  status: "pending" | "paid" | "cancelled" | "expired";
  expiresAt: string | null;
  createdAt: string;
  paidAt?: string | null;
  senderName?: string | null;
  rrn?: string | null;
  upiId?: string | null;
}

export interface LogEntry {
  id: number;
  level: "info" | "warn" | "error" | "debug";
  message: string;
  meta: string | null;
  created_at: string;
}

export interface Settings {
  port: number;
  host: string;
  publicBaseUrl: string;
  rpId: string;
  ticketTtlMinutes: number;
  upiId: string;
  upiPayeeName: string;
  appwriteEnabled: boolean;
  appwriteEndpoint: string | null;
}

export interface PoolSnapshot {
  baseAmount: number;
  free: number;
  pending: number;
  paidReserved: number;
  reusable: number;
  nextFree?: number;
  freeAmounts: number[];
}
