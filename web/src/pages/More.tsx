const groups = [
  { title: "Money operations", items: [
    { href: "#/refunds", title: "Refunds", copy: "Track refund requests and bank references." },
    { href: "#/reconciliation", title: "Reconciliation", copy: "Compare bank statements when an exception needs investigation." },
    { href: "#/razorpay_test", title: "Razorpay Test", copy: "Isolated provider test tools." },
  ]},
  { title: "Evidence & delivery", items: [
    { href: "#/sms", title: "SMS evidence", copy: "Inspect parsed bank-message evidence." },
    { href: "#/email", title: "Email evidence", copy: "Inspect parsed email evidence." },
    { href: "#/webhooks", title: "Webhook deliveries", copy: "Inspect outbound delivery attempts." },
  ]},
  { title: "System", items: [
    { href: "#/alerts", title: "Alerts", copy: "Operational alerts and delivery status." },
    { href: "#/audit", title: "Audit trail", copy: "Immutable history of operator actions." },
    { href: "#/settings", title: "Advanced settings", copy: "Connectors, backups, relay devices and migration controls." },
  ]},
];

export function More() {
  return <div className="more-layout">
    <section className="more-intro"><span>Advanced tools</span><h2>Everything else, when you need it.</h2><p>Daily payment work stays simple. Investigation, recovery and infrastructure controls live here without crowding the main workflow.</p></section>
    {groups.map((group) => <section className="tool-group" key={group.title}><h3>{group.title}</h3><div className="tool-grid">{group.items.map((item) => <a className="tool-card" href={item.href} key={item.href}><div><strong>{item.title}</strong><p>{item.copy}</p></div><span>→</span></a>)}</div></section>)}
  </div>;
}
