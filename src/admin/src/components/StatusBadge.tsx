import type { Ticket } from "../types";

const labels: Record<Ticket["status"], string> = {
  pending: "Pending",
  paid: "Paid",
  cancelled: "Cancelled",
  expired: "Expired",
};

export function StatusBadge({ status }: { status: Ticket["status"] }) {
  return <span className={`badge ${status}`}>{labels[status]}</span>;
}
