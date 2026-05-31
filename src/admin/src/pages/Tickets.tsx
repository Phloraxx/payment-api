import { useCallback, useEffect, useState } from "react";
import { cancelTicket, listTickets, markPaid } from "../api/tickets";
import { DataTable } from "../components/DataTable";
import { StatusBadge } from "../components/StatusBadge";
import type { Ticket } from "../types";

export function Tickets() {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [selectedTicket, setSelectedTicket] = useState<Ticket | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 50;

  const load = useCallback(() => {
    const params = new URLSearchParams();
    if (query) params.set("q", query);
    if (status) params.set("status", status);
    params.set("limit", String(pageSize));
    params.set("offset", String((page - 1) * pageSize));
    void listTickets(`?${params.toString()}`).then((result) => setTickets(result.tickets));
  }, [query, status, page]);

  useEffect(load, [load]);

  const handleFilterChange = (setter: typeof setQuery | typeof setStatus) => (value: string) => {
    setter(value);
    setPage(1);
  };

  return (
    <div className="page">
      <div className="toolbar">
        <input
          value={query}
          onChange={(event) => handleFilterChange(setQuery)(event.target.value)}
          placeholder="Search tickets, sender, RRN"
        />
        <select value={status} onChange={(event) => handleFilterChange(setStatus)(event.target.value)}>
          <option value="">All statuses</option>
          <option value="pending">Pending</option>
          <option value="paid">Paid</option>
          <option value="expired">Expired</option>
          <option value="cancelled">Cancelled</option>
        </select>
        <button onClick={load}>Refresh</button>
        <button onClick={() => download("tickets.json", JSON.stringify(tickets, null, 2), "application/json")}>
          Export JSON
        </button>
        <button onClick={() => download("tickets.csv", toCsv(tickets), "text/csv")}>Export CSV</button>
      </div>
      <DataTable
        tickets={tickets}
        onSelect={(ticket) => setSelectedTicket(ticket)}
        onMarkPaid={(id) =>
          void markPaid(id).then(() => {
            if (selectedTicket?.ticketId === id) setSelectedTicket(null);
            load();
          })
        }
        onCancel={(id) =>
          void cancelTicket(id).then(() => {
            if (selectedTicket?.ticketId === id) setSelectedTicket(null);
            load();
          })
        }
      />
      <div className="pagination">
        <button disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
          Previous
        </button>
        <span>Page {page}</span>
        <button disabled={tickets.length < pageSize} onClick={() => setPage((p) => p + 1)}>
          Next
        </button>
      </div>
      {selectedTicket && (
        <div className="drawer-backdrop" onClick={() => setSelectedTicket(null)}>
          <section className="panel drawer" onClick={(e) => e.stopPropagation()}>
            <div className="drawer-header">
              <h2>Ticket Details</h2>
              <button className="close" onClick={() => setSelectedTicket(null)}>✕</button>
            </div>
            <div className="drawer-body">
              <dl>
                <dt>Ticket ID</dt>
                <dd>{selectedTicket.ticketId}</dd>
                <dt>Amount</dt>
                <dd>₹{selectedTicket.amount.toFixed(2)} ({selectedTicket.amountPaisa} paisa)</dd>
                <dt>Status</dt>
                <dd><StatusBadge status={selectedTicket.status} /></dd>
                <dt>Sender Name</dt>
                <dd>{selectedTicket.senderName ?? "-"}</dd>
                <dt>RRN</dt>
                <dd>{selectedTicket.rrn ?? "-"}</dd>
                <dt>UPI ID</dt>
                <dd>{selectedTicket.upiId ?? "-"}</dd>
                <dt>Created At</dt>
                <dd>{new Date(selectedTicket.createdAt).toLocaleString()}</dd>
                <dt>Paid At</dt>
                <dd>{selectedTicket.paidAt ? new Date(selectedTicket.paidAt).toLocaleString() : "-"}</dd>
                <dt>Expires At</dt>
                <dd>{selectedTicket.expiresAt ? new Date(selectedTicket.expiresAt).toLocaleString() : "-"}</dd>
              </dl>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}

function download(name: string, content: string, type: string) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  URL.revokeObjectURL(url);
}

function toCsv(tickets: Ticket[]) {
  const rows = [
    ["ticketId", "amount", "status", "senderName", "rrn", "upiId", "createdAt", "paidAt"],
    ...tickets.map((ticket) => [
      ticket.ticketId,
      ticket.amount.toFixed(2),
      ticket.status,
      ticket.senderName ?? "",
      ticket.rrn ?? "",
      ticket.upiId ?? "",
      ticket.createdAt,
      ticket.paidAt ?? "",
    ]),
  ];
  return rows.map((row) => row.map((cell) => `"${String(cell).replaceAll('"', '""')}"`).join(",")).join("\n");
}
