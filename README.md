# Payment Gateway Worker (PoC)

This project is a Proof of Concepts (PoC) payment gateway system built with **Cloudflare Workers** (Hono framework) and **Appwrite Database**. It replaces the traditional server-side logic with a serverless edge API.

## Features
- **Ticket Generation**: Creates a unique ticket with a specified amount.
- **Webhook Verification**: Validates payment via SMS forwarding webhook.
- **Secure Processing**: Uses a `WEBHOOK_SECRET` to prevent unauthorized calls.
- **Data Persistence**: Stores tickets and status in Appwrite.
- **Sender Verification**: Extracts and verifies sender name and amount from the payment SMS.

## Tech Stack
- **Runtime**: Cloudflare Workers
- **Framework**: [Hono](https://hono.dev/)
- **Database**: [Appwrite](https://appwrite.io/)
- **Language**: TypeScript

## Setup & configuration

### 1. Install Dependencies
```bash
npm install
```

### 2. Environment Variables
Create a `.dev.vars` file for local development (do not commit this file):

```ini
APPWRITE_ENDPOINT="https://backend.mulearnscet.in/v1"
APPWRITE_PROJECT_ID="YOUR_PROJECT_ID"
APPWRITE_DATABASE_ID="YOUR_DB_ID"
APPWRITE_COLLECTION_ID="YOUR_COLLECTION_ID"
APPWRITE_API_KEY="YOUR_API_KEY"
WEBHOOK_SECRET="YOUR_SECURE_SECRET"
```

For production, verify these secrets are set in your Cloudflare Worker dashboard.

### 3. Run Locally
```bash
npm run dev
# or 
npx wrangler dev
```
The server typically starts at `http://localhost:8787`.

## API Reference

### 1. Generate Ticket
Creates a new payment ticket.
- **Endpoint**: `POST /api/ticket`
- **Body**:
  ```json
  { "amount": 100 }
  ```
- **Response**:
  ```json
  {
    "id": "TICKET1769333...",
    "amount": 100,
    "status": "pending"
  }
  ```

### 2. Payment Webhook
Called by the SMS forwarder app when a payment is received.
- **Endpoint**: `POST /api/webhook?secret=YOUR_SECURE_SECRET`
- **Headers**: `Content-Type: application/json`
- **Body Requirement**:
  The JSON body must contain a field (`sms`, `body`, or `message`) that includes the **Ticket ID** and the **Payment Message**.
  
  **Format Regex**: `"{Name} [has] paid you ₹{Amount}"`

  **Example Payload**:
  ```json
  {
    "sms": "Confirmed payment for TICKET1769333...",
    "body": "Sourav P Bijoy has paid you ₹100"
  }
  ```
- **Logic**:
  1. Validates `ticketId` exists in payload.
  2. Parses `senderName` ("Sourav P Bijoy") and `amount` (100).
  3. Verifies `amount` matches the amount stored in Appwrite for that ticket.
  4. If matched, updates status to `paid` and saves `senderName`.

### 3. Check Status
Checks the status of a specific ticket.
- **Endpoint**: `GET /api/status/:id`
- **Example**: `/api/status/TICKET1769333...`
- **Response**:
  ```json
  {
    "ticketId": "TICKET1769333...",
    "status": "paid",
    "amount": 100,
    "senderName": "Sourav P Bijoy"
  }
  ```

## Deployment
To deploy to Cloudflare Workers:
```bash
npx wrangler deploy
```
