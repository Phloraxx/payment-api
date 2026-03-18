import { Hono } from "hono";
import { cors } from "hono/cors";
import { upgradeWebSocket } from "hono/cloudflare-workers";
import { AppwriteService, AdminUpdateFields, Env, toCents } from "./lib/appwrite";
import { ADMIN_HTML } from "./admin-html";
import { EmailParser } from "./lib/emailParser";

// Temporary in-memory lock to prevent exact same decimal allocation on the same Edge node
// This provides a "best effort" lock to prevent concurrent requests in the same colo from snagging the same decimal.
const localDecimalLocks = new Set<string>();

/**
 * Constant-time string comparison to prevent timing attacks.
 */
function timingSafeEqual(a: string, b: string): boolean {
    const aLen = a.length;
    const bLen = b.length;
    let result = aLen ^ bLen;
    const len = Math.min(aLen, bLen);
    for (let i = 0; i < len; i++) {
        result |= a.charCodeAt(i) ^ b.charCodeAt(i);
    }
    return result === 0;
}

async function requireAdminAuth(c: any, next: any) {
    const password = c.req.header("X-Admin-Password") || "";
    const adminPassword: string = (c.env as Env).ADMIN_PASSWORD || "";
    if (!adminPassword || !timingSafeEqual(password, adminPassword)) {
        return c.json({ error: "Unauthorized" }, 401);
    }
    return next();
}

const app = new Hono<{ Bindings: Env }>();

app.use("/*", async (c, next) => {
    const allowedOrigin = c.env.ALLOWED_ORIGIN || "*";
    const corsMiddleware = cors({
        origin: allowedOrigin.includes(",") ? allowedOrigin.split(",") : allowedOrigin,
    });
    return corsMiddleware(c, next);
});

app.get("/", (c) => {
    return c.text("Payment Gateway API is running!");
});

app.get(
    "/api/ping",
    upgradeWebSocket((c) => {
        return {
            onMessage(event, ws) {
                console.log("PING RX");
                ws.send("pong");
            },
            onOpen() {
                console.log("PING WS OPEN");
            }
        };
    })
);

