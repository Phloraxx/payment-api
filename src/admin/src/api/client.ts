export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const body = options.body;
  const response = await fetch(path, {
    credentials: "include",
    headers: {
      ...(body ? { "Content-Type": "application/json" } : {}),
      ...(options.headers ?? {}),
    },
    ...options,
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => null);
    throw new Error(payload?.error?.message ?? `Request failed: ${response.status}`);
  }
  return response.json() as Promise<T>;
}
