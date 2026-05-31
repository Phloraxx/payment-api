import type { FastifyRequest } from "fastify";
import { AppError } from "../errors.js";
import type { AuthService } from "../services/auth.service.js";

export function requireAdmin(auth: AuthService, request: FastifyRequest): void {
  const signed = request.unsignCookie(request.cookies.token ?? "");
  const token = signed.valid ? signed.value : undefined;
  if (!auth.verifySession(token)) {
    throw new AppError("ADMIN_UNAUTHORIZED", "Admin session is missing or expired.");
  }
}

export function currentSessionToken(request: FastifyRequest): string | undefined {
  const signed = request.unsignCookie(request.cookies.token ?? "");
  return signed.valid ? signed.value : undefined;
}
