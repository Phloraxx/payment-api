import type { Ticket } from "../types";
import { StatusBadge } from "./StatusBadge";

export function DataTable({
  tickets,
  onMarkPaid,
  onCancel,
  onSelect,
}: {
  tickets: Ticket[];
  onMarkPaid: (id: string) => void;
  onCancel: (id: string) => void;
  onSelect: (ticket: Ticket) => void;
}) {
  return (
    <div className="panel table-wrap">
      <table>
        <thead>
          <tr>
            <th>Ticket</th>
            <th>Amount</th>
            <th>Status</th>
            <th>Sender</th>
            <th>RRN</th>
            <th>Created</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => (
            <tr key={ticket.ticketId}>
              <td>
                <button className="link" onClick={() => onSelect(ticket)}>
                  {ticket.ticketId}
                </button>
              </td>
              <td>₹{ticket.amount.toFixed(2)}</td>
              <td>
                <StatusBadge status={ticket.status} />
              </td>
              <td>{ticket.senderName ?? "-"}</td>
              <td>{ticket.rrn ?? "-"}</td>
              <td>{new Date(ticket.createdAt).toLocaleString()}</td>
              <td>
                <div className="row-actions">
                  <button disabled={ticket.status !== "pending"} onClick={() => onMarkPaid(ticket.ticketId)}>
                    Paid
                  </button>
                  <button disabled={ticket.status !== "pending"} onClick={() => onCancel(ticket.ticketId)}>
                    Cancel
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {tickets.length === 0 && (
            <tr>
              <td colSpan={7} className="empty">
                No tickets found.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
