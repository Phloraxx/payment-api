import { api } from "./client";
import { startAuthentication, startRegistration } from "@simplewebauthn/browser";

export function setupStatus() {
  return api<{ needs_setup: boolean; has_one_time_code: boolean }>("/api/admin/setup/status");
}

export function verifyCode(code: string) {
  return api<{ ok: boolean }>("/api/admin/setup/verify-code", { method: "POST", body: JSON.stringify({ code }) });
}

export async function registerPasskey() {
  const begin = await api<{ publicKey: Parameters<typeof startRegistration>[0]["optionsJSON"] }>("/api/admin/register/begin");
  const credential = await startRegistration({ optionsJSON: begin.publicKey });
  return api<{ ok: boolean }>("/api/admin/register/complete", { method: "POST", body: JSON.stringify({ credential }) });
}

export async function loginPasskey() {
  const begin = await api<{ requestId: string; publicKey: Parameters<typeof startAuthentication>[0]["optionsJSON"] }>("/api/admin/login/begin");
  const assertion = await startAuthentication({ optionsJSON: begin.publicKey });
  return api<{ ok: boolean }>("/api/admin/login/complete", {
    method: "POST",
    body: JSON.stringify({ requestId: begin.requestId, assertion }),
  });
}

export function session() {
  return api<{ ok: boolean }>("/api/admin/session");
}

export function logout() {
  return api<{ ok: boolean }>("/api/admin/logout", { method: "POST" });
}
