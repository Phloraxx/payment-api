import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { AppwriteService, Env } from './lib/appwrite';

const app = new Hono<{ Bindings: Env }>();

app.use('/*', cors());

app.get('/', (c) => {
    return c.text('Payment Gateway API is running!');
});

// ---------------------------------------------------------------------------
// POST /api/ticket  — create a new pending payment ticket
// ---------------------------------------------------------------------------
app.post('/api/ticket', async (c) => {
    try {
        const body = await c.req.json();
        const amount = body.amount;

        if (!amount || typeof amount !== 'number') {
            return c.json({ error: 'Invalid amount' }, 400);
        }

        const timestamp = Date.now();
        const ticketId = `TICKET${timestamp}`;

        const appwrite = new AppwriteService(c.env);
        await appwrite.createTicket(ticketId, amount);

        return c.json({
            id: ticketId,
            amount: amount,
            status: 'pending'
        });
    } catch (error) {
        console.error('Error creating ticket:', error);
        return c.json({ error: 'Internal Server Error' }, 500);
    }
});

// ---------------------------------------------------------------------------
// POST /api/webhook  — legacy SMS / generic webhook handler
// ---------------------------------------------------------------------------
app.post('/api/webhook', async (c) => {
    try {
        const rawBody = await c.req.text();
        console.log('--- WEBHOOK RECEIVED ---');
        console.log('TIMESTAMP:', new Date().toISOString());
        console.log('RAW BODY:', rawBody);

        let body;
        try {
            body = JSON.parse(rawBody);
        } catch (e) {
            console.log('Could not parse JSON body, assuming raw text or URL encoded.');
            body = { sms: rawBody };
        }

        const secret = c.req.query('secret') || body.secret_key;
        if (secret !== c.env.WEBHOOK_SECRET) {
            console.log('Unauthorized Webhook Attempt');
            return c.json({ error: 'Unauthorized' }, 401);
        }

        // Combine potential fields to search for Ticket ID
        const content = ((body.sms || '') + ' ' + (body.body || '') + ' ' + (body.message || '')).trim() || rawBody;

        console.log('SEARCH CONTENT:', content);

        // Regex to find "TICKET" followed by numbers
        const ticketMatch = content.match(/TICKET(\d+)/);

        const paymentSource = body.body || content;
        const paymentMatch = paymentSource.match(/([a-zA-Z0-9\s\.]+?) (?:has )?paid you ₹(\d+(\.\d{1,2})?)/i);

        let foundId = null;
        let status = 'ignored';
        let updated = false;

        if (ticketMatch && paymentMatch) {
            foundId = ticketMatch[0];
            const senderName = paymentMatch[1].trim();
            const paidAmount = parseFloat(paymentMatch[2]);

            console.log('FOUND TICKET ID:', foundId);
            console.log('SENDER:', senderName);
            console.log('PAID AMOUNT:', paidAmount);

            const appwrite = new AppwriteService(c.env);

            const ticket = await appwrite.getTicketStatus(foundId);

            if (!ticket) {
                console.log(`Ticket ${foundId} NOT FOUND`);
                status = 'ticket_not_found';
            } else if (ticket.status === 'paid') {
                console.log(`Ticket ${foundId} ALREADY PAID`);
                status = 'already_paid';
            } else if (ticket.amount !== paidAmount) {
                console.log(`AMOUNT MISMATCH: Ticket requires ${ticket.amount}, but received ${paidAmount}`);
                status = 'amount_mismatch';
            } else {
                updated = await appwrite.markAsPaid(foundId, senderName);
                if (updated) {
                    console.log(`Ticket ${foundId} MARKED AS PAID`);
                    status = 'success';
                } else {
                    status = 'update_failed';
                }
            }
        } else {
            console.log('INVALID SMS FORMAT: Missing Ticket ID or Payment Details');
            if (!ticketMatch) console.log(' - Missing Ticket ID');
            if (!paymentMatch) console.log(' - Missing Payment Details (Name/Amount)');
            status = 'invalid_format';
        }

        return c.json({
            status: 'received',
            processed_id: foundId,
            action: status
        });

    } catch (error) {
        console.error('Webhook error:', error);
        return c.json({ error: 'Internal Server Error' }, 500);
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
app.post('/api/email-webhook', async (c) => {
    try {
        const rawBody = await c.req.text();
        console.log('--- EMAIL WEBHOOK RECEIVED ---');
        console.log('TIMESTAMP:', new Date().toISOString());

        let body: Record<string, string>;
        try {
            body = JSON.parse(rawBody);
        } catch {
            return c.json({ error: 'Invalid JSON' }, 400);
        }

        // ── 1. Shared-secret authentication ──────────────────────────────────
        const secret = c.req.query('secret') || body.secret;
        if (!secret || secret !== c.env.EMAIL_SECRET) {
            console.log('Unauthorized email webhook attempt');
            return c.json({ error: 'Unauthorized' }, 401);
        }

        const from = (body.from || '').toLowerCase().trim();
        const subject = (body.subject || '').trim();
        const text = (body.text || body.html || '').replace(/=\r?\n/g, '').replace(/=3D/g, '=');

        console.log('FROM:', from, '| SUBJECT:', subject);

        // ── 2. Sender domain check ───────────────────────────────────────────
        if (!from.endsWith('@slice.bank.in')) {
            console.log('Rejected: sender domain is not slice.bank.in');
            return c.json({ status: 'ignored', reason: 'sender_domain_mismatch' });
        }

        // ── 3. Subject check ─────────────────────────────────────────────────
        if (!subject.toLowerCase().includes('received') || !subject.toLowerCase().includes('via upi')) {
            console.log('Rejected: subject does not match UPI credit pattern');
            return c.json({ status: 'ignored', reason: 'subject_mismatch' });
        }

        // ── 4. Extract amount, RRN, and sender name from email body ──────────
        //
        //  Amount line: "You have received ₹1.02 via UPI in your slice bank account"
        //  The unicode ₹ is sometimes encoded as =E2=82=B9 in quoted-printable,
        //  we normalise it before matching.
        //
        const normalised = text
            .replace(/=E2=82=B9/gi, '₹')  // QP-encoded ₹
            .replace(/&nbsp;/g, ' ');

        // Amount — e.g. ₹1.02
        const amountMatch = normalised.match(/₹\s*(\d+)\.(\d{2})/);
        if (!amountMatch) {
            console.log('Could not extract amount from email body');
            return c.json({ status: 'ignored', reason: 'amount_not_found' });
        }
        const intPart = parseInt(amountMatch[1], 10);   // 1
        const decPart = parseInt(amountMatch[2], 10);   // 02  → 2
        const paidAmount = parseFloat(`${intPart}.${amountMatch[2]}`); // 1.02

        console.log('PAID AMOUNT:', paidAmount, '| DEC PART:', decPart);

        // RRN — appears after "RRN" label in the table
        const rrnMatch = normalised.match(/RRN\D{0,10}(\d{9,15})/i);
        const rrn = rrnMatch ? rrnMatch[1] : null;
        console.log('RRN:', rrn);

        // Sender name — appears after "From" label in the transaction table
        const senderMatch = normalised.match(/From\s*<\/td>\s*<td[^>]*>\s*([A-Z0-9 ]+)\s*</i)
            || normalised.match(/From\s*[\|\t:]\s*([A-Z0-9 ]+)/i);
        const senderName = senderMatch ? senderMatch[1].trim() : 'UNKNOWN';
        console.log('SENDER NAME:', senderName);

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
        const candidates = await appwrite.listRecentPendingTickets(6);

        const now = Date.now();
        const FIVE_MIN_MS = 5 * 60 * 1000;

        let matchedTicket: Awaited<ReturnType<typeof appwrite.listRecentPendingTickets>>[0] | null = null;
        let matchedTicketId: string | null = null;

        for (const ticket of candidates) {
            // ── 6. 5-minute window check ─────────────────────────────────────
            const ticketTime = new Date(ticket.createdAt).getTime();
            if (now - ticketTime > FIVE_MIN_MS) {
                console.log(`Ticket ${ticket.ticketId}: outside 5-minute window, skipping`);
                continue;
            }

            // ── Decimal suffix check ──────────────────────────────────────────
            //  Extract trailing digits of the ticket numeric part
            const numericPart = ticket.ticketId.replace(/^TICKET/i, '');
            const ticketSuffix = parseInt(numericPart.slice(-2), 10); // last 2 digits

            if (ticketSuffix !== decPart) {
                console.log(`Ticket ${ticket.ticketId}: suffix ${ticketSuffix} !== dec ${decPart}, skipping`);
                continue;
            }

            // ── 7. Amount integer check ───────────────────────────────────────
            //  The ticket stores the full expected amount (integer), e.g. 1.
            //  The paid amount must match (floor-comparing integer part).
            if (Math.floor(ticket.amount) !== intPart) {
                console.log(`Ticket ${ticket.ticketId}: integer amount mismatch (expected ${ticket.amount}, got ${intPart}), skipping`);
                continue;
            }

            if (ticket.status === 'paid') {
                console.log(`Ticket ${ticket.ticketId}: already paid, skipping`);
                continue;
            }

            matchedTicket = ticket;
            matchedTicketId = ticket.ticketId;
            break;
        }

        if (!matchedTicket || !matchedTicketId) {
            console.log(`No matching pending ticket found for amount ₹${paidAmount} (dec=${decPart})`);
            return c.json({
                status: 'ignored',
                reason: 'no_matching_ticket',
                paid_amount: paidAmount,
                dec_part: decPart,
            });
        }

        // ── 8. Mark ticket as paid ────────────────────────────────────────────
        const updated = await appwrite.markAsPaid(matchedTicketId, senderName, rrn ?? undefined);

        if (updated) {
            console.log(`Ticket ${matchedTicketId} MARKED AS PAID via email. RRN: ${rrn}`);
            return c.json({
                status: 'success',
                ticket_id: matchedTicketId,
                paid_amount: paidAmount,
                rrn: rrn,
                sender: senderName,
            });
        } else {
            return c.json({ status: 'error', reason: 'update_failed' }, 500);
        }

    } catch (error) {
        console.error('Email webhook error:', error);
        return c.json({ error: 'Internal Server Error' }, 500);
    }
});

// ---------------------------------------------------------------------------
// GET /api/status/:id  — poll ticket status
// ---------------------------------------------------------------------------
app.get('/api/status/:id', async (c) => {
    const id = c.req.param('id');
    const appwrite = new AppwriteService(c.env);
    const status = await appwrite.getTicketStatus(id);

    if (!status) {
        return c.json({ status: 'not_found' }, 404);
    }

    return c.json(status);
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
                text: rawEmail
            };

            const req = new Request("http://localhost/api/email-webhook", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload)
            });

            const res = await app.fetch(req, env, ctx);
            console.log("Email webhook local processing status:", res.status);
            const resultText = await res.text();
            console.log("Email webhook result:", resultText);
        } catch (e) {
            console.error("Email worker error:", e);
        }
    }
};
