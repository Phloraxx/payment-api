export type TicketStatus = "pending" | "paid" | "cancelled" | "expired";

export interface Ticket {
  id: string;
  amount: number;
  status: TicketStatus;
  base_amount: number;
  decimal_val: number;
  sender_name: string | null;
  rrn: string | null;
  upi_id: string | null;
  paid_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface TicketResponse {
  ticketId: string;
  amount: number;
  amountPaisa: number;
  status: TicketStatus;
  createdAt: string;
  paidAt?: string | null;
  senderName?: string | null;
  rrn?: string | null;
  upiId?: string | null;
}

export interface ParsedSms {
  ticketId?: string | undefined;
  amount: number;
  senderName?: string | undefined;
  rrn?: string | undefined;
  upiId?: string | undefined;
  method: "generic" | "bank";
}
