import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { AppwriteService, Env } from './lib/appwrite';

const app = new Hono<{ Bindings: Env }>();

app.use('/*', cors());

app.get('/', (c) => {
    return c.text('Payment Gateway API is running!');
});

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

        // Regex to find "{Sender Name} paid you ₹{Amount}" or "{Sender Name} has paid you ₹{Amount}"
        // Prefer extracting from 'body' field if available, as it likely contains just the message.
        // We use a non-greedy capture for the name if possible, or rely on the field boundary.
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

            // First get the ticket to verify amount
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

app.get('/api/status/:id', async (c) => {
    const id = c.req.param('id');
    const appwrite = new AppwriteService(c.env);
    const status = await appwrite.getTicketStatus(id);

    if (!status) {
        return c.json({ status: 'not_found' }, 404);
    }

    return c.json(status);
});

export default app;
