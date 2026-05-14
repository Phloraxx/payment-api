export const ADMIN_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Payment Admin</title>
  <style>
    :root {
      --bg-app: #050505;
      --bg-card: rgba(255,255,255,0.03);
      --bg-card-hover: rgba(255,255,255,0.06);
      --bg-input: rgba(255,255,255,0.05);
      --bg-input-focus: rgba(255,255,255,0.09);
      --bg-modal: rgba(18,18,18,0.95);
      --bg-header: rgba(5,5,5,0.75);
      --border-default: rgba(255,255,255,0.08);
      --border-bright: rgba(255,255,255,0.15);
      --border-focus: rgba(10,132,255,0.6);
      --text-primary: #fcfcfc;
      --text-secondary: rgba(235,235,245,0.65);
      --text-tertiary: rgba(235,235,245,0.35);
      --blue: #0a84ff;
      --green: #30d158;
      --orange: #ff9f0a;
      --red: #ff453a;
      --purple: #bf5af2;
      --teal: #64d2ff;
      --font-ui: -apple-system, BlinkMacSystemFont, "SF Pro Display", "SF Pro Text", system-ui, "Helvetica Neue", Arial, sans-serif;
      --font-mono: "SF Mono", ui-monospace, "Cascadia Code", "Fira Code", Menlo, Monaco, Consolas, monospace;
      --r-sm: 8px; --r-md: 12px; --r-lg: 16px; --r-xl: 24px; --r-2xl: 32px; --r-full: 9999px;
      --shadow-sm: 0 4px 12px rgba(0,0,0,0.25);
      --shadow-md: 0 12px 32px rgba(0,0,0,0.40);
      --shadow-lg: 0 24px 64px rgba(0,0,0,0.50);
      --shadow-xl: 0 40px 100px rgba(0,0,0,0.60);
      --ease-spring: cubic-bezier(0.34,1.56,0.64,1);
      --ease-out: cubic-bezier(0.16,1,0.3,1);
      --ease-default: cubic-bezier(0.25,0.46,0.45,0.94);
    }
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    * { scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.15) transparent; }
    ::-webkit-scrollbar { width: 5px; height: 5px; }
    ::-webkit-scrollbar-track { background: transparent; }
    ::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.13); border-radius: 3px; }
    ::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.22); }
    html, body { height: 100%; }
    body {
      font-family: var(--font-ui);
      color: var(--text-primary);
      background:
        radial-gradient(circle at 10% 20%, rgba(10,132,255,0.08) 0%, transparent 40%),
        radial-gradient(circle at 90% 10%, rgba(191,90,242,0.07) 0%, transparent 40%),
        radial-gradient(circle at 80% 80%, rgba(48,209,88,0.06) 0%, transparent 50%),
        radial-gradient(circle at 20% 80%, rgba(255,159,10,0.05) 0%, transparent 50%),
        var(--bg-app);
      background-attachment: fixed;
      min-height: 100vh;
      -webkit-font-smoothing: antialiased;
    }

    /* ── LOGIN ── */
    #login-screen {
      display: flex; align-items: center; justify-content: center;
      min-height: 100vh;
    }
    .login-card {
      width: 360px;
      background: rgba(20,20,22,0.92);
      backdrop-filter: blur(40px) saturate(180%);
      border: 1px solid var(--border-bright);
      border-top-color: rgba(255,255,255,0.22);
      border-radius: var(--r-2xl);
      padding: 40px 36px;
      box-shadow: var(--shadow-xl);
    }
    .login-logo { font-size: 28px; font-weight: 700; letter-spacing: -0.5px; margin-bottom: 6px; }
    .login-sub { color: var(--text-secondary); font-size: 14px; margin-bottom: 32px; }
    .login-label { font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 8px; }
    .login-input {
      width: 100%; padding: 12px 14px; font-size: 15px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default);
      border-radius: var(--r-md); color: var(--text-primary); outline: none;
      transition: border-color 0.2s, background 0.2s, box-shadow 0.2s;
    }
    .login-input:focus { background: var(--bg-input-focus); border-color: var(--border-focus); box-shadow: 0 0 0 3px rgba(10,132,255,0.25); }
    .login-btn {
      width: 100%; margin-top: 20px; padding: 13px; font-size: 15px; font-weight: 600;
      background: var(--blue); color: #fff; border: none; border-radius: var(--r-md);
      cursor: pointer; transition: all 0.2s;
      box-shadow: 0 2px 8px rgba(10,132,255,0.3);
    }
    .login-btn:hover { opacity: 0.9; transform: translateY(-1px); box-shadow: 0 4px 12px rgba(10,132,255,0.4); }
    .login-btn:active { transform: scale(0.98) translateY(0); box-shadow: 0 2px 4px rgba(10,132,255,0.3); }
    .login-err { color: var(--red); font-size: 13px; margin-top: 10px; min-height: 18px; text-align: center; }
    @keyframes shake {
      0%,100% { transform: translateX(0); }
      15% { transform: translateX(-7px); }
      30% { transform: translateX(6px); }
      45% { transform: translateX(-5px); }
      60% { transform: translateX(4px); }
      75% { transform: translateX(-3px); }
    }
    .login-card.shake { animation: shake 0.45s ease; }

    /* ── DASHBOARD ── */
    #dashboard { display: flex; flex-direction: column; min-height: 100vh; }

    /* Header */
    header {
      position: sticky; top: 0; z-index: 100;
      background: var(--bg-header);
      backdrop-filter: blur(30px) saturate(160%);
      border-bottom: 1px solid var(--border-default);
      padding: 0 24px;
      display: flex; align-items: center; gap: 16px; height: 56px;
    }
    .header-logo { font-size: 16px; font-weight: 700; letter-spacing: -0.3px; margin-right: 8px; white-space: nowrap; }
    .nav-tabs { display: flex; gap: 2px; background: rgba(255,255,255,0.05); border-radius: var(--r-md); padding: 3px; }
    .nav-tab {
      padding: 5px 14px; font-size: 13px; font-weight: 500; border-radius: var(--r-sm);
      cursor: pointer; color: var(--text-secondary); transition: all 0.18s; border: none; background: none;
    }
    .nav-tab.active { background: rgba(255,255,255,0.12); color: var(--text-primary); }
    .nav-tab:hover:not(.active) { color: var(--text-primary); background: rgba(255,255,255,0.06); }
    .header-spacer { flex: 1; }
    @keyframes pulse {
      0% { box-shadow: 0 0 0 0 rgba(48,209,88,0.4); }
      70% { box-shadow: 0 0 0 6px rgba(48,209,88,0); }
      100% { box-shadow: 0 0 0 0 rgba(48,209,88,0); }
    }
    .rt-dot {
      width: 8px; height: 8px; border-radius: 50%; transition: background 0.3s;
    }
    .rt-dot.connected { background: var(--green); box-shadow: 0 0 6px var(--green); animation: pulse 2s infinite; }
    .rt-dot.disconnected { background: rgba(255,255,255,0.2); animation: none; }
    .rt-label { font-size: 11px; color: var(--text-tertiary); margin-right: 4px; }
    .header-btn {
      padding: 6px 12px; font-size: 12px; font-weight: 500;
      background: rgba(255,255,255,0.07); border: 1px solid var(--border-default);
      border-radius: var(--r-sm); color: var(--text-secondary); cursor: pointer; transition: all 0.2s;
    }
    .header-btn:hover { color: var(--text-primary); background: rgba(255,255,255,0.12); border-color: var(--border-bright); transform: translateY(-1px); }
    .header-btn:active { transform: scale(0.97); }

    @keyframes slideUpFade {
      from { opacity: 0; transform: translateY(20px); }
      to { opacity: 1; transform: translateY(0); }
    }

    /* Main content */
    main { flex: 1; padding: 24px; max-width: 1400px; margin: 0 auto; width: 100%; animation: slideUpFade 0.4s var(--ease-out) forwards; }

    /* Stats */
    .stats-grid {
      display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 20px;
    }
    .stat-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl);
      padding: 18px 20px; cursor: pointer; transition: all 0.2s;
      position: relative; overflow: hidden;
      animation: slideUpFade 0.4s var(--ease-out) backwards;
    }
    .stat-card:nth-child(1) { animation-delay: 0.05s; }
    .stat-card:nth-child(2) { animation-delay: 0.1s; }
    .stat-card:nth-child(3) { animation-delay: 0.15s; }
    .stat-card:nth-child(4) { animation-delay: 0.2s; }
    .stat-card::before {
      content: ''; position: absolute; inset: 0; opacity: 0;
      transition: opacity 0.2s; border-radius: inherit;
    }
    .stat-card:hover { background: var(--bg-card-hover); transform: translateY(-1px); box-shadow: var(--shadow-md); }
    .stat-card.active-filter::before { opacity: 1; }
    .stat-card[data-key="all"].active-filter::before { background: rgba(10,132,255,0.08); }
    .stat-card[data-key="paid"].active-filter::before { background: rgba(48,209,88,0.08); }
    .stat-card[data-key="pending"].active-filter::before { background: rgba(255,159,10,0.08); }
    .stat-card[data-key="cancelled"].active-filter::before { background: rgba(255,69,58,0.08); }
    .stat-label { font-size: 11px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 8px; }
    .stat-value { font-size: 32px; font-weight: 700; letter-spacing: -1px; font-feature-settings: "tnum"; }
    .stat-card[data-key="all"] .stat-value { color: var(--blue); }
    .stat-card[data-key="paid"] .stat-value { color: var(--green); }
    .stat-card[data-key="pending"] .stat-value { color: var(--orange); }
    .stat-card[data-key="cancelled"] .stat-value { color: var(--red); }

    /* Toolbar */
    .toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; flex-wrap: wrap; }
    .search-wrap { position: relative; flex: 1; min-width: 200px; max-width: 360px; }
    .search-icon { position: absolute; left: 10px; top: 50%; transform: translateY(-50%); color: var(--text-tertiary); pointer-events: none; }
    .search-input {
      width: 100%; padding: 8px 12px 8px 34px; font-size: 14px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-md);
      color: var(--text-primary); outline: none; transition: all 0.2s;
    }
    .search-input:focus { background: var(--bg-input-focus); border-color: var(--border-focus); box-shadow: 0 0 0 2px rgba(10,132,255,0.2); }
    .filter-pills { display: flex; gap: 6px; flex-wrap: wrap; }
    .pill {
      padding: 5px 12px; font-size: 12px; font-weight: 500; border-radius: var(--r-full);
      border: 1px solid var(--border-default); background: transparent; color: var(--text-secondary);
      cursor: pointer; transition: all 0.16s;
    }
    .pill:hover { color: var(--text-primary); border-color: var(--border-bright); }
    .pill.active { background: rgba(10,132,255,0.15); border-color: rgba(10,132,255,0.45); color: var(--blue); }
    .pill.active[data-status="paid"] { background: rgba(48,209,88,0.12); border-color: rgba(48,209,88,0.4); color: var(--green); }
    .pill.active[data-status="pending"] { background: rgba(255,159,10,0.12); border-color: rgba(255,159,10,0.4); color: var(--orange); }
    .pill.active[data-status="cancelled"] { background: rgba(255,69,58,0.1); border-color: rgba(255,69,58,0.35); color: var(--red); }
    .date-range { display: flex; align-items: center; gap: 6px; }
    .date-input {
      padding: 6px 10px; font-size: 12px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-md);
      color: var(--text-secondary); outline: none; cursor: pointer; transition: all 0.2s;
    }
    .date-input:focus { border-color: var(--border-focus); color: var(--text-primary); }
    .date-sep { color: var(--text-tertiary); font-size: 12px; }
    .date-clear { padding: 5px 10px; font-size: 11px; cursor: pointer; color: var(--text-tertiary); background: none; border: none; transition: color 0.15s; }
    .date-clear:hover { color: var(--red); }

    /* Table */
    .table-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl);
      overflow: hidden; box-shadow: var(--shadow-sm);
      animation: slideUpFade 0.5s var(--ease-out) backwards;
      animation-delay: 0.3s;
    }
    .table-scroll { overflow-x: auto; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    thead th {
      padding: 11px 14px; text-align: left; font-size: 11px; font-weight: 600;
      letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary);
      border-bottom: 1px solid var(--border-default); white-space: nowrap; background: rgba(255,255,255,0.02);
    }
    tbody tr {
      border-bottom: 1px solid rgba(255,255,255,0.04); cursor: pointer;
      transition: background 0.14s;
      animation: slideUpFade 0.3s var(--ease-out) backwards;
    }
    tbody tr:nth-child(1) { animation-delay: 0.35s; }
    tbody tr:nth-child(2) { animation-delay: 0.4s; }
    tbody tr:nth-child(3) { animation-delay: 0.45s; }
    tbody tr:nth-child(4) { animation-delay: 0.5s; }
    tbody tr:nth-child(5) { animation-delay: 0.55s; }
    tbody tr:last-child { border-bottom: none; }
    tbody tr:hover { background: rgba(10,132,255,0.04); }
    tbody td { padding: 12px 14px; color: var(--text-secondary); font-feature-settings: "tnum"; }
    tbody td.td-id { color: var(--text-primary); font-weight: 500; font-family: var(--font-mono); font-size: 12px; }
    @keyframes rowFlash {
      0% { background: rgba(10,132,255,0.14); }
      100% { background: transparent; }
    }
    .row-flash { animation: rowFlash 1.1s var(--ease-out); }

    /* Status badge */
    .badge {
      display: inline-flex; align-items: center; gap: 5px; padding: 3px 9px;
      border-radius: var(--r-full); font-size: 11px; font-weight: 600; letter-spacing: 0.03em; white-space: nowrap;
    }
    .badge-dot { width: 5px; height: 5px; border-radius: 50%; flex-shrink: 0; }
    .badge-paid { background: rgba(48,209,88,0.12); border: 1px solid rgba(48,209,88,0.35); color: var(--green); box-shadow: 0 0 8px rgba(48,209,88,0.1); }
    .badge-paid .badge-dot { background: var(--green); }
    .badge-pending { background: rgba(255,159,10,0.12); border: 1px solid rgba(255,159,10,0.35); color: var(--orange); }
    .badge-pending .badge-dot { background: var(--orange); }
    .badge-cancelled { background: rgba(255,69,58,0.09); border: 1px solid rgba(255,69,58,0.28); color: var(--red); }
    .badge-cancelled .badge-dot { background: var(--red); }

    /* Skeleton */
    @keyframes shimmer { from { background-position: -400px 0; } to { background-position: 400px 0; } }
    .skeleton {
      background: linear-gradient(90deg, rgba(255,255,255,0.04) 25%, rgba(255,255,255,0.09) 50%, rgba(255,255,255,0.04) 75%);
      background-size: 400px 100%; animation: shimmer 1.4s ease-in-out infinite;
      border-radius: 4px; height: 14px;
    }

    /* Table footer */
    .table-footer {
      display: flex; align-items: center; justify-content: space-between;
      padding: 12px 16px; border-top: 1px solid var(--border-default);
    }
    .table-count { font-size: 12px; color: var(--text-tertiary); }
    .load-more-btn {
      padding: 7px 16px; font-size: 13px; font-weight: 500;
      background: rgba(10,132,255,0.12); border: 1px solid rgba(10,132,255,0.3);
      border-radius: var(--r-md); color: var(--blue); cursor: pointer; transition: all 0.15s;
    }
    .load-more-btn:hover { background: rgba(10,132,255,0.2); }
    .load-more-btn:disabled { opacity: 0.4; cursor: not-allowed; }

    /* ── TESTING PANEL ── */
    .test-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 16px; }
    .test-card {
      background: var(--bg-card); border: 1px solid var(--border-default);
      border-top-color: var(--border-bright); border-radius: var(--r-xl); overflow: hidden;
      animation: slideUpFade 0.4s var(--ease-out) backwards;
    }
    .test-card:nth-child(1) { animation-delay: 0.1s; }
    .test-card:nth-child(2) { animation-delay: 0.15s; }
    .test-card:nth-child(3) { animation-delay: 0.2s; }
    .test-card:nth-child(4) { animation-delay: 0.25s; }
    .test-card:nth-child(5) { animation-delay: 0.3s; }
    .test-card:nth-child(6) { animation-delay: 0.35s; }
    .test-card:nth-child(7) { animation-delay: 0.4s; }
    .test-card-header {
      display: flex; align-items: center; gap: 10px;
      padding: 14px 16px; border-bottom: 1px solid var(--border-default);
    }
    .method-badge {
      padding: 2px 7px; border-radius: var(--r-sm); font-size: 10px; font-weight: 700;
      font-family: var(--font-mono); letter-spacing: 0.05em; flex-shrink: 0;
    }
    .method-get { background: rgba(48,209,88,0.15); color: var(--green); }
    .method-post { background: rgba(10,132,255,0.15); color: var(--blue); }
    .method-ws { background: rgba(191,90,242,0.15); color: var(--purple); }
    .method-patch { background: rgba(255,159,10,0.15); color: var(--orange); }
    .test-endpoint { font-size: 12px; font-family: var(--font-mono); color: var(--text-secondary); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .test-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
    .test-card-body { padding: 14px 16px; }
    .test-field { margin-bottom: 10px; }
    .test-field-label { font-size: 10px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 5px; }
    .test-input, .test-textarea {
      width: 100%; padding: 8px 10px; font-size: 12px; font-family: var(--font-mono);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-sm);
      color: var(--text-primary); outline: none; transition: all 0.2s;
    }
    .test-input:focus, .test-textarea:focus { background: var(--bg-input-focus); border-color: var(--border-focus); box-shadow: 0 0 0 2px rgba(10,132,255,0.2); }
    .test-textarea { resize: vertical; min-height: 60px; max-height: 200px; }
    .test-actions { display: flex; gap: 8px; margin-top: 10px; align-items: center; }
    .test-run-btn {
      padding: 7px 16px; font-size: 12px; font-weight: 600; border-radius: var(--r-sm);
      background: var(--blue); color: #fff; border: none; cursor: pointer; transition: all 0.2s; flex-shrink: 0;
      box-shadow: 0 2px 6px rgba(10,132,255,0.3);
    }
    .test-run-btn:hover { opacity: 0.9; transform: translateY(-1px); box-shadow: 0 4px 10px rgba(10,132,255,0.4); }
    .test-run-btn:active { transform: scale(0.96) translateY(0); box-shadow: 0 2px 4px rgba(10,132,255,0.3); }
    .test-run-btn.running { opacity: 0.6; cursor: not-allowed; transform: none; box-shadow: none; }
    .ws-btn {
      padding: 7px 14px; font-size: 12px; font-weight: 600; border-radius: var(--r-sm);
      border: 1px solid var(--border-default); color: var(--text-secondary); background: transparent;
      cursor: pointer; transition: all 0.15s; flex-shrink: 0;
    }
    .ws-btn.connected { color: var(--red); border-color: rgba(255,69,58,0.4); background: rgba(255,69,58,0.08); }
    .ws-btn:not(.connected):hover { color: var(--purple); border-color: rgba(191,90,242,0.4); }
    .autofill-hint { font-size: 11px; color: var(--text-tertiary); padding: 6px 10px; background: rgba(10,132,255,0.07); border: 1px solid rgba(10,132,255,0.2); border-radius: var(--r-sm); display: none; }
    .autofill-hint.visible { display: block; }
    .test-response {
      margin-top: 10px; background: rgba(0,0,0,0.35); border: 1px solid var(--border-default);
      border-radius: var(--r-sm); overflow: hidden; display: none;
    }
    .test-response.visible { display: block; }
    .test-response-head {
      display: flex; align-items: center; gap: 8px; padding: 7px 10px;
      border-bottom: 1px solid var(--border-default); background: rgba(255,255,255,0.02);
    }
    .status-pill { padding: 2px 7px; border-radius: var(--r-full); font-size: 10px; font-weight: 700; font-family: var(--font-mono); }
    .status-ok { background: rgba(48,209,88,0.15); color: var(--green); }
    .status-err { background: rgba(255,69,58,0.12); color: var(--red); }
    .timing { font-size: 10px; color: var(--text-tertiary); font-family: var(--font-mono); }
    .test-response-body {
      padding: 10px; font-family: var(--font-mono); font-size: 11px; color: var(--text-secondary);
      max-height: 180px; overflow-y: auto; white-space: pre; line-height: 1.55;
    }
    .json-key { color: var(--blue); }
    .json-str { color: var(--green); }
    .json-num { color: var(--orange); }
    .json-bool { color: var(--purple); }
    .json-null { color: var(--red); }

    /* ── CONSOLE ── */
    .console-toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
    .console-level-btn {
      padding: 4px 10px; font-size: 11px; font-weight: 500; border-radius: var(--r-sm);
      border: 1px solid var(--border-default); background: transparent; color: var(--text-tertiary); cursor: pointer; transition: all 0.15s;
    }
    .console-level-btn.active { background: rgba(255,255,255,0.08); color: var(--text-primary); border-color: var(--border-bright); }
    .console-clear { margin-left: auto; padding: 4px 10px; font-size: 11px; font-weight: 500; border-radius: var(--r-sm); border: 1px solid var(--border-default); background: transparent; color: var(--text-tertiary); cursor: pointer; transition: all 0.15s; }
    .console-clear:hover { color: var(--red); border-color: rgba(255,69,58,0.4); }
    .console-box {
      background: rgba(0,0,0,0.5); border: 1px solid var(--border-default); border-radius: var(--r-xl);
      padding: 14px; font-family: var(--font-mono); font-size: 12px; height: calc(100vh - 220px);
      overflow-y: auto; line-height: 1.65;
    }
    .log-entry { display: flex; gap: 10px; padding: 3px 0; border-bottom: 1px solid rgba(255,255,255,0.03); }
    .log-entry:last-child { border-bottom: none; }
    .log-time { color: var(--text-tertiary); flex-shrink: 0; }
    .log-level { flex-shrink: 0; font-weight: 700; width: 42px; }
    .log-level-info { color: var(--blue); }
    .log-level-ok { color: var(--green); }
    .log-level-warn { color: var(--orange); }
    .log-level-error { color: var(--red); }
    .log-msg { color: var(--text-secondary); word-break: break-all; }
    .log-data { color: var(--text-tertiary); margin-top: 2px; white-space: pre; }

    /* ── MODAL ── */
    #modal-overlay {
      position: fixed; inset: 0; z-index: 200;
      background: rgba(0,0,0,0.65); backdrop-filter: blur(12px);
      display: flex; align-items: center; justify-content: center; padding: 20px;
      animation: overlayIn 0.3s var(--ease-out) forwards;
    }
    @keyframes overlayIn {
      from { opacity: 0; }
      to { opacity: 1; }
    }
    .modal {
      background: var(--bg-modal); border: 1px solid rgba(255,255,255,0.1);
      border-radius: var(--r-2xl); width: 100%; max-width: 520px;
      box-shadow: 0 0 0 1px rgba(255,255,255,0.05), var(--shadow-xl); overflow: hidden;
      animation: modalIn 0.4s var(--ease-spring) forwards;
    }
    @keyframes modalIn {
      from { opacity: 0; transform: scale(0.92) translateY(20px); }
      to { opacity: 1; transform: scale(1) translateY(0); }
    }
    .modal-header {
      padding: 20px 24px 16px; border-bottom: 1px solid var(--border-default);
      display: flex; align-items: flex-start; gap: 12px;
    }
    .modal-tid { font-size: 13px; font-family: var(--font-mono); color: var(--blue); font-weight: 600; }
    .modal-title { font-size: 18px; font-weight: 700; margin-top: 2px; }
    .modal-close { margin-left: auto; width: 28px; height: 28px; border-radius: 50%; background: rgba(255,255,255,0.08); border: none; cursor: pointer; color: var(--text-secondary); font-size: 16px; display: flex; align-items: center; justify-content: center; transition: all 0.15s; flex-shrink: 0; }
    .modal-close:hover { background: rgba(255,255,255,0.15); color: var(--text-primary); }
    .modal-body { padding: 20px 24px; max-height: 55vh; overflow-y: auto; }
    .modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
    .modal-grid .full { grid-column: 1 / -1; }
    .modal-field { }
    .modal-label { font-size: 10px; font-weight: 600; letter-spacing: 0.07em; text-transform: uppercase; color: var(--text-tertiary); margin-bottom: 6px; }
    .modal-input, .modal-select {
      width: 100%; padding: 9px 12px; font-size: 13px; font-family: var(--font-ui);
      background: var(--bg-input); border: 1px solid var(--border-default); border-radius: var(--r-sm);
      color: var(--text-primary); outline: none; transition: all 0.2s;
    }
    .modal-input:focus, .modal-select:focus { background: var(--bg-input-focus); border-color: var(--border-focus); }
    .modal-select option { background: #1c1c1e; }
    .modal-footer {
      padding: 16px 24px; border-top: 1px solid var(--border-default);
      display: flex; align-items: center; gap: 8px;
    }
    .modal-action-btn {
      padding: 8px 16px; font-size: 13px; font-weight: 600; border-radius: var(--r-sm);
      cursor: pointer; transition: all 0.15s; border: none;
    }
    .btn-paid { background: rgba(48,209,88,0.15); color: var(--green); border: 1px solid rgba(48,209,88,0.3); }
    .btn-paid:hover { background: rgba(48,209,88,0.25); }
    .btn-cancel { background: rgba(255,69,58,0.1); color: var(--red); border: 1px solid rgba(255,69,58,0.25); }
    .btn-cancel:hover { background: rgba(255,69,58,0.18); }
    .modal-footer-spacer { flex: 1; }
    .btn-discard { padding: 8px 14px; font-size: 13px; font-weight: 500; border-radius: var(--r-sm); cursor: pointer; background: transparent; border: 1px solid var(--border-default); color: var(--text-secondary); transition: all 0.15s; }
    .btn-discard:hover { border-color: var(--border-bright); color: var(--text-primary); }
    .btn-save { padding: 8px 18px; font-size: 13px; font-weight: 600; border-radius: var(--r-sm); cursor: pointer; background: var(--blue); border: none; color: #fff; transition: opacity 0.15s; }
    .btn-save:hover { opacity: 0.85; }
    .btn-save:disabled { opacity: 0.4; cursor: not-allowed; }

    /* ── TOASTS ── */
    #toast-stack { position: fixed; bottom: 24px; right: 24px; display: flex; flex-direction: column; gap: 8px; z-index: 500; pointer-events: none; }
    .toast {
      padding: 12px 16px; border-radius: var(--r-md); font-size: 13px; font-weight: 500;
      box-shadow: 0 4px 16px rgba(0,0,0,0.4), 0 0 0 1px rgba(255,255,255,0.05); backdrop-filter: blur(24px) saturate(150%);
      animation: toastIn 0.4s var(--ease-spring) forwards;
      border: 1px solid rgba(255,255,255,0.12);
    }
    .toast-success { background: rgba(48,209,88,0.15); color: var(--green); border-color: rgba(48,209,88,0.3); }
    .toast-error { background: rgba(255,69,58,0.15); color: var(--red); border-color: rgba(255,69,58,0.3); }
    .toast-info { background: rgba(10,132,255,0.15); color: var(--blue); border-color: rgba(10,132,255,0.3); }
    @keyframes toastIn { from { opacity: 0; transform: translateX(30px) scale(0.96); } to { opacity: 1; transform: translateX(0) scale(1); } }
    @keyframes toastOut { to { opacity: 0; transform: translateX(30px) scale(0.96); } }

    @media (max-width: 768px) {
      .stats-grid { grid-template-columns: repeat(2, 1fr); }
      .test-grid { grid-template-columns: 1fr; }
      .modal-grid { grid-template-columns: 1fr; }
      .modal-grid .full { grid-column: 1; }
      header { padding: 0 16px; }
      main { padding: 16px; }
    }
  </style>
</head>
<body>

<!-- LOGIN -->
<div id="login-screen">
  <div class="login-card" id="login-card">
    <div class="login-logo">&#9632; PayAdmin</div>
    <div class="login-sub">Sign in to access the dashboard</div>
    <div class="login-label">Password</div>
    <input class="login-input" id="login-pw" type="password" placeholder="Enter password" autocomplete="current-password">
    <button class="login-btn" id="login-btn">Sign In</button>
    <div class="login-err" id="login-err"></div>
  </div>
</div>

<!-- DASHBOARD -->
<div id="dashboard" style="display:none">
  <header>
    <div class="header-logo">&#9632; PayAdmin</div>
    <nav class="nav-tabs">
      <button class="nav-tab active" data-tab="overview">Overview</button>
      <button class="nav-tab" data-tab="testing">Testing</button>
      <button class="nav-tab" data-tab="console">Console</button>
    </nav>
    <div class="header-spacer"></div>
    <span class="rt-label">Live</span>
    <div class="rt-dot disconnected" id="rt-dot"></div>
    <button class="header-btn" id="refresh-btn">&#8635; Refresh</button>
    <button class="header-btn" id="signout-btn">Sign Out</button>
  </header>

  <main>
    <!-- OVERVIEW TAB -->
    <div id="tab-overview">
      <div class="stats-grid">
        <div class="stat-card active-filter" data-key="all" onclick="Dashboard.setFilter('')">
          <div class="stat-label">Total Tickets</div>
          <div class="stat-value" id="stat-total">&#8212;</div>
        </div>
        <div class="stat-card" data-key="paid" onclick="Dashboard.setFilter('paid')">
          <div class="stat-label">Paid</div>
          <div class="stat-value" id="stat-paid">&#8212;</div>
        </div>
        <div class="stat-card" data-key="pending" onclick="Dashboard.setFilter('pending')">
          <div class="stat-label">Pending</div>
          <div class="stat-value" id="stat-pending">&#8212;</div>
        </div>
        <div class="stat-card" data-key="cancelled" onclick="Dashboard.setFilter('cancelled')">
          <div class="stat-label">Cancelled</div>
          <div class="stat-value" id="stat-cancelled">&#8212;</div>
        </div>
      </div>

      <div class="toolbar">
        <div class="search-wrap">
          <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>
          <input class="search-input" id="search-input" type="text" placeholder="Search tickets, names, UPI IDs...">
        </div>
        <div class="filter-pills">
          <button class="pill active" data-status="" onclick="Dashboard.setFilter('')">All</button>
          <button class="pill" data-status="paid" onclick="Dashboard.setFilter('paid')">Paid</button>
          <button class="pill" data-status="pending" onclick="Dashboard.setFilter('pending')">Pending</button>
          <button class="pill" data-status="cancelled" onclick="Dashboard.setFilter('cancelled')">Cancelled</button>
        </div>
        <div class="date-range">
          <input class="date-input" id="date-from" type="date" title="From date">
          <span class="date-sep">&#8594;</span>
          <input class="date-input" id="date-to" type="date" title="To date">
          <button class="date-clear" id="date-clear" title="Clear dates">&#10005;</button>
        </div>
      </div>

      <div class="table-card">
        <div class="table-scroll">
          <table>
            <thead>
              <tr>
                <th>Ticket ID</th>
                <th>Amount</th>
                <th>Status</th>
                <th>Sender</th>
                <th>UPI ID</th>
                <th>RRN</th>
                <th>Created</th>
                <th>Paid At</th>
              </tr>
            </thead>
            <tbody id="ticket-tbody"></tbody>
          </table>
        </div>
        <div class="table-footer">
          <div class="table-count" id="table-count">Loading...</div>
          <button class="load-more-btn" id="load-more-btn" style="display:none" onclick="Dashboard.loadMore()">Load More</button>
        </div>
      </div>
    </div>

    <!-- TESTING TAB -->
    <div id="tab-testing" style="display:none">
      <div class="test-grid">

        <!-- 1. Create Ticket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/ticket</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Create Ticket</div>
            <div class="autofill-hint" id="hint-create"></div>
            <div class="test-field">
              <div class="test-field-label">Amount (₹)</div>
              <input class="test-input" id="tc-amount" type="number" value="500" min="1">
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.createTicket()">Run</button>
            </div>
            <div class="test-response" id="resp-create"></div>
          </div>
        </div>

        <!-- 2. Standard SMS Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">SMS Webhook (Standard)</div>
            <div class="autofill-hint" id="hint-sms"></div>
            <div class="test-field">
              <div class="test-field-label">Webhook Secret</div>
              <input class="test-input" id="tw-secret" type="text" placeholder="Enter webhook secret">
            </div>
            <div class="test-field">
              <div class="test-field-label">SMS Body</div>
              <textarea class="test-textarea" id="tw-sms" rows="3">TICKET1234567890
Test Sender paid you &#8377;500</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.smsWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-sms"></div>
          </div>
        </div>

        <!-- 3. Kotak SMS Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">SMS Webhook (Kotak)</div>
            <div class="autofill-hint" id="hint-kotak"></div>
            <div class="test-field">
              <div class="test-field-label">Webhook Secret</div>
              <input class="test-input" id="tk-secret" type="text" placeholder="Enter webhook secret">
            </div>
            <div class="test-field">
              <div class="test-field-label">Kotak SMS Body</div>
              <textarea class="test-textarea" id="tk-sms" rows="3">Confirmed payment for Received Rs.500.00 in your Kotak Bank AC X4959 from testuser@oksbi on 15-03-26.UPI Ref:123456789012. KOTAK</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.kotakWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-kotak"></div>
          </div>
        </div>

        <!-- 4. Email Webhook -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-post">POST</span>
            <span class="test-endpoint">/api/email-webhook</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Email Webhook (Slice)</div>
            <div class="autofill-hint" id="hint-email"></div>
            <div class="test-field">
              <div class="test-field-label">Email Secret</div>
              <input class="test-input" id="te-secret" type="text" placeholder="your-email-secret">
            </div>
            <div class="test-field">
              <div class="test-field-label">From</div>
              <input class="test-input" id="te-from" value="no-reply@sliceit.com">
            </div>
            <div class="test-field">
              <div class="test-field-label">Subject</div>
              <input class="test-input" id="te-subject" value="&#8377;500 received via UPI">
            </div>
            <div class="test-field">
              <div class="test-field-label">Body (HTML)</div>
              <textarea class="test-textarea" id="te-body" rows="4">&lt;table&gt;&lt;tr&gt;&lt;td&gt;From&lt;/td&gt;&lt;td&gt;TEST SENDER&lt;/td&gt;&lt;/tr&gt;&lt;tr&gt;&lt;td&gt;RRN&lt;/td&gt;&lt;td&gt;123456789012&lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;</textarea>
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.emailWebhook()">Run</button>
            </div>
            <div class="test-response" id="resp-email"></div>
          </div>
        </div>

        <!-- 5. Status Check -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-get">GET</span>
            <span class="test-endpoint">/api/status/:id</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Check Ticket Status</div>
            <div class="autofill-hint" id="hint-status"></div>
            <div class="test-field">
              <div class="test-field-label">Ticket ID</div>
              <input class="test-input" id="ts-id" type="text" placeholder="TICKET1234567890">
            </div>
            <div class="test-actions">
              <button class="test-run-btn" onclick="Testing.checkStatus()">Run</button>
            </div>
            <div class="test-response" id="resp-status"></div>
          </div>
        </div>

        <!-- 6. Ping WebSocket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-ws">WS</span>
            <span class="test-endpoint">/api/ping</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Ping WebSocket</div>
            <div class="test-actions">
              <button class="ws-btn" id="ws-ping-btn" onclick="Testing.togglePingWs()">Connect</button>
            </div>
            <div class="test-response" id="resp-ping"></div>
          </div>
        </div>

        <!-- 7. Ticket WebSocket -->
        <div class="test-card">
          <div class="test-card-header">
            <span class="method-badge method-ws">WS</span>
            <span class="test-endpoint">/api/ws</span>
          </div>
          <div class="test-card-body">
            <div class="test-title" style="margin-bottom:10px">Ticket WebSocket</div>
            <div class="autofill-hint" id="hint-ticketws"></div>
            <div class="test-field">
              <div class="test-field-label">Ticket ID</div>
              <input class="test-input" id="ws-ticket-id" type="text" placeholder="TICKET1234567890">
            </div>
            <div class="test-actions">
              <button class="ws-btn" id="ws-ticket-btn" onclick="Testing.toggleTicketWs()">Connect</button>
            </div>
            <div class="test-response" id="resp-ticketws"></div>
          </div>
        </div>

      </div>
    </div>

    <!-- CONSOLE TAB -->
    <div id="tab-console" style="display:none">
      <div class="console-toolbar">
        <button class="console-level-btn active" data-level="" onclick="AppConsole.setFilter('')">All</button>
        <button class="console-level-btn" data-level="info" onclick="AppConsole.setFilter('info')">Info</button>
        <button class="console-level-btn" data-level="ok" onclick="AppConsole.setFilter('ok')">OK</button>
        <button class="console-level-btn" data-level="warn" onclick="AppConsole.setFilter('warn')">Warn</button>
        <button class="console-level-btn" data-level="error" onclick="AppConsole.setFilter('error')">Error</button>
        <button class="console-clear" onclick="AppConsole.clear()">Clear</button>
      </div>
      <div class="console-box" id="console-box"></div>
    </div>
  </main>
</div>

<!-- MODAL -->
<div id="modal-overlay" style="display:none">
  <div class="modal" id="modal">
    <div class="modal-header">
      <div>
        <div class="modal-tid" id="modal-tid"></div>
        <div class="modal-title" id="modal-title">Ticket Details</div>
      </div>
      <button class="modal-close" onclick="Modal.close()">&#10005;</button>
    </div>
    <div class="modal-body">
      <div class="modal-grid">
        <div class="modal-field full">
          <div class="modal-label">Status</div>
          <select class="modal-select" id="m-status">
            <option value="pending">Pending</option>
            <option value="paid">Paid</option>
            <option value="cancelled">Cancelled</option>
          </select>
        </div>
        <div class="modal-field">
          <div class="modal-label">Amount (₹)</div>
          <input class="modal-input" id="m-amount" type="number" step="0.01">
        </div>
        <div class="modal-field">
          <div class="modal-label">Sender Name</div>
          <input class="modal-input" id="m-sender" type="text">
        </div>
        <div class="modal-field">
          <div class="modal-label">RRN</div>
          <input class="modal-input" id="m-rrn" type="text">
        </div>
        <div class="modal-field">
          <div class="modal-label">UPI ID</div>
          <input class="modal-input" id="m-upi" type="text">
        </div>
        <div class="modal-field full">
          <div class="modal-label">Paid At (ISO)</div>
          <input class="modal-input" id="m-paidat" type="text" placeholder="e.g. 2024-03-15T10:30:00.000Z">
        </div>
        <div class="modal-field full">
          <div class="modal-label">Created At</div>
          <input class="modal-input" id="m-createdat" type="text" readonly style="opacity:0.5;cursor:not-allowed">
        </div>
      </div>
    </div>
    <div class="modal-footer">
      <button class="modal-action-btn btn-paid" onclick="Modal.markPaid()">Mark as Paid</button>
      <button class="modal-action-btn btn-cancel" onclick="Modal.cancelTicket()">Cancel Ticket</button>
      <div class="modal-footer-spacer"></div>
      <button class="btn-discard" onclick="Modal.discard()">Discard</button>
      <button class="btn-save" id="modal-save-btn" onclick="Modal.save()">Save</button>
    </div>
  </div>
</div>

<div id="toast-stack"></div>

<script>
(function() {

// ─── STATE ───────────────────────────────────────────────────────────────────
var State = {
  pw: '',
  allLoaded: [],
  stats: { total: 0, paid: 0, pending: 0, cancelled: 0 },
  hasMore: false,
  currentPage: 0,
  PAGE_SIZE: 10,
  statusFilter: '',
  search: '',
  searchTimer: null,
  dateFrom: '',
  dateTo: '',
  lastCreated: null,  // { ticketId, amount }
  logs: [],
  logFilter: '',
  activeTab: 'overview',
  pingWs: null,
  ticketWs: null,
  modalTicketId: null
};

// ─── UTIL ────────────────────────────────────────────────────────────────────
var Util = {
  escape: function(s) {
    return String(s || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
  },
  fmtAmount: function(n) {
    return '\\u20B9' + Number(n).toFixed(2);
  },
  relativeTime: function(iso) {
    if (!iso) return '—';
    var diff = Date.now() - new Date(iso).getTime();
    var s = Math.floor(diff / 1000);
    if (s < 60) return s + 's ago';
    var m = Math.floor(s / 60);
    if (m < 60) return m + 'm ago';
    var h = Math.floor(m / 60);
    if (h < 24) return h + 'h ago';
    return Math.floor(h / 24) + 'd ago';
  },
  fmtDateShort: function(iso) {
    if (!iso) return '—';
    var d = new Date(iso);
    return d.toLocaleDateString('en-IN', { day:'2-digit', month:'short', year:'2-digit' })
      + ' ' + d.toLocaleTimeString('en-IN', { hour:'2-digit', minute:'2-digit', hour12: true });
  },
  colorizeJson: function(obj) {
    var json = typeof obj === 'string' ? obj : JSON.stringify(obj, null, 2);
    json = json.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    return json.split('\\n').map(function(line) {
      // Key: leading "word":
      line = line.replace(/^(\s*)("(?:[^"]*)")(\s*:)/, function(m, sp, k, c) {
        return sp + '<span class="json-key">' + k + '</span>' + c;
      });
      // String value after colon
      line = line.replace(/(:\s*)("(?:[^"]*)")/, function(m, c, v) {
        return c + '<span class="json-str">' + v + '</span>';
      });
      // Number value
      line = line.replace(/(:\s*)([-]?[0-9]+[.]?[0-9]*)/, function(m, c, v) {
        return c + '<span class="json-num">' + v + '</span>';
      });
      // Boolean
      line = line.replace(/(:\s*)(true|false)/, function(m, c, v) {
        return c + '<span class="json-bool">' + v + '</span>';
      });
      // Null
      line = line.replace(/(:\s*)(null)/, function(m, c, v) {
        return c + '<span class="json-null">' + v + '</span>';
      });
      return line;
    }).join('\\n');
  }
};

// ─── API ─────────────────────────────────────────────────────────────────────
var API = {
  buildTicketsUrl: function() {
    var p = new URLSearchParams();
    p.set('page', State.currentPage.toString());
    p.set('pageSize', State.PAGE_SIZE.toString());
    if (State.search.trim()) p.set('search', State.search.trim());
    if (State.statusFilter) p.set('status', State.statusFilter);
    if (State.dateFrom) p.set('dateFrom', State.dateFrom);
    if (State.dateTo) p.set('dateTo', State.dateTo);
    return '/api/admin/tickets?' + p.toString();
  },
  headers: function() {
    return { 'Content-Type': 'application/json', 'X-Admin-Password': State.pw };
  },
  get: async function(url) {
    var t0 = Date.now();
    AppConsole.log('info', 'GET ' + url);
    try {
      var r = await fetch(url, { headers: this.headers() });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'GET ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'GET ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  },
  post: async function(url, body, extraHeaders) {
    var t0 = Date.now();
    AppConsole.log('info', 'POST ' + url);
    try {
      var hdrs = Object.assign({}, this.headers(), extraHeaders || {});
      var r = await fetch(url, { method: 'POST', headers: hdrs, body: JSON.stringify(body) });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'POST ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'POST ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  },
  patch: async function(url, body) {
    var t0 = Date.now();
    AppConsole.log('info', 'PATCH ' + url);
    try {
      var r = await fetch(url, { method: 'PATCH', headers: this.headers(), body: JSON.stringify(body) });
      var data = await r.json();
      AppConsole.log(r.ok ? 'ok' : 'warn', 'PATCH ' + url + ' -> ' + r.status + ' (' + (Date.now()-t0) + 'ms)');
      return { ok: r.ok, status: r.status, data: data, ms: Date.now()-t0 };
    } catch(e) {
      AppConsole.log('error', 'PATCH ' + url + ' failed: ' + e.message);
      return { ok: false, status: 0, data: { error: e.message }, ms: Date.now()-t0 };
    }
  }
};

// ─── AUTH ────────────────────────────────────────────────────────────────────
var Auth = {
  init: function() {
    var saved = sessionStorage.getItem('adminPw');
    if (saved) { State.pw = saved; this.enter(); return; }
    document.getElementById('login-pw').addEventListener('keydown', function(e) {
      if (e.key === 'Enter') Auth.login();
    });
    document.getElementById('login-btn').addEventListener('click', Auth.login.bind(Auth));
  },
  login: async function() {
    var pw = document.getElementById('login-pw').value;
    if (!pw) return;
    State.pw = pw;
    var res = await API.get('/api/admin/tickets?page=0&pageSize=1');
    if (!res.ok && res.status === 401) {
      State.pw = '';
      document.getElementById('login-err').textContent = 'Incorrect password.';
      var card = document.getElementById('login-card');
      card.classList.remove('shake');
      void card.offsetWidth;
      card.classList.add('shake');
      return;
    }
    sessionStorage.setItem('adminPw', pw);
    this.enter();
  },
  enter: function() {
    document.getElementById('login-screen').style.display = 'none';
    document.getElementById('dashboard').style.display = 'flex';
    Dashboard.loadTickets(true, true);
    Realtime.connect();
  },
  logout: function() {
    sessionStorage.removeItem('adminPw');
    State.pw = '';
    Realtime.disconnect();
    document.getElementById('dashboard').style.display = 'none';
    document.getElementById('login-screen').style.display = 'flex';
    document.getElementById('login-pw').value = '';
    document.getElementById('login-err').textContent = '';
  }
};

// ─── REALTIME ─────────────────────────────────────────────────────────────────
var Realtime = {
  ws: null,
  reconnTimer: null,
  forceClosed: false,
  connect: function() {
    this.forceClosed = false;
    clearTimeout(this.reconnTimer);
    try {
      var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      var url = proto + '//' + location.host + '/api/admin/ws?pw=' + encodeURIComponent(State.pw);
      this.ws = new WebSocket(url);
      this.ws.onopen = function() {
        Realtime.setStatus(true);
        AppConsole.log('ok', '[Realtime] Connected to live feed');
      };
      this.ws.onmessage = function(e) {
        try {
          var msg = JSON.parse(e.data);
          if (msg.type === 'ticket_update') Realtime.handleUpdate(msg.ticket, msg.action);
        } catch(_) {}
      };
      this.ws.onclose = function() {
        Realtime.setStatus(false);
        if (!Realtime.forceClosed) {
          AppConsole.log('warn', '[Realtime] Disconnected — reconnecting in 5s');
          Realtime.reconnTimer = setTimeout(function() { Realtime.connect(); }, 5000);
        }
      };
      this.ws.onerror = function() {
        Realtime.setStatus(false);
        AppConsole.log('error', '[Realtime] WebSocket error');
      };
    } catch(e) {
      AppConsole.log('error', '[Realtime] Could not connect: ' + e.message);
    }
  },
  disconnect: function() {
    this.forceClosed = true;
    clearTimeout(this.reconnTimer);
    try { if (this.ws) this.ws.close(); } catch(_) {}
    this.ws = null;
    this.setStatus(false);
  },
  handleUpdate: function(ticket, action) {
    AppConsole.log('info', '[Realtime] ' + action + ': ' + ticket.ticketId + ' (' + ticket.status + ')');
    if (action === 'delete') {
      State.allLoaded = State.allLoaded.filter(function(t) { return t.id !== ticket.id; });
      if (State.stats.total > 0) State.stats.total--;
    } else if (action === 'create') {
      State.allLoaded.unshift(ticket);
      State.stats.total = (State.stats.total || 0) + 1;
      State.stats[ticket.status] = (State.stats[ticket.status] || 0) + 1;
      UI.toast('New ticket: ' + ticket.ticketId, 'info');
    } else {
      var idx = -1;
      for (var i = 0; i < State.allLoaded.length; i++) {
        if (State.allLoaded[i].id === ticket.id) { idx = i; break; }
      }
      if (idx >= 0) {
        var old = State.allLoaded[idx];
        if (old.status !== ticket.status) {
          State.stats[old.status] = Math.max(0, (State.stats[old.status] || 0) - 1);
          State.stats[ticket.status] = (State.stats[ticket.status] || 0) + 1;
          if (ticket.status === 'paid') UI.toast('Payment received: ' + ticket.ticketId, 'success');
        }
        State.allLoaded[idx] = ticket;
      } else {
        // Not in current page — just update stats footer silently
      }
    }
    Dashboard.renderStats();
    Dashboard.renderTickets();
    // Flash updated row
    setTimeout(function() {
      var row = document.getElementById('row-' + ticket.id);
      if (row) {
        row.classList.remove('row-flash');
        void row.offsetWidth;
        row.classList.add('row-flash');
      }
    }, 30);
  },
  setStatus: function(connected) {
    var dot = document.getElementById('rt-dot');
    if (dot) dot.className = 'rt-dot ' + (connected ? 'connected' : 'disconnected');
  }
};

// ─── DASHBOARD ───────────────────────────────────────────────────────────────
var Dashboard = {
  loading: false,
  prevStats: { total: 0, paid: 0, pending: 0, cancelled: 0 },

  setFilter: function(status) {
    State.statusFilter = status;
    // Update pills
    document.querySelectorAll('.pill').forEach(function(el) {
      el.classList.toggle('active', el.dataset.status === status);
    });
    // Update stat cards
    document.querySelectorAll('.stat-card').forEach(function(el) {
      var key = el.dataset.key;
      var matches = (status === '' && key === 'all') || (status === key);
      el.classList.toggle('active-filter', matches);
    });
    this.loadTickets(true, true);
  },

  loadTickets: async function(reset, spin) {
    if (this.loading) return;
    this.loading = true;
    if (reset) { State.currentPage = 0; State.allLoaded = []; }
    if (spin) UI.showTableLoading();
    var res = await API.get(API.buildTicketsUrl());
    this.loading = false;
    if (!res.ok) { UI.toast('Failed to load tickets', 'error'); return; }
    var d = res.data;
    State.stats = d.stats || State.stats;
    State.hasMore = d.hasMore || false;
    if (reset) {
      State.allLoaded = d.tickets || [];
    } else {
      State.allLoaded = State.allLoaded.concat(d.tickets || []);
    }
    this.renderStats();
    this.renderTickets();
  },

  loadMore: async function() {
    var btn = document.getElementById('load-more-btn');
    if (btn) { btn.disabled = true; btn.textContent = 'Loading...'; }
    State.currentPage++;
    await this.loadTickets(false, false);
    if (btn) { btn.disabled = false; btn.textContent = 'Load More'; }
  },

  renderStats: function() {
    var s = State.stats;
    var keys = ['total','paid','pending','cancelled'];
    for (var i = 0; i < keys.length; i++) {
      var k = keys[i];
      var el = document.getElementById('stat-' + k);
      if (el) {
        var prev = this.prevStats[k] || 0;
        var next = s[k] || 0;
        if (prev !== next) UI.animateCounter(el, prev, next);
        else el.textContent = next;
        this.prevStats[k] = next;
      }
    }
  },

  renderTickets: function() {
    var tbody = document.getElementById('ticket-tbody');
    if (!tbody) return;
    var items = State.allLoaded;
    if (items.length === 0) {
      tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-tertiary)">No tickets found</td></tr>';
      document.getElementById('table-count').textContent = '0 tickets';
      document.getElementById('load-more-btn').style.display = 'none';
      return;
    }
    var html = '';
    for (var i = 0; i < items.length; i++) {
      var t = items[i];
      var badgeCls = 'badge badge-' + t.status;
      var dot = '<span class="badge-dot"></span>';
      html += '<tr id="row-' + Util.escape(t.id) + '" data-id="' + Util.escape(t.id) + '" onclick="Modal.open(this.dataset.id)">';
      html += '<td class="td-id">' + Util.escape(t.ticketId) + '</td>';
      html += '<td>' + Util.fmtAmount(t.amount) + '</td>';
      html += '<td><span class="' + badgeCls + '">' + dot + Util.escape(t.status) + '</span></td>';
      html += '<td>' + Util.escape(t.senderName || '—') + '</td>';
      html += '<td style="font-family:var(--font-mono);font-size:11px">' + Util.escape(t.upiId || '—') + '</td>';
      html += '<td style="font-family:var(--font-mono);font-size:11px">' + Util.escape(t.rrn || '—') + '</td>';
      html += '<td title="' + Util.escape(t.createdAt) + '">' + Util.relativeTime(t.createdAt) + '</td>';
      html += '<td title="' + Util.escape(t.paidAt || '') + '">' + (t.paidAt ? Util.fmtDateShort(t.paidAt) : '—') + '</td>';
      html += '</tr>';
    }
    tbody.innerHTML = html;

    var total = State.stats.total || 0;
    var showing = items.length;
    document.getElementById('table-count').textContent = 'Showing ' + showing + ' of ' + total + ' tickets';

    var loadBtn = document.getElementById('load-more-btn');
    loadBtn.style.display = State.hasMore ? 'block' : 'none';
  }
};

// ─── MODAL ───────────────────────────────────────────────────────────────────
var Modal = {
  original: null,
  open: function(id) {
    var ticket = null;
    for (var i = 0; i < State.allLoaded.length; i++) {
      if (State.allLoaded[i].id === id) { ticket = State.allLoaded[i]; break; }
    }
    if (!ticket) return;
    State.modalTicketId = id;
    this.original = JSON.parse(JSON.stringify(ticket));
    document.getElementById('modal-tid').textContent = ticket.ticketId;
    document.getElementById('modal-title').textContent = 'Ticket Details';
    document.getElementById('m-status').value = ticket.status;
    document.getElementById('m-amount').value = ticket.amount;
    document.getElementById('m-sender').value = ticket.senderName || '';
    document.getElementById('m-rrn').value = ticket.rrn || '';
    document.getElementById('m-upi').value = ticket.upiId || '';
    document.getElementById('m-paidat').value = ticket.paidAt || '';
    document.getElementById('m-createdat').value = ticket.createdAt || '';
    document.getElementById('modal-overlay').style.display = 'flex';
  },
  close: function() {
    document.getElementById('modal-overlay').style.display = 'none';
    State.modalTicketId = null;
    this.original = null;
  },
  discard: function() { this.close(); },
  save: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var payload = {
      status: document.getElementById('m-status').value,
      amount: parseFloat(document.getElementById('m-amount').value),
      senderName: document.getElementById('m-sender').value || undefined,
      rrn: document.getElementById('m-rrn').value || undefined,
      upiId: document.getElementById('m-upi').value || undefined,
      paidAt: document.getElementById('m-paidat').value || undefined
    };
    var btn = document.getElementById('modal-save-btn');
    btn.disabled = true; btn.textContent = 'Saving...';
    var res = await API.patch('/api/admin/tickets/' + encodeURIComponent(id), payload);
    btn.disabled = false; btn.textContent = 'Save';
    if (res.ok) {
      UI.toast('Ticket updated', 'success');
      this.close();
      Dashboard.loadTickets(true, false);
    } else {
      UI.toast('Update failed: ' + (res.data.error || res.status), 'error');
    }
  },
  markPaid: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var res = await API.post('/api/admin/tickets/' + encodeURIComponent(id) + '/mark-paid', {});
    if (res.ok) { UI.toast('Marked as paid', 'success'); this.close(); Dashboard.loadTickets(true, false); }
    else UI.toast('Failed: ' + (res.data.error || res.status), 'error');
  },
  cancelTicket: async function() {
    var id = State.modalTicketId;
    if (!id) return;
    var res = await API.post('/api/admin/tickets/' + encodeURIComponent(id) + '/cancel', {});
    if (res.ok) { UI.toast('Ticket cancelled', 'success'); this.close(); Dashboard.loadTickets(true, false); }
    else UI.toast('Failed: ' + (res.data.error || res.status), 'error');
  }
};

