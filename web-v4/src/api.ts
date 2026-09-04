import type {
  ActivityEntry, ApiKeyInfo, DeviceInfo, Overview, PairingSession,
  Payment, PaymentDetail, PaymentList, Profile, WebhookSettings,
} from "./types";

export class ApiError extends Error {
  constructor(readonly status: number, readonly code: string, message: string) {
    super(message);
  }
}

type ErrorEnvelope = { error?: { code?: string; message?: string } };

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: "same-origin",
    cache: "no-store",
    headers: { Accept: "application/json", ...init.headers },
  });
  if (response.status === 204) return undefined as T;
  let body: unknown;
  try { body = await response.json(); }
  catch { throw new ApiError(response.status, "invalid_response", "PayGate returned an invalid response."); }
  if (!response.ok) {
    const envelope = body as ErrorEnvelope;
    const code = envelope.error?.code ?? "request_failed";
    const message = envelope.error?.message ?? "PayGate could not complete the request.";
    if (response.status === 401 && path !== "/admin/session") {
      window.dispatchEvent(new Event("paygate:unauthorized"));
    }
    throw new ApiError(response.status, code, message);
  }
  return body as T;
}

export async function login(password: string): Promise<void> {
  await request<{ expires_at: string }>("/admin/session", {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ password }),
  });
}
export function logout(): Promise<void> { return request("/admin/session", { method: "DELETE" }); }
export function getOverview(): Promise<Overview> { return request("/admin/overview"); }
export async function getActivity(limit = 100): Promise<ActivityEntry[]> {
  const body = await request<{ items: ActivityEntry[] }>(`/admin/activity?limit=${limit}`);
  return body.items;
}

export interface PaymentFilters {
  q?: string; status?: string; profile?: string; externalId?: string; limit?: number; offset?: number;
}
export function listPayments(filters: PaymentFilters = {}): Promise<PaymentList> {
  const q = new URLSearchParams();
  if (filters.q) q.set("q", filters.q);
  if (filters.status) q.set("status", filters.status);
  if (filters.profile) q.set("profile", filters.profile);
  if (filters.externalId) q.set("external_id", filters.externalId);
  q.set("limit", String(filters.limit ?? 50));
  q.set("offset", String(filters.offset ?? 0));
  return request(`/admin/payments?${q.toString()}`);
}
export function getPayment(id: string): Promise<PaymentDetail> {
  return request(`/admin/payments/${encodeURIComponent(id)}`);
}
export type PaymentEdit = Partial<Pick<Payment,
  "name" | "external_id" | "status" | "payer_name" | "payer_upi_id" | "paid_at" | "internal_note"
>> & { metadata?: Record<string, unknown> | null };
export async function editPayment(id: string, edit: PaymentEdit): Promise<Payment> {
  const body = await request<{ payment: Payment }>(`/admin/payments/${encodeURIComponent(id)}`, {
    method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(edit),
  });
  return body.payment;
}
export function retryWebhook(id: string): Promise<void> {
  return request(`/admin/webhooks/${encodeURIComponent(id)}/retry`, { method: "POST" });
}

export async function getProfiles(): Promise<Profile[]> {
  return (await request<{ items: Profile[] }>("/admin/profiles")).items;
}
export async function saveProfile(profile: Pick<Profile, "id" | "label" | "upi_id" | "payee_name" | "parser" | "enabled">): Promise<Profile> {
  return (await request<{ profile: Profile }>("/admin/profiles", {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(profile),
  })).profile;
}
export async function activateProfile(id: string): Promise<Profile> {
  return (await request<{ profile: Profile }>(`/admin/profiles/${encodeURIComponent(id)}/activate`, { method: "POST" })).profile;
}

export async function getWebhookSettings(): Promise<WebhookSettings> {
  return (await request<{ webhook: WebhookSettings }>("/admin/settings")).webhook;
}
export async function saveWebhook(endpoint: string, rotateSecret: boolean): Promise<{ webhook: WebhookSettings; signing_secret?: string }> {
  return request("/admin/settings/webhook", {
    method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ endpoint, rotate_secret: rotateSecret }),
  });
}
export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return request("/admin/settings/password", {
    method: "PATCH", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
}

export async function getApiKeys(): Promise<ApiKeyInfo[]> {
  return (await request<{ items: ApiKeyInfo[] }>("/admin/api-keys")).items;
}
export function createApiKey(label: string): Promise<{ id: string; label: string; secret: string; created_at: string }> {
  return request("/admin/api-keys", {
    method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ label }),
  });
}
export function revokeApiKey(id: string): Promise<void> {
  return request(`/admin/api-keys/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export async function getDevices(): Promise<DeviceInfo[]> {
  const body = await request<{ device: DeviceInfo | null; devices?: DeviceInfo[] }>("/admin/device");
  return body.devices ?? (body.device ? [body.device] : []);
}
export function createPairingSession(): Promise<PairingSession> {
  return request("/admin/device/pairing-session", {
    method: "POST", headers: { "Content-Type": "application/json" }, body: "{}",
  });
}
export function updateProfileDestination(id: string, upiId: string, payeeName: string): Promise<{ profile: import("./types").Profile }> {
  return request(`/admin/profiles/${encodeURIComponent(id)}/destination`, {
    method: "PATCH", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ upi_id: upiId.trim(), payee_name: payeeName.trim() }),
  });
}

export function revokeDevice(id: string): Promise<void> {
  return request(`/admin/device/${encodeURIComponent(id)}`, { method: "DELETE" });
}
