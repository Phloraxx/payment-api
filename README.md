# Payment Gateway v2

Zero-fee UPI payment matching service for college events. SQLite is the source of truth, Appwrite is an optional read replica, and Dynamic Decimal Matching assigns exact payable amounts.

## Development

```bash
npm install
npm run dev
```

Admin SPA:

```bash
npm --prefix src/admin install
npm run admin:dev
```

## Verification

```bash
npm run lint
npm run typecheck
npm test
npm run admin:build
```
