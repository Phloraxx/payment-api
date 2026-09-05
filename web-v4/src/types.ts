export type PaymentStatus = "pending" | "paid" | "expired" | "cancelled";

export interface DailyVolume {
  date: string;
  amount_paise: number;
  payments: number;
}
export interface Overview {
  collected_today_paise: number;
  payments_today: number;
  paid_today: number;
  pending: number;
  expired_today: number;
  status_counts: Record<string, number>;
  volume: DailyVolume[];
  active_profile?: { id: string; label: string } | null;
  relay: {
    connected: boolean;
    name?: string;
    last_seen_at?: string;
    app_version?: string;
  };
  webhooks: {
    pending: number;
    exhausted: number;
    last_delivered_at?: string;
  };
}

export interface Payment {
  id: string;
  name: string;
  external_id?: string;
  metadata: Record<string, unknown> | null;
  requested_amount_paise: number;
  payable_amount_paise: number;
  adjustment_paise: number;
  collection_profile_id: string;
  upi_id_snapshot: string;
  payee_name_snapshot?: string;
  transaction_note: string;
  status: PaymentStatus;
  created_at: string;
  expires_at: string;
  grace_until: string;
  reuse_after: string;
  paid_at?: string | null;
  payer_name?: string;
  payer_upi_id?: string;
  internal_note?: string;
}
export interface PaymentList {
  items: Payment[];
  total: number;
  limit: number;
  offset: number;
}
export interface PaymentHistory {
  id: string;
  type: string;
  actor: string;
  summary: string;
  changes: Record<string, unknown> | null;
  created_at: string;
}
export interface WebhookDelivery {
  id: string;
  event_type: string;
  status: string;
  attempts: number;
  next_attempt_at?: string;
  last_http_status?: number;
  last_error?: string;
  created_at: string;
  delivered_at?: string;
}
export interface PaymentDetail {
  payment: Payment;
  history: PaymentHistory[];
  webhooks: WebhookDelivery[];
}

export interface ActivityEntry {
  at: string;
  kind: "payment" | "payment_detected" | "webhook" | string;
  status?: string;
  source?: string;
  title: string;
  payment_id?: string;
  amount_paise?: number;
  detail?: string;
}
export interface Profile {
  id: string;
  label: string;
  upi_id: string;
  payee_name?: string;
  parser: string;
  enabled: boolean;
  active: boolean;
  created_at: string;
  updated_at: string;
}
export interface WebhookSettings {
  enabled: boolean;
  endpoint: string;
  secret_configured: boolean;
}
export interface ApiKeyInfo {
  id: string;
  label: string;
  enabled: boolean;
  created_at: string;
  last_used_at?: string;
}
export interface DeviceInfo {
  id: string;
  name: string;
  enabled: boolean;
  enrolled_at: string;
  last_seen_at?: string;
  last_heartbeat_at?: string;
  app_version?: string;
  device_model?: string;
  android_version?: string;
  notification_access?: boolean;
  listener_connected?: boolean;
  battery_optimization_exempt?: boolean;
  power_save_mode?: boolean;
  background_restricted?: boolean;
  foreground_service?: boolean;
  pending_count?: number;
  failed_count?: number;
  last_successful_delivery_at?: string;
  last_client_error?: string;
}
export interface PairingSession {
  token: string;
  expires_at: string;
  replace_existing: boolean;
  pairing_url?: string;
}
