export function StatsCard({ label, value }: { label: string; value: string | number }) {
  return (
    <section className="panel stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </section>
  );
}
