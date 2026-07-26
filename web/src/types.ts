import type { RecordModel } from "pocketbase";

export type Page = "dashboard" | "payments" | "sms" | "webhooks" | "settings";

export type Payment = RecordModel & {
  requested_amount: number;
  payable_amount: number;
  status: "pending" | "paid" | "expired" | "cancelled" | "late";
  expires_at: string;
  reuse_after: string;
  rrn: string;
  upi_id: string;
  payer_name: string;
  paid_at: string;
  external_id: string;
  metadata?: unknown;
};

export type Connector = {
  enabled: boolean;
  state: string;
  paired: boolean;
  connected: boolean;
  phoneResponsive: boolean;
  pairingMethod?: string;
  pairingEmoji?: string;
  accountEmail?: string;
  lastConnectedAt?: string;
  lastMessageAt?: string;
  lastError?: string;
};

export type DashboardData = {
  stats: Record<string, number>;
  connector: Connector;
};

export type PaymentCreateResponse = {
  id: string;
  requestedAmount: number;
  requestedAmountPaise: number;
  payableAmount: string;
  payableAmountPaise: number;
  status: string;
  expiresAt: string;
  paidAt: string | null;
  externalId: string;
  upiUri: string;
};
