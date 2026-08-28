type OperatorRecord = { email?: string };
type AuthResponse = { token?: string; record?: OperatorRecord };
type ErrorEnvelope = { message?: string; error?: { code?: string; message?: string }; data?: Record<string, { message?: string }> };

const TOKEN_KEY = "paygate_operator_token";
const EMAIL_KEY = "paygate_operator_email";
const listeners = new Set<() => void>();
let token = sessionStorage.getItem(TOKEN_KEY) ?? "";
let email = sessionStorage.getItem(EMAIL_KEY) ?? "";

function emit() { listeners.forEach((listener) => listener()); }
function save(nextToken: string, nextEmail: string) {
  token = nextToken.trim(); email = nextEmail.trim();
  if (token) sessionStorage.setItem(TOKEN_KEY, token); else sessionStorage.removeItem(TOKEN_KEY);
  if (email) sessionStorage.setItem(EMAIL_KEY, email); else sessionStorage.removeItem(EMAIL_KEY);
  emit();
}
export const auth = {
  get token() { return token; },
  get email() { return email; },
  get isValid() { return token.length > 0; },
  clear() { save("", ""); },
  subscribe(listener: () => void) { listeners.add(listener); return () => { listeners.delete(listener); }; },
};

async function parse<T>(response: Response): Promise<T> {
  const body = (await response.json().catch(() => ({}))) as ErrorEnvelope & T;
  if (!response.ok) {
    const fieldError = body.data ? Object.values(body.data).find((value) => value?.message)?.message : undefined;
    throw new Error(body.error?.message ?? fieldError ?? body.message ?? `Request failed with HTTP ${response.status}`);
  }
  return body as T;
}
export async function login(emailValue: string, password: string) {
  const response = await fetch("/api/collections/users/auth-with-password", { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ identity: emailValue.trim(), password }) });
  const body = await parse<AuthResponse>(response);
  if (!body.token) throw new Error("PayGate returned no operator token");
  save(body.token, body.record?.email ?? emailValue.trim());
}
export async function refreshAuth() {
  if (!token) return;
  const response = await fetch("/api/collections/users/auth-refresh", { method: "POST", headers: { Authorization: `Bearer ${token}`, Accept: "application/json" } });
  const body = await parse<AuthResponse>(response);
  if (!body.token) throw new Error("PayGate returned no refreshed token");
  save(body.token, body.record?.email ?? email);
}
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(path, { ...init, headers });
  try { return await parse<T>(response); }
  catch (error) { if (response.status === 401 && token) auth.clear(); throw error; }
}