// ─── TESTING ─────────────────────────────────────────────────────────────────
var Testing = {
  showResponse: function(elId, res) {
    var el = document.getElementById(elId);
    if (!el) return;
    var pillCls = res.ok ? 'status-pill status-ok' : 'status-pill status-err';
    var html = '<div class="test-response-head">'
      + '<span class="' + pillCls + '">' + (res.status || 'ERR') + '</span>'
      + '<span class="timing">' + res.ms + 'ms</span>'
      + '</div>'
      + '<div class="test-response-body">' + Util.colorizeJson(res.data) + '</div>';
    el.innerHTML = html;
    el.classList.add('visible');
  },

  autoFill: function(ticketId, amount) {
    State.lastCreated = { ticketId: ticketId, amount: amount };
    var amtStr = amount.toFixed(2);
    var intAmt = Math.floor(amount);
    var decAmt = Math.round((amount - intAmt) * 100);
    var decStr = String(decAmt).padStart(2, '0');
    var today = new Date();
    var dd = String(today.getDate()).padStart(2,'0');
    var mm = String(today.getMonth()+1).padStart(2,'0');
    var yy = String(today.getFullYear()).slice(-2);
    var dateStr = dd + '-' + mm + '-' + yy;

    // SMS
    var smsEl = document.getElementById('tw-sms');
    if (smsEl) smsEl.value = ticketId + '\\nTest Sender paid you \\u20B9' + amtStr;
    var hintSms = document.getElementById('hint-sms');
    if (hintSms) { hintSms.textContent = '\\u2713 Auto-filled with ticket ' + ticketId; hintSms.classList.add('visible'); }

    // Kotak SMS
    var kotakEl = document.getElementById('tk-sms');
    if (kotakEl) kotakEl.value = 'Confirmed payment for Received Rs.' + amtStr + ' in your Kotak Bank AC X4959 from testuser@oksbi on ' + dateStr + '.UPI Ref:' + (123456789000 + decAmt) + '. KOTAK';
    var hintKotak = document.getElementById('hint-kotak');
    if (hintKotak) { hintKotak.textContent = '\\u2713 Auto-filled with amount ' + amtStr; hintKotak.classList.add('visible'); }

    // Email
    var subEl = document.getElementById('te-subject');
    var bodyEl = document.getElementById('te-body');
    if (subEl) subEl.value = '\\u20B9' + amtStr + ' received via UPI';
    if (bodyEl) bodyEl.value = '<table><tr><td>From</td><td>TEST SENDER</td></tr><tr><td>RRN</td><td>123456789' + decStr + '</td></tr></table>';
    var hintEmail = document.getElementById('hint-email');
    if (hintEmail) { hintEmail.textContent = '\\u2713 Auto-filled with amount ' + amtStr; hintEmail.classList.add('visible'); }

    // Status check
    var statusEl = document.getElementById('ts-id');
    if (statusEl) statusEl.value = ticketId;
    var hintStatus = document.getElementById('hint-status');
    if (hintStatus) { hintStatus.textContent = '\\u2713 Auto-filled with ' + ticketId; hintStatus.classList.add('visible'); }

    // Ticket WS
    var wsEl = document.getElementById('ws-ticket-id');
    if (wsEl) wsEl.value = ticketId;
    var hintWs = document.getElementById('hint-ticketws');
    if (hintWs) { hintWs.textContent = '\\u2713 Auto-filled with ' + ticketId; hintWs.classList.add('visible'); }

    UI.toast('All test fields updated with ticket ' + ticketId, 'success');
  },

  createTicket: async function() {
    var amount = parseFloat(document.getElementById('tc-amount').value) || 500;
    var res = await API.post('/api/ticket', { amount: amount }, {});
    this.showResponse('resp-create', res);
    if (res.ok && res.data.ticketId) {
      this.autoFill(res.data.ticketId, res.data.amount || amount);
    }
  },

  smsWebhook: async function() {
    var secret = document.getElementById('tw-secret').value;
    var sms = document.getElementById('tw-sms').value;
    var res = await API.post('/api/webhook', { sms: sms }, { 'X-Webhook-Secret': secret });
    this.showResponse('resp-sms', res);
  },

  kotakWebhook: async function() {
    var secret = document.getElementById('tk-secret').value;
    var sms = document.getElementById('tk-sms').value;
    var res = await API.post('/api/webhook', { sms: sms }, { 'X-Webhook-Secret': secret });
    this.showResponse('resp-kotak', res);
  },

  emailWebhook: async function() {
    var secret = document.getElementById('te-secret').value;
    var from = document.getElementById('te-from').value;
    var subject = document.getElementById('te-subject').value;
    var text = document.getElementById('te-body').value;
    var res = await API.post('/api/email-webhook', { from: from, subject: subject, text: text }, { 'X-Email-Secret': secret });
    this.showResponse('resp-email', res);
  },

  checkStatus: async function() {
    var id = document.getElementById('ts-id').value.trim();
    if (!id) { UI.toast('Enter a ticket ID', 'warn'); return; }
    var res = await API.get('/api/status/' + encodeURIComponent(id));
    this.showResponse('resp-status', res);
  },

  togglePingWs: function() {
    var btn = document.getElementById('ws-ping-btn');
    if (State.pingWs) {
      State.pingWs.close();
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
      return;
    }
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    var ws = new WebSocket(proto + '//' + location.host + '/api/ping');
    State.pingWs = ws;
    btn.textContent = 'Connecting...';
    ws.onopen = function() {
      AppConsole.log('ok', '[WS-Ping] Connected');
      btn.textContent = 'Disconnect';
      btn.classList.add('connected');
      var respEl = document.getElementById('resp-ping');
      respEl.innerHTML = '<div class="test-response-head"><span class="status-pill status-ok">OPEN</span></div><div class="test-response-body">Connected. Waiting for pong...</div>';
      respEl.classList.add('visible');
      ws.send('ping');
    };
    ws.onmessage = function(e) {
      AppConsole.log('info', '[WS-Ping] Received: ' + e.data);
      var respEl = document.getElementById('resp-ping');
      if (respEl) {
        var body = respEl.querySelector('.test-response-body');
        if (body) body.textContent = 'Received: ' + e.data;
      }
    };
    ws.onclose = function() {
      AppConsole.log('warn', '[WS-Ping] Closed');
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
    ws.onerror = function() {
      AppConsole.log('error', '[WS-Ping] Error');
      State.pingWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
  },

  toggleTicketWs: function() {
    var btn = document.getElementById('ws-ticket-btn');
    if (State.ticketWs) {
      State.ticketWs.close();
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
      return;
    }
    var ticketId = document.getElementById('ws-ticket-id').value.trim();
    if (!ticketId) { UI.toast('Enter a ticket ID', 'warn'); return; }
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    var ws = new WebSocket(proto + '//' + location.host + '/api/ws?ticketId=' + encodeURIComponent(ticketId));
    State.ticketWs = ws;
    btn.textContent = 'Connecting...';
    ws.onopen = function() {
      AppConsole.log('ok', '[WS-Ticket] Connected for ' + ticketId);
      btn.textContent = 'Disconnect';
      btn.classList.add('connected');
      var respEl = document.getElementById('resp-ticketws');
      respEl.innerHTML = '<div class="test-response-head"><span class="status-pill status-ok">OPEN</span></div><div class="test-response-body">Listening for payment updates on ' + Util.escape(ticketId) + '...</div>';
      respEl.classList.add('visible');
    };
    ws.onmessage = function(e) {
      AppConsole.log('info', '[WS-Ticket] ' + e.data);
      var respEl = document.getElementById('resp-ticketws');
      if (respEl) {
        try {
          var d = JSON.parse(e.data);
          var body = respEl.querySelector('.test-response-body');
          if (body) body.innerHTML = Util.colorizeJson(d);
          if (d.status === 'paid') UI.toast('Payment confirmed! ' + ticketId, 'success');
        } catch(_) {}
      }
    };
    ws.onclose = function() {
      AppConsole.log('warn', '[WS-Ticket] Closed');
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
    ws.onerror = function() {
      AppConsole.log('error', '[WS-Ticket] Error');
      State.ticketWs = null;
      btn.textContent = 'Connect';
      btn.classList.remove('connected');
    };
  }
};

// ─── CONSOLE ─────────────────────────────────────────────────────────────────
var AppConsole = {
  setFilter: function(level) {
    State.logFilter = level;
    document.querySelectorAll('.console-level-btn').forEach(function(el) {
      el.classList.toggle('active', el.dataset.level === level);
    });
    this.rerender();
  },
  log: function(level, msg, data) {
    var now = new Date();
    var time = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3,'0');
    State.logs.push({ level: level, msg: msg, data: data || null, time: time });
    if (State.logs.length > 500) State.logs.shift();
    if (!State.logFilter || State.logFilter === level) this.appendEntry(State.logs[State.logs.length-1]);
  },
  appendEntry: function(entry) {
    var box = document.getElementById('console-box');
    if (!box) return;
    var div = document.createElement('div');
    div.className = 'log-entry';
    div.innerHTML = '<span class="log-time">' + Util.escape(entry.time) + '</span>'
      + '<span class="log-level log-level-' + entry.level + '">' + entry.level.toUpperCase() + '</span>'
      + '<span class="log-msg">' + Util.escape(entry.msg) + '</span>';
    box.appendChild(div);
    box.scrollTop = box.scrollHeight;
  },
  rerender: function() {
    var box = document.getElementById('console-box');
    if (!box) return;
    box.innerHTML = '';
    var filter = State.logFilter;
    for (var i = 0; i < State.logs.length; i++) {
      if (!filter || State.logs[i].level === filter) this.appendEntry(State.logs[i]);
    }
    box.scrollTop = box.scrollHeight;
  },
  clear: function() {
    State.logs = [];
    var box = document.getElementById('console-box');
    if (box) box.innerHTML = '';
  }
};

// ─── UI ───────────────────────────────────────────────────────────────────────
var UI = {
  showTableLoading: function() {
    var tbody = document.getElementById('ticket-tbody');
    if (!tbody) return;
    var html = '';
    for (var i = 0; i < 8; i++) {
      html += '<tr><td><div class="skeleton" style="width:120px"></div></td>'
        + '<td><div class="skeleton" style="width:60px"></div></td>'
        + '<td><div class="skeleton" style="width:70px"></div></td>'
        + '<td><div class="skeleton" style="width:100px"></div></td>'
        + '<td><div class="skeleton" style="width:120px"></div></td>'
        + '<td><div class="skeleton" style="width:80px"></div></td>'
        + '<td><div class="skeleton" style="width:70px"></div></td>'
        + '<td><div class="skeleton" style="width:80px"></div></td></tr>';
    }
    tbody.innerHTML = html;
  },
  animateCounter: function(el, from, to) {
    var start = null;
    var dur = 400;
    var step = function(ts) {
      if (!start) start = ts;
      var prog = Math.min((ts - start) / dur, 1);
      var ease = 1 - Math.pow(1 - prog, 3);
      el.textContent = Math.round(from + (to - from) * ease);
      if (prog < 1) requestAnimationFrame(step);
      else el.textContent = to;
    };
    requestAnimationFrame(step);
  },
  toast: function(msg, type) {
    var stack = document.getElementById('toast-stack');
    var el = document.createElement('div');
    el.className = 'toast toast-' + (type || 'info');
    el.textContent = msg;
    stack.appendChild(el);
    setTimeout(function() {
      el.style.animation = 'toastOut 0.25s var(--ease-out) forwards';
      setTimeout(function() { if (el.parentNode) el.parentNode.removeChild(el); }, 280);
    }, 3000);
  }
};

// ─── TABS ────────────────────────────────────────────────────────────────────
function switchTab(tab) {
  State.activeTab = tab;
  document.querySelectorAll('.nav-tab').forEach(function(el) {
    el.classList.toggle('active', el.dataset.tab === tab);
  });
  var tabs = ['overview','testing','console'];
  for (var i = 0; i < tabs.length; i++) {
    var el = document.getElementById('tab-' + tabs[i]);
    if (el) el.style.display = (tabs[i] === tab) ? 'block' : 'none';
  }
}

// ─── EVENT WIRING ────────────────────────────────────────────────────────────
function init() {
  // Tab switching
  document.querySelectorAll('.nav-tab').forEach(function(el) {
    el.addEventListener('click', function() { switchTab(el.dataset.tab); });
  });

  // Refresh button
  document.getElementById('refresh-btn').addEventListener('click', function() {
    Dashboard.loadTickets(true, true);
  });

  // Sign out
  document.getElementById('signout-btn').addEventListener('click', function() {
    Auth.logout();
  });

  // Search — debounced DB call
  document.getElementById('search-input').addEventListener('input', function() {
    State.search = this.value;
    clearTimeout(State.searchTimer);
    State.searchTimer = setTimeout(function() {
      Dashboard.loadTickets(true, true);
    }, 450);
  });

  // Date filters
  document.getElementById('date-from').addEventListener('change', function() {
    State.dateFrom = this.value;
    Dashboard.loadTickets(true, true);
  });
  document.getElementById('date-to').addEventListener('change', function() {
    State.dateTo = this.value;
    Dashboard.loadTickets(true, true);
  });
  document.getElementById('date-clear').addEventListener('click', function() {
    State.dateFrom = ''; State.dateTo = '';
    document.getElementById('date-from').value = '';
    document.getElementById('date-to').value = '';
    Dashboard.loadTickets(true, true);
  });

  // Close modal on overlay click
  document.getElementById('modal-overlay').addEventListener('click', function(e) {
    if (e.target === this) Modal.close();
  });

  // Close modal on Escape
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' && document.getElementById('modal-overlay').style.display !== 'none') {
      Modal.close();
    }
  });

  Auth.init();
}

init();

// expose for inline onclick handlers
window.Dashboard = Dashboard;
window.Modal = Modal;
window.Testing = Testing;
window.AppConsole = AppConsole;

})();
</script>
</body>
</html>`;