// ---------------------------------------------------------------------------
// POST /api/ticket  — create a new pending payment ticket
// ---------------------------------------------------------------------------
app.post("/api/ticket", async (c) => {
    try {
        const body = await c.req.json();
        const baseAmount = body.amount;

        if (!baseAmount || typeof baseAmount !== "number") {
            return c.json({ error: "Invalid amount" }, 400);
        }

        const appwrite = new AppwriteService(c.env);

        // 1. Get currently allocated decimals for this base amount from DB
        const dbAllocatedDecimals = await appwrite.getPendingDecimalsForAmount(
            Math.floor(baseAmount),
        );

        // 2. Find the lowest available decimal from 00 to 99.
        // We start at a random index to reduce DB lock contention.
        let availableDecimal = -1;
        const startOffset = Math.floor(Math.random() * 100);

        for (let i = 0; i < 100; i++) {
            const candidateDecimal = (startOffset + i) % 100;
            const lockKey = `${Math.floor(baseAmount)}_${candidateDecimal}`;

            if (
                !dbAllocatedDecimals.includes(candidateDecimal) &&
                !localDecimalLocks.has(lockKey)
            ) {
                // Determine lock via Database (100% atomic constraint on Document ID)
                const lockAcquired = await appwrite.claimDatabaseLock(
                    Math.floor(baseAmount),
                    candidateDecimal,
                );

                if (lockAcquired) {
                    availableDecimal = candidateDecimal;
                    // Temporarily lock this decimal in memory (auto-expire after 5 mins)
                    localDecimalLocks.add(lockKey);
                    setTimeout(() => localDecimalLocks.delete(lockKey), 5 * 60 * 1000);
                    break;
                }
            }
        }

        if (availableDecimal === -1) {
            return c.json(
                {
                    error:
                        "System busy: Too many concurrent transactions for this amount. Please try again later.",
                },
                503,
            );
        }

        // 3. Construct final decimal amount (e.g. 3 + 0.02 = 3.02)
        const finalAmount = Math.floor(baseAmount) + availableDecimal / 100;

        // 4. Construct Ticket ID ending exactly with that 2-digit decimal
        const timestamp = Date.now().toString();
        // Chop off the last two digits of the timestamp and replace them with the padded decimal
        const prefix = timestamp.slice(0, -2);
        const decimalStr = availableDecimal.toString().padStart(2, "0");
        const ticketId = `TICKET${prefix}${decimalStr}`;

        // 5. Store ticket
        const createdDoc = await appwrite.createTicket(ticketId, finalAmount);

        return c.json({
            ticketId: ticketId, // THE READABLE TICKET REFERENCE
            amount: finalAmount,
            status: "pending",
            createdAt: createdDoc.$createdAt,
        });
    } catch (error) {
        console.error("Error creating ticket:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// ---------------------------------------------------------------------------
// POST /api/webhook  — legacy SMS / generic webhook handler
// ---------------------------------------------------------------------------
app.post("/api/webhook", async (c) => {
    try {
        const rawBody = await c.req.text();
        console.log("--- WEBHOOK RECEIVED ---");
        console.log("TIMESTAMP:", new Date().toISOString());
        console.log("RAW BODY:", rawBody);

        let body;
        try {
            body = JSON.parse(rawBody);
        } catch (e) {
            console.log(
                "Could not parse JSON body, assuming raw text or URL encoded.",
            );
            body = { sms: rawBody };
        }

        const secret = c.req.header("X-Webhook-Secret") || c.req.query("secret") || body.secret_key;
        if (!secret || !timingSafeEqual(secret, c.env.WEBHOOK_SECRET)) {
            console.log("Unauthorized Webhook Attempt");
            return c.json({ error: "Unauthorized" }, 401);
        }

        // Combine potential fields to search for Ticket ID
        const content =
            (
                (body.sms || "") +
                " " +
                (body.body || "") +
                " " +
                (body.message || "")
            ).trim() || rawBody;

        console.log("SEARCH CONTENT:", content);

        // Regex to find "TICKET" followed by numbers
        const ticketMatch = content.match(/TICKET(\d+)/);

        const paymentSource = body.body || content;
        const paymentMatch = paymentSource.match(
            /([a-zA-Z0-9\s\.]+?) (?:has )?paid you ₹(\d+(\.\d{1,2})?)/i,
        );

        let foundId = null;
        let status = "ignored";
        let updatedDoc = null;

        if (ticketMatch && paymentMatch) {
            foundId = ticketMatch[0];
            const senderName = paymentMatch[1].trim();
            const paidAmount = parseFloat(paymentMatch[2]);

            console.log("FOUND TICKET ID:", foundId);
            console.log("SENDER:", senderName);
            console.log("PAID AMOUNT:", paidAmount);

            const appwrite = new AppwriteService(c.env);

            const ticket = await appwrite.getTicketStatus(foundId);

            if (!ticket) {
                console.log(`Ticket ${foundId} NOT FOUND`);
                status = "ticket_not_found";
            } else if (toCents(ticket.amount) !== toCents(paidAmount)) {
                console.log(
                    `AMOUNT MISMATCH: Ticket requires ${ticket.amount}, but received ${paidAmount} `,
                );
                status = "amount_mismatch";
            } else {
                if (ticket.status === "paid") {
                    console.log(`Ticket ${foundId} ALREADY PAID, updating sender name anyway`);
                }
                updatedDoc = await appwrite.markAsPaid(foundId, senderName);
                if (updatedDoc) {
                    console.log(`Ticket ${foundId} MARKED AS PAID`);
                    status = "success";

                    // Free the decimal lock immediately so it can be reused
                    const baseAmount = Math.floor(ticket.amount);
                    const decPart = Math.round((ticket.amount - baseAmount) * 100);
                    localDecimalLocks.delete(`${baseAmount}_${decPart}`);
                    c.executionCtx.waitUntil(
                        appwrite.releaseDatabaseLock(baseAmount, decPart).catch(() => null)
                    );
                } else {
                    status = "update_failed";
                }
            }
        } else {
            const kotakParsed = EmailParser.parseKotakSms(content);
            if (kotakParsed && kotakParsed.paidAmount !== null && kotakParsed.intPart !== null && kotakParsed.decPart !== null) {
                console.log("KOTAK SMS DETECTED WITHOUT EXPLICIT TICKET ID");
                const { paidAmount, decPart, intPart, rrn, upiId } = kotakParsed;

                console.log("PAID AMOUNT:", paidAmount, "| DEC PART:", decPart);
                console.log("UPI ID:", upiId, "| RRN:", rrn);

                const appwrite = new AppwriteService(c.env);
                const candidates = await appwrite.listRecentTickets(20);

                const now = Date.now();
                const FIVE_MIN_MS = 5 * 60 * 1000;
                const cutoffIso = new Date(now - FIVE_MIN_MS).toISOString();

                let matchedTicket = null;
                let matchedTicketId = null;

                for (const ticket of candidates) {
                    if (ticket.ticketId.startsWith("lock_")) continue;

                    if (ticket.createdAt < cutoffIso) continue;

                    const numericPart = ticket.ticketId.replace(/^TICKET/i, "");
                    const ticketSuffix = parseInt(numericPart.slice(-2), 10);

                    if (ticketSuffix !== decPart) continue;
                    if (toCents(Math.floor(ticket.amount)) !== toCents(intPart)) continue;

                    matchedTicket = ticket;
                    matchedTicketId = ticket.ticketId;
                    break;
                }

                if (matchedTicket && matchedTicketId) {
                    updatedDoc = await appwrite.markAsPaid(matchedTicketId, undefined, rrn ?? undefined, upiId ?? undefined);
                    if (updatedDoc) {
                        console.log(`Ticket ${matchedTicketId} MARKED AS PAID via Kotak SMS. RRN: ${rrn}`);
                        status = "success";
                        foundId = matchedTicketId;

                        localDecimalLocks.delete(`${intPart}_${decPart}`);
                        c.executionCtx.waitUntil(
                            appwrite.releaseDatabaseLock(intPart, decPart).catch(() => null)
                        );
                    } else {
                        status = "update_failed";
                    }
                } else {
                    console.log(`No matching pending ticket found for Kotak amount ₹${paidAmount} (dec = ${decPart})`);
                    status = "no_matching_ticket";
                }
            } else {
                console.log("INVALID SMS FORMAT: Missing Ticket ID or Payment Details");
                if (!ticketMatch) console.log(" - Missing Ticket ID");
                if (!paymentMatch)
                    console.log(" - Missing Payment Details (Name/Amount)");
                status = "invalid_format";
            }
        }

        return c.json({
            status: "received",
            processed_id: foundId,
            action: status,
        });
    } catch (error) {
        console.error("Webhook error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// ---------------------------------------------------------------------------
// POST /api/email-webhook  — Cloudflare Email Worker forwarded notification
//
// The Cloudflare Email Worker (email.ts) receives the inbound email from
// slice <noreply@slice.bank.in>, extracts relevant fields, and POSTs them here
// as JSON:
//
//   {
//     "secret":   "<EMAIL_SECRET>",
//     "from":     "noreply@slice.bank.in",
//     "subject":  "Received ₹1.02 via UPI",
//     "text":     "<plain-text body>",
//     "html":     "<html body>"
//   }
//
// Verification steps performed here:
//   1. Shared-secret check (EMAIL_SECRET)
//   2. Sender domain check  — must be @slice.bank.in
//   3. Subject check        — must contain "Received ₹" and "via UPI"
//   4. Extract amount (x.yy), ticket suffix, RRN, sender name from body
//   5. Decimal ticket-ID check — the .yy cents value must equal the last 2
//      digits of the numeric part of the ticket ID  (within 00-99)
//   6. 5-minute window check — ticket must have been created ≤ 5 min ago
//   7. Amount integrity check — paid amount integer part must match ticket
//   8. Mark as paid + store RRN
// ---------------------------------------------------------------------------
app.post("/api/email-webhook", async (c) => {
    try {
        const rawBody = await c.req.text();
        console.log("--- EMAIL WEBHOOK RECEIVED ---");
        console.log("TIMESTAMP:", new Date().toISOString());

        let body: Record<string, string>;
        try {
            body = JSON.parse(rawBody);
        } catch {
            return c.json({ error: "Invalid JSON" }, 400);
        }

        // ── 1. Shared-secret authentication ──────────────────────────────────
        const secret = c.req.header("X-Email-Secret") || c.req.query("secret") || body.secret;
        if (!secret || !timingSafeEqual(secret, c.env.EMAIL_SECRET)) {
            console.log("Unauthorized email webhook attempt");
            return c.json({ error: "Unauthorized" }, 401);
        }

        const from = (body.from || "").toLowerCase().trim();
        const subject = (body.subject || "").trim();
        const text = (body.text || body.html || "")
            .replace(/=\r?\n/g, "")
            .replace(/=3D/g, "=");

        console.log("FROM:", from, "| SUBJECT:", subject);

        // ── 2. Sender check (disabled due to Gmail Forwarding) ───────────────
        // We cannot rely on 'from' because Gmail forwarding changes the envelope sender.
        // We will rely on Subject and strict Decimal Ticket Matching instead.

        // ── 3. Subject check ─────────────────────────────────────────────────
        if (
            !subject.toLowerCase().includes("received") ||
            !subject.toLowerCase().includes("via upi")
        ) {
            console.log("Rejected: subject does not match UPI credit pattern");
            return c.json({ status: "ignored", reason: "subject_mismatch" });
        }

        // ── 4. Use EmailParser to Extract amount, RRN, and sender name ───────
        const parsedEmail = EmailParser.parseSliceEmail(subject, text);

        if (
            parsedEmail.paidAmount === null ||
            parsedEmail.decPart === null ||
            parsedEmail.intPart === null
        ) {
            console.log("Could not extract amount from email body or subject");
            return c.json({ status: "ignored", reason: "amount_not_found" });
        }

        const { paidAmount, decPart, intPart, rrn, senderName } = parsedEmail;

        console.log("PAID AMOUNT:", paidAmount, "| DEC PART:", decPart);
        console.log("RRN:", rrn);
        console.log("SENDER NAME:", senderName);

        // ── 5. Decimal ticket-ID check ───────────────────────────────────────
        //
        //  The customer's ticket ID ends with a numeric timestamp, e.g.
        //    TICKET1709123456789
        //  The last two digits of that timestamp (89) must equal the decimal
        //  part of the paid amount (e.g. ₹x.89).
        //
        //  The merchant instructs the customer to transfer the exact amount
        //  shown on the payment page, which is constructed as:
        //    base_amount_integer + (ticket_suffix_2_digits / 100)
        //  This makes each payment uniquely attributable to one ticket.
        //
        const appwrite = new AppwriteService(c.env);

        // We don't know the ticket ID yet — look up all recently-created
        // pending tickets and find the one whose suffix matches the decimal.
        //
        // Strategy: query Appwrite for pending tickets created in last 6 min,
        // then filter by decimal suffix match.
        //
        // Because Appwrite free tier may not support range queries on $createdAt
        // easily, we fetch recent pending tickets (limit 20) and filter in-code.
        const candidates = await appwrite.listRecentPendingTickets(20);

        const now = Date.now();
        const FIVE_MIN_MS = 5 * 60 * 1000;
        const cutoffIso = new Date(now - FIVE_MIN_MS).toISOString();

        let matchedTicket:
            | Awaited<ReturnType<typeof appwrite.listRecentPendingTickets>>[0]
            | null = null;
        let matchedTicketId: string | null = null;

        for (const ticket of candidates) {
            // Ignore lock documents during ticket matching
            if (ticket.ticketId.startsWith("lock_")) continue;

            // ── 6. 5-minute window check ─────────────────────────────────────
            if (ticket.createdAt < cutoffIso) {
                console.log(
                    `Ticket ${ticket.ticketId}: outside 5 - minute window, skipping`,
                );
                continue;
            }

            // ── Decimal suffix check ──────────────────────────────────────────
            //  Extract trailing digits of the ticket numeric part
            const numericPart = ticket.ticketId.replace(/^TICKET/i, "");
            const ticketSuffix = parseInt(numericPart.slice(-2), 10); // last 2 digits

            if (ticketSuffix !== decPart) {
                console.log(
                    `Ticket ${ticket.ticketId}: suffix ${ticketSuffix} !== dec ${decPart}, skipping`,
                );
                continue;
            }

            // ── 7. Amount integer check ───────────────────────────────────────
            //  The ticket stores the full expected amount (integer), e.g. 1.
            //  The paid amount must match (floor-comparing integer part).
            if (toCents(Math.floor(ticket.amount)) !== toCents(intPart)) {
                console.log(
                    `Ticket ${ticket.ticketId}: integer amount mismatch(expected ${ticket.amount}, got ${intPart}), skipping`,
                );
                continue;
            }

            if (ticket.status === "paid") {
                console.log(`Ticket ${ticket.ticketId}: already paid, skipping`);
                continue;
            }

            matchedTicket = ticket;
            matchedTicketId = ticket.ticketId;
            break;
        }

        if (!matchedTicket || !matchedTicketId) {
            console.log(
                `No matching pending ticket found for amount ₹${paidAmount} (dec = ${decPart})`,
            );
            return c.json({
                status: "ignored",
                reason: "no_matching_ticket",
                paid_amount: paidAmount,
                dec_part: decPart,
            });
        }

        // ── 8. Mark ticket as paid ────────────────────────────────────────────
        const updatedDoc = await appwrite.markAsPaid(
            matchedTicketId,
            senderName,
            rrn ?? undefined,
        );

        if (updatedDoc) {
            console.log(
                `Ticket ${matchedTicketId} MARKED AS PAID via email. RRN: ${rrn}`,
            );

            // Free the decimal lock immediately so it can be reused
            localDecimalLocks.delete(`${intPart}_${decPart}`);
            c.executionCtx.waitUntil(
                appwrite.releaseDatabaseLock(intPart, decPart).catch(() => null)
            );

            const { id, ...sanitized } = updatedDoc as any;
            return c.json(sanitized);
        } else {
            return c.json({ status: "error", reason: "update_failed" }, 500);
        }
    } catch (error) {
        console.error("Email webhook error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// ---------------------------------------------------------------------------
// GET /api/status/:id  — poll ticket status
// ---------------------------------------------------------------------------
app.get("/api/status/:id", async (c) => {
    const id = c.req.param("id");
    const appwrite = new AppwriteService(c.env);
    const status = await appwrite.getTicketStatus(id);

    if (!status) {
        return c.json({ status: "not_found" }, 404);
    }

    const { id: internalId, ...sanitized } = status as any;
    return c.json(sanitized);
});

// ---------------------------------------------------------------------------
// 🌟 SECURE WEBSOCKET PROXY (CLOUDFLARE -> APPWRITE REALTIME)
// Prevents frontend from connecting directly to the database.
//
// Endpoint: wss://payment-api.nerdpixel.workers.dev/api/ws?ticketId=TICKET...
// ---------------------------------------------------------------------------
app.get("/api/ws", async (c) => {
    const ticketId = (c.req.query("ticketId") || "").trim();
    if (!ticketId || !/^TICKET\d+$/.test(ticketId)) {
        return c.text("Invalid ticketId", 400);
    }

    // Use WebSocketPair so we can attach immediately.
    // Hono's Cloudflare WS helper does not support an onOpen event.
    const pair = new WebSocketPair();
    const client = pair[0];
    const server = pair[1];
    server.accept();

    let upstreamWs: WebSocket | null = null;
    let closing = false;

    const safeCloseAll = () => {
        if (closing) return;
        closing = true;
        try {
            server.close();
        } catch { }
        try {
            upstreamWs?.close();
        } catch { }
    };

    server.addEventListener("close", safeCloseAll);
    server.addEventListener("error", safeCloseAll);
    server.addEventListener("message", (event) => {
        try {
            if (event.data === "ping") server.send("pong");
        } catch { }
    });

    // Immediately send a status snapshot so the frontend can connect even after payment is already paid.
    const appwrite = new AppwriteService(c.env);
    const snapshotPromise = appwrite
        .getTicketStatus(ticketId)
        .then((status) => {
            if (!status) return;
            try {
                server.send(
                    JSON.stringify({
                        type: "payment_update",
                        status: status.status,
                        paidAt: status.paidAt || null,
                    }),
                );
            } catch { }
        })
        .catch(() => null);
    c.executionCtx?.waitUntil(snapshotPromise);

    const appwriteHost = c.env.APPWRITE_ENDPOINT.replace(/^https?:\/\//, "").split("/")[0];
    const channel = `databases.${c.env.APPWRITE_DATABASE_ID}.collections.${c.env.APPWRITE_COLLECTION_ID}.documents.${ticketId}`;
    const appwriteWsUrl = `wss://${appwriteHost}/v1/realtime?project=${c.env.APPWRITE_PROJECT_ID}&channels[]=${encodeURIComponent(
        channel,
    )}`;

    const upstreamReady = fetch(appwriteWsUrl.replace("wss://", "https://"), {
        headers: {
            Upgrade: "websocket",
            "X-Appwrite-Project": c.env.APPWRITE_PROJECT_ID,
            "X-Appwrite-Key": c.env.APPWRITE_API_KEY,
        },
    })
        .then((res) => {
            const ws = res.webSocket;
            if (!ws) throw new Error("Appwrite handshake failed");
            ws.accept();
            upstreamWs = ws;
            return ws;
        })
        .then((ws) => {
            ws.addEventListener("message", (msg) => {
                try {
                    const envelope = JSON.parse(msg.data as string);
                    if (envelope?.type !== "event") return;

                    const doc = envelope?.data?.payload;
                    if (!doc) return;

                    const docId = doc.$id || doc.ticketId;
                    const status = doc.status;
                    if (docId !== ticketId || !status) return;

                    server.send(
                        JSON.stringify({
                            type: "payment_update",
                            status,
                            paidAt: doc.paidAt || null,
                        }),
                    );
                } catch { }
            });
            ws.addEventListener("close", safeCloseAll);
            ws.addEventListener("error", safeCloseAll);
        })
        .catch((err) => {
            console.error(`[WS-PROXY] Upstream fatal: ${err}`);
            safeCloseAll();
        });

    c.executionCtx?.waitUntil(upstreamReady);

    return new Response(null, { status: 101, webSocket: client });
});

// ---------------------------------------------------------------------------
// GET /admin  — Admin Dashboard HTML
// ---------------------------------------------------------------------------
app.get("/admin", (c) => {
    return c.html(ADMIN_HTML);
});

// ---------------------------------------------------------------------------
// ADMIN API — all routes protected by X-Admin-Password header
// ---------------------------------------------------------------------------

// GET /api/admin/tickets?page=0&pageSize=10&search=X&status=paid&dateFrom=2024-01-01&dateTo=2024-01-31
app.get("/api/admin/tickets", requireAdminAuth, async (c) => {
    try {
        const page     = Math.max(0, parseInt(c.req.query("page") || "0"));
        const pageSize = Math.min(Math.max(1, parseInt(c.req.query("pageSize") || "10")), 100);
        const search   = (c.req.query("search") || "").toLowerCase().trim();
        const statusFilter = c.req.query("status") || "";
        const dateFrom = c.req.query("dateFrom") || "";
        const dateTo   = c.req.query("dateTo") || "";

        // Convert date strings to ISO bounds (treat as UTC)
        const isoFrom = dateFrom ? new Date(dateFrom + "T00:00:00.000Z").toISOString() : undefined;
        const isoTo   = dateTo   ? new Date(dateTo   + "T23:59:59.999Z").toISOString() : undefined;

        const appwrite = new AppwriteService(c.env);
        let tickets = await appwrite.listAllTickets({
            statusFilter: statusFilter || undefined,
            dateFrom: isoFrom,
            dateTo: isoTo,
        });


        if (!tickets) tickets = [];
        // Text search in memory (no full-text index required)
        if (search) {
            tickets = tickets.filter((t) =>
                t.ticketId.toLowerCase().includes(search) ||
                (t.senderName || "").toLowerCase().includes(search) ||
                (t.upiId || "").toLowerCase().includes(search) ||
                (t.rrn || "").toLowerCase().includes(search) ||
                (t.amount != null && t.amount.toString().includes(search))
            );
        }

        const total     = tickets.length;
        const paid      = tickets.filter((t) => t.status === "paid").length;
        const pending   = tickets.filter((t) => t.status === "pending").length;
        const cancelled = tickets.filter((t) => t.status === "cancelled").length;

        const start      = page * pageSize;
        const pageSlice  = tickets.slice(start, start + pageSize);
        const hasMore    = start + pageSize < total;

        return c.json({
            tickets: pageSlice,
            stats:   { total, paid, pending, cancelled },
            hasMore,
            page,
            pageSize,
        });
    } catch (error) {
        console.error("Admin list tickets error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// ---------------------------------------------------------------------------
// GET /api/admin/ws  — Admin collection-wide realtime WebSocket
// Auth via ?pw= query param (WebSocket clients cannot send custom headers)
// ---------------------------------------------------------------------------
app.get("/api/admin/ws", async (c) => {
    const pw = c.req.query("pw") || "";
    const adminPassword: string = (c.env as Env).ADMIN_PASSWORD || "";
    if (!adminPassword || !timingSafeEqual(pw, adminPassword)) {
        return c.text("Unauthorized", 401);
    }

    const pair   = new WebSocketPair();
    const client = pair[0];
    const server = pair[1];
    server.accept();

    let upstreamWs: WebSocket | null = null;
    let closing = false;

    const safeCloseAll = () => {
        if (closing) return;
        closing = true;
        try { server.close(); } catch { /* ignore */ }
        try { upstreamWs?.close(); } catch { /* ignore */ }
    };

    server.addEventListener("close", safeCloseAll);
    server.addEventListener("error", safeCloseAll);
    server.addEventListener("message", (event) => {
        try { if (event.data === "ping") server.send("pong"); } catch { /* ignore */ }
    });

    const appwriteHost = c.env.APPWRITE_ENDPOINT.replace(/^https?:\/\//, "").split("/")[0];
    // Subscribe to all document events in the payment collection
    const channel = `databases.${c.env.APPWRITE_DATABASE_ID}.collections.${c.env.APPWRITE_COLLECTION_ID}.documents`;
    const appwriteWsUrl = `wss://${appwriteHost}/v1/realtime?project=${c.env.APPWRITE_PROJECT_ID}&channels[]=${encodeURIComponent(channel)}`;

    const upstreamReady = fetch(appwriteWsUrl.replace("wss://", "https://"), {
        headers: {
            Upgrade: "websocket",
            "X-Appwrite-Project": c.env.APPWRITE_PROJECT_ID,
            "X-Appwrite-Key": c.env.APPWRITE_API_KEY,
        },
    })
        .then((res) => {
            const ws = res.webSocket;
            if (!ws) throw new Error("Appwrite admin WS handshake failed");
            ws.accept();
            upstreamWs = ws;
            return ws;
        })
        .then((ws) => {
            ws.addEventListener("message", (msg) => {
                try {
                    const envelope = JSON.parse(msg.data as string);
                    if (envelope?.type !== "event") return;

                    const doc    = envelope?.data?.payload;
                    if (!doc) return;

                    // Filter out internal lock documents
                    if (doc.ticketId?.startsWith("lock_")) return;

                    const events: string[] = envelope?.data?.events || [];
                    let action = "update";
                    if (events.some((e: string) => e.endsWith(".create"))) action = "create";
                    else if (events.some((e: string) => e.endsWith(".delete"))) action = "delete";

                    server.send(JSON.stringify({
                        type: "ticket_update",
                        action,
                        ticket: {
                            id:         doc.$id,
                            ticketId:   doc.ticketId,
                            amount:     doc.amount,
                            status:     doc.status,
                            createdAt:  doc.$createdAt,
                            senderName: doc.senderName ?? null,
                            rrn:        doc.rrn ?? null,
                            paidAt:     doc.paidAt ?? null,
                            upiId:      doc.upiId ?? null,
                        },
                    }));
                } catch { /* ignore parse errors */ }
            });
            ws.addEventListener("close", safeCloseAll);
            ws.addEventListener("error", safeCloseAll);
        })
        .catch((err) => {
            console.error(`[ADMIN-WS] Upstream fatal: ${err}`);
            safeCloseAll();
        });

    c.executionCtx?.waitUntil(upstreamReady);
    return new Response(null, { status: 101, webSocket: client });
});

// POST /api/admin/tickets/:id/mark-paid  (must be before /:id GET/PATCH to avoid route conflict)
app.post("/api/admin/tickets/:id/mark-paid", requireAdminAuth, async (c) => {
    try {
        const id = c.req.param("id");
        const body = await c.req.json().catch(() => ({})) as Record<string, string>;
        const fields: AdminUpdateFields = {
            status: "paid",
            paidAt: new Date().toISOString(),
        };
        if (body.senderName) fields.senderName = body.senderName;
        if (body.rrn)        fields.rrn = body.rrn;
        if (body.upiId)      fields.upiId = body.upiId;

        const appwrite = new AppwriteService(c.env);
        const updated = await appwrite.updateTicket(id, fields);
        if (!updated) return c.json({ error: "Update failed" }, 500);
        return c.json(updated);
    } catch (error) {
        console.error("Admin mark-paid error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// POST /api/admin/tickets/:id/cancel
app.post("/api/admin/tickets/:id/cancel", requireAdminAuth, async (c) => {
    try {
        const id = c.req.param("id");
        const appwrite = new AppwriteService(c.env);
        const updated = await appwrite.updateTicket(id, { status: "cancelled" });
        if (!updated) return c.json({ error: "Update failed" }, 500);
        return c.json(updated);
    } catch (error) {
        console.error("Admin cancel error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// GET /api/admin/tickets/:id
app.get("/api/admin/tickets/:id", requireAdminAuth, async (c) => {
    try {
        const id = c.req.param("id");
        const appwrite = new AppwriteService(c.env);
        const ticket = await appwrite.getTicketStatus(id);
        if (!ticket) return c.json({ error: "Not found" }, 404);
        return c.json(ticket);
    } catch (error) {
        console.error("Admin get ticket error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

// PATCH /api/admin/tickets/:id
app.patch("/api/admin/tickets/:id", requireAdminAuth, async (c) => {
    try {
        const id = c.req.param("id");
        let body: Record<string, unknown>;
        try {
            body = await c.req.json();
        } catch {
            return c.json({ error: "Invalid JSON body" }, 400);
        }

        const allowedKeys = ["status", "senderName", "rrn", "upiId", "amount", "paidAt"];
        const fields: AdminUpdateFields = {};
        for (const key of allowedKeys) {
            if (key in body) (fields as any)[key] = body[key];
        }

        const appwrite = new AppwriteService(c.env);
        const updated = await appwrite.updateTicket(id, fields);
        if (!updated) return c.json({ error: "Update failed or no fields provided" }, 400);
        return c.json(updated);
    } catch (error) {
        console.error("Admin patch ticket error:", error);
        return c.json({ error: "Internal Server Error" }, 500);
    }
});

export default {
    fetch: app.fetch,
    async email(message: any, env: Env, ctx: any) {
        try {
            console.log("Received email from:", message.from, "to:", message.to);
            const rawEmail = await new Response(message.raw).text();

            const payload = {
                secret: env.EMAIL_SECRET,
                from: message.from,
                subject: message.headers.get("Subject") || "",
                text: rawEmail,
            };

            const req = new Request("http://localhost/api/email-webhook", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            });

            const res = await app.fetch(req, env, ctx);
            console.log("Email webhook local processing status:", res.status);
            const resultText = await res.text();
            console.log("Email webhook result:", resultText);
        } catch (e) {
            console.error("Email worker error:", e);
        }
    },
};
