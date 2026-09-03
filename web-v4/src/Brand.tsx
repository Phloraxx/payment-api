export function Brand({ compact = false, subtitle }: { compact?: boolean; subtitle?: string }) {
  return <div className={`paygate-brand ${compact ? "compact" : ""}`}>
    <PayGateMark />
    {!compact && <div className="paygate-wordmark"><span>Pay</span><strong>Gate</strong>{subtitle && <small>{subtitle}</small>}</div>}
  </div>;
}

export function PayGateMark({ className = "" }: { className?: string }) {
  return <svg className={`paygate-mark ${className}`} viewBox="0 0 64 64" aria-hidden="true">
    <defs>
      <linearGradient id="pg-mark-gradient" x1="25" y1="4" x2="54" y2="58" gradientUnits="userSpaceOnUse"><stop stopColor="#58ff72"/><stop offset=".46" stopColor="#13e99f"/><stop offset="1" stopColor="#08a9c9"/></linearGradient>
      <linearGradient id="pg-bar-gradient" x1="15" y1="26" x2="42" y2="42" gradientUnits="userSpaceOnUse"><stop stopColor="#5cff6c"/><stop offset="1" stopColor="#04b9c2"/></linearGradient>
    </defs>
    <path d="M31.2 4.2c-4.2 0-7.2 1.4-10.8 3.5L10.2 13.6C6.3 15.8 4 19.9 4 24.4v24.1c0 5.6 5.8 9 10.5 6.1l8.3-5.2c3.4-2.1 5.4-5.7 5.4-9.7V25.9c0-3.2 1.7-6.1 4.5-7.7l5.8-3.4c3.6-2.1 3-7.5-.8-9-2.2-.9-4.3-1.6-6.5-1.6Z" fill="#0b2029"/>
    <path d="M30.8 4.2c2.8 0 5.5.8 8 2.2l13.6 8C57.1 17.2 60 22.2 60 27.7v8.5c0 5.6-2.9 10.7-7.7 13.5l-13 7.7c-4.8 2.8-10.8-.6-10.8-6.2v-6.5c0-3.4 1.8-6.5 4.7-8.2l13.1-7.6c2.1-1.2 2.1-4.2 0-5.5l-13-7.5c-3-1.7-4.8-4.9-4.8-8.3V6.5c0-1.3.9-2.3 2.3-2.3Z" fill="url(#pg-mark-gradient)"/>
    <circle cx="14" cy="36.2" r="3.2" fill="url(#pg-bar-gradient)"/><rect x="18.4" y="26.2" width="18.6" height="5.2" rx="2.6" fill="url(#pg-bar-gradient)"/><rect x="16.5" y="33.5" width="24" height="5.2" rx="2.6" fill="url(#pg-bar-gradient)"/><rect x="18.2" y="40.8" width="18.9" height="5.2" rx="2.6" fill="url(#pg-bar-gradient)"/>
  </svg>;
}
