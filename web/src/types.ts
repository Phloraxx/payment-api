import type { RecordModel } from "pocketbase";

export type Page = "dashboard" | "payments" | "reviews" | "reconciliation" | "sms" | "email" | "alerts" | "refunds" | "webhooks" | "audit" | "razorpay_test" | "settings";

export type Payment = RecordModel & {
  payment_account: "kotak" | "slice" | "paytm";
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


export type RelayStatus = {
  ready: boolean;
  enabledDevices: number;
  activeDevices: number;
  staleAfterSeconds: number;
  lastSeenAt?: string | null;
  lastHeartbeatAt?: string | null;
  lastEventAt?: string | null;
  lastMatchedAt?: string | null;
  recentErrorCount: number;
  pendingQueueCount: number;
  failedQueueCount: number;
};

export type RelayDevice = {
  id: string;
  deviceId: string;
  name: string;
  enabled: boolean;
  appVersion: string;
  androidVersion: string;
  deviceModel: string;
  lastSeenAt?: string | null;
  lastHeartbeatAt?: string | null;
  heartbeatGraceUntil?: string | null;
  notificationAccess: boolean;
  listenerConnected: boolean;
  pendingCount: number;
  failedCount: number;
  lastClientError?: string;
  lastDeliveryAt?: string | null;
  lastEventAt?: string | null;
  lastMatchedAt?: string | null;
  lastMatchedPaymentId?: string;
  recentErrorCount: number;
  active: boolean;
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
  relay?: RelayStatus;
  capacity?: CapacitySnapshot;
  openReviewCount?: number;
  openAlertCount?: number;
  backup?: BackupStatus;
};

export type PaymentCreateResponse = {
  id: string;
  paymentAccount: "kotak" | "slice" | "paytm";
  paymentAccountLabel: string;
  verificationMethod: "sms" | "email" | "notification";
  requestedAmount: number;
  requestedAmountPaise: number;
  payableAmount: string;
  payableAmountPaise: number;
  status: string;
  expiresAt: string;
  paidAt: string | null;
  externalId: string;
  paymentFlow: "upi_intent" | "merchant_qr" | "qr_only";
  upiUri?: string;
  qrPayload?: string;
};

export type ReviewCase = RecordModel & {
  kind: string;
  status: "open" | "resolved" | "dismissed";
  severity: string;
  sms_event: string;
  email_event: string;
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

export type RazorpayTestConfig = {
  enabled: boolean;
  keyId: string;
  displayName: string;
  mode: "test";
};

export type RazorpayTestOrder = RecordModel & {
  amount: number;
  currency: string;
  status: string;
  external_id: string;
  razorpay_order_id: string;
  razorpay_payment_id: string;
  provider_status: string;
  payment_method: string;
  amount_refunded: number;
  error: string;
  created_at: string;
  captured_at: string;
};

export type RazorpayTestOrderResponse = {
  id: string;
  amountPaise: number;
  currency: string;
  status: string;
  externalId: string;
  razorpayOrderId: string;
  razorpayPaymentId: string;
  providerStatus: string;
  paymentMethod: string;
  amountRefunded: number;
  error: string;
  createdAt: string;
  capturedAt: string;
  keyId: string;
  displayName: string;
};
