import type { RecordModel } from "pocketbase";

export type Page = "dashboard" | "payments" | "reviews" | "reconciliation" | "sms" | "alerts" | "refunds" | "webhooks" | "audit" | "settings";

export type Payment = RecordModel & {
  requested_amount: number;
  payable_amount: number;
  status: "pending" | "paid" | "expired" | "cancelled" | "late";
  expires_at: string;
  reuse_after: string;
  resolved_at: string;
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

export type CapacityPool = {
  requestedAmountPaise: number;
  requestedAmount: string;
  pending: number;
  quarantined: number;
  blocked: number;
  available: number;
  utilizationPercent: number;
  level: "normal" | "warning" | "critical";
};

export type CapacitySnapshot = {
  pools: CapacityPool[];
  warningPools: number;
  criticalPools: number;
};

export type BackupStatus = {
  enabled: boolean;
  cron?: string;
  maxKeep: number;
  offsite: boolean;
  backupCount: number;
  latest?: { name: string; size: number; modTime: string };
  latestVerified: boolean;
  verificationError?: string;
  error?: string;
};

export type DashboardData = {
  stats: Record<string, number>;
  connector: Connector;
  capacity?: CapacitySnapshot;
  openReviewCount?: number;
  openAlertCount?: number;
  backup?: BackupStatus;
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

export type ReviewCase = RecordModel & {
  kind: string;
  status: "open" | "resolved" | "dismissed";
  severity: string;
  sms_event: string;
  reconciliation_entry: string;
  payment: string;
  candidate_payment_ids?: string[];
  reason: string;
  resolution: string;
  resolution_note: string;
  resolved_by: string;
  opened_at: string;
  resolved_at: string;
  expand?: Record<string, RecordModel>;
};

export type AlertRecord = RecordModel & {
  kind: string;
  status: "open" | "resolved";
  severity: string;
  dedupe_key: string;
  message: string;
  details?: unknown;
  occurrence_count: number;
  first_seen_at: string;
  last_seen_at: string;
  resolved_at: string;
  notification_status: string;
  notification_attempts: number;
  notification_last_error: string;
  notification_delivered_at: string;
};

export type ReconciliationRun = RecordModel & {
  filename: string;
  sha256: string;
  status: string;
  total_rows: number;
  matched_rows: number;
  unmatched_rows: number;
  duplicate_rows: number;
  conflict_rows: number;
  invalid_rows: number;
  error: string;
  started_at: string;
  completed_at: string;
};

export type RefundRecord = RecordModel & {
  payment: string;
  amount: number;
  status: string;
  reason: string;
  reference: string;
  external_id: string;
  requested_at: string;
  completed_at: string;
  expand?: Record<string, RecordModel>;
};
