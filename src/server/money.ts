import { AppError } from "./errors.js";

export function toPaisa(value: number | string): number {
  const raw = String(value).trim();
  if (!/^\d+(\.\d{1,2})?$/.test(raw)) {
    throw new AppError("INVALID_AMOUNT", "Amount must be a positive number with up to two decimals.");
  }
  const [rupeesRaw, paisaRaw = ""] = raw.split(".");
  const rupees = Number.parseInt(rupeesRaw ?? "0", 10);
  const paisa = Number.parseInt(paisaRaw.padEnd(2, "0") || "0", 10);
  const total = rupees * 100 + paisa;
  if (!Number.isSafeInteger(total) || total <= 0) {
    throw new AppError("INVALID_AMOUNT", "Amount must be greater than zero.");
  }
  return total;
}

export function fromPaisa(value: number): number {
  return Number((value / 100).toFixed(2));
}

export function baseAmountFromPaisa(value: number): number {
  return Math.floor(value / 100) * 100;
}

export function decimalFromPaisa(value: number): number {
  return value % 100;
}
