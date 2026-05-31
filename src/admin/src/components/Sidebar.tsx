import { Activity, FlaskConical, Gauge, ListChecks, LogOut, ReceiptText, Settings } from "lucide-react";

const items = [
  { id: "overview", label: "Overview", icon: Activity },
  { id: "tickets", label: "Tickets", icon: ReceiptText },
  { id: "pool", label: "Decimal Pool", icon: Gauge },
  { id: "test", label: "Test Harness", icon: FlaskConical },
  { id: "logs", label: "Logs", icon: ListChecks },
  { id: "settings", label: "Settings", icon: Settings },
] as const;

export type PageId = (typeof items)[number]["id"];

export function Sidebar({ page, onPage, onLogout }: { page: PageId; onPage: (page: PageId) => void; onLogout: () => void }) {
  return (
    <aside className="sidebar">
      <div className="brand">Pay Console</div>
      <nav>
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <button key={item.id} className={page === item.id ? "active" : ""} onClick={() => onPage(item.id)}>
              <Icon size={18} />
              {item.label}
            </button>
          );
        })}
      </nav>
      <button className="logout" onClick={onLogout}>
        <LogOut size={18} />
        Logout
      </button>
    </aside>
  );
}
