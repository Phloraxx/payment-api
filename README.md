# 💳 Stateless UPI Payment Gateway API

A high-performance, stateless, and secure payment gateway PoC built on **Cloudflare Workers** (Hono) and **Appwrite**. Designed for high-concurrency event registration where traditional payment gateway fees or infrastructure costs are a barrier.

---

## 🚀 Unique Features

### 1. Dynamic Decimal Matching (DDM)

Solves the "Single VPA" problem. Instead of a custom UPI ID for every user, we use a single merchant VPA and uniquely identify payments by allocating specific decimals:

- **Base Amount**: ₹100
- **Allocation**: User A pays ₹100.01, User B pays ₹100.02.
- **Verification**: The system parses banking emails in real-time and matches the decimal (`.02`) to the specific transaction.

### 2. Stateless Refresh Strategy

Eliminates `localStorage` dependency. The transaction state is persisted in the URL and verified against the backend on every load:

- Users can refresh the page, switch browsers, or lose connection.
- Reading the `ticketId` from the URL allows the frontend to instantly resume status and re-render the QR code.

### 3. Edge-Optimized Lazy Evaluation

Zero-cost cleanup of expired tickets.

- **5-Minute Window**: Tickets expire automatically.
- **Lazy Cleanup**: No cron jobs needed. The system cleans up stale records passively during active requests, saving compute and DB units.

---

## 🔒 Security Architecture

### Edge-Proxied WebSockets (Zero Backend Exposure)

Most Appwrite implementations use collection-level permissions which are insecure for payments (everyone hears everyone's events). We solved this with an **Edge WebSocket Proxy**:

1.  **Strict Security**: `Permission.read(Role.any())` is entirely removed from the database collection. No one on the internet can query your database anonymously.
2.  **Internal ID Protection**: The API **never** returns the internal Appwrite Document `$id` to the frontend.
3.  **Scoped Read**: The Cloudflare Worker handles the Appwrite Realtime connection using an admin `APPWRITE_API_KEY`. It listens to the document and acts as a strict middleman for the frontend.
4.  **Frontend**: Subscribes directly to a custom Cloudflare WebSocket (`/api/ws`), hiding the backend infrastructure and taking advantage of Cloudflare's massive DDoS protection.

---

## 📡 API Reference

### 1. Create Payment Ticket

`POST /api/ticket`

- **Request**: `{ "amount": 100 }`
- **Success Response**:
  ```json
  {
    "ticketId": "TICKET1709123456789", // PUBLIC REFERENCE (For URL)
    "amount": 100.03, // ALLOCATED AMOUNT
    "status": "pending",
    "createdAt": "2026-03-05T..."
  }
  ```

### 2. Verify/Resume Status

`GET /api/status/:ticketId`

- **Use Case**: Used on page load to re-fetch transaction details from a URL `ticketId`.

---

## ⚡ Frontend Integration Pattern

The frontend heavily relies on Cloudflare's Edge WebSockets. You do **not** need the Appwrite SDK on the frontend.

```javascript
/* Secure, Stateless Frontend Loop */

async function startPayment() {
  const params = new URLSearchParams(window.location.search);
  const ticketId = params.get("ticketId");

  // 1. Fetch current state (Stateless Resume)
  const res = await fetch(
    `/api/status/${ticketId}`,
  );
  const data = await res.json();

  if (data.status === "paid") return handleSuccess();

  // 2. Generate UPI locally (No Backend dependency for QR)
  const vpa = "merchant@upi";
  const upiStr = `upi://pay?pa=${vpa}&am=${data.amount}&tn=${data.ticketId}`;
  renderQR(upiStr);

  // 3. Connect to the Cloudflare WebSocket Proxy
  const wsUrl = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/api/ws?ticketId=${ticketId}`;
  const socket = new WebSocket(wsUrl);

  socket.onmessage = (event) => {
    const payload = JSON.parse(event.data);

    if (payload.type === "payment_update" && payload.status === "paid") {
      console.log("Payment successful at:", payload.paidAt);
      socket.close();
      handleSuccess();
    }
  };
}
```

---

## 🛠️ Environment Setup

Required variables for the Worker (`.dev.vars` or Wrangler:

- `APPWRITE_ENDPOINT`, `APPWRITE_PROJECT_ID`, `APPWRITE_DATABASE_ID`, `APPWRITE_COLLECTION_ID`
- `APPWRITE_API_KEY`: Must have `documents.write` and `documents.read` permissions.
- `WEBHOOK_SECRET`: For internal authentication.
- `EMAIL_SECRET`: Shared secret with the Cloudflare Email Worker.

---

## 🚀 Deployment

```bash
npx wrangler deploy
```

---

_Built with ❤️ for Mulearn SCET & IEEESahrdaya_
