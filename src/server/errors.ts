export type ErrorCode =
  | "INVALID_AMOUNT"
  | "TICKET_NOT_FOUND"
  | "POOL_EXHAUSTED"
  | "RRN_DUPLICATE"
  | "AMOUNT_MISMATCH"
  | "TICKET_ALREADY_RESOLVED"
  | "WEBHOOK_UNAUTHORIZED"
  | "RATE_LIMITED"
  | "INTERNAL_ERROR";

const statusByCode: Record<ErrorCode, number> = {
  INVALID_AMOUNT: 400,
  TICKET_NOT_FOUND: 404,
  POOL_EXHAUSTED: 503,
  RRN_DUPLICATE: 409,
  AMOUNT_MISMATCH: 400,
  TICKET_ALREADY_RESOLVED: 409,
  WEBHOOK_UNAUTHORIZED: 401,
  RATE_LIMITED: 429,
  INTERNAL_ERROR: 500,
};

export class AppError extends Error {
  public readonly code: ErrorCode;
  public readonly statusCode: number;
  public readonly details: Record<string, unknown> | undefined;

  constructor(code: ErrorCode, message: string, details?: Record<string, unknown>) {
    super(message);
    this.code = code;
    this.statusCode = statusByCode[code];
    this.details = details;
  }
}

export function isSqliteUniqueError(error: unknown): boolean {
  return error instanceof Error && error.message.includes("UNIQUE constraint failed");
}
