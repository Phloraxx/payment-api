import PocketBase from "pocketbase";

export const pb = new PocketBase(window.location.origin);
pb.autoCancellation(false);

type ErrorEnvelope = {
  message?: string;
  error?: { code?: string; message?: string };
};

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !(init.body instanceof FormData) && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (pb.authStore.token) {
    headers.set("Authorization", `Bearer ${pb.authStore.token}`);
  }
  const response = await fetch(path, { ...init, headers });
  const body = (await response.json().catch(() => ({}))) as ErrorEnvelope & T;
  if (!response.ok) {
    if (response.status === 401 && pb.authStore.token) pb.authStore.clear();
    const message = body.error?.message ?? body.message ?? `Request failed with HTTP ${response.status}`;
    throw new Error(message);
  }
  return body as T;
}
