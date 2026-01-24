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

        const smsText = body.sms || body.message || rawBody;
        // Regex to find "TICKET" followed by numbers (and optional hyphens if we changed format, but currently it's TICKET\d+)
        const ticketMatch = smsText.match(/TICKET(\d+)/);

        let foundId = null;
        let status = 'ignored';
        let updated = false;

        if (ticketMatch) {
            foundId = ticketMatch[0];
            console.log('FOUND TICKET ID:', foundId);

            const appwrite = new AppwriteService(c.env);
            updated = await appwrite.markAsPaid(foundId);

            if (updated) {
                console.log(`Ticket ${foundId} MARKED AS PAID`);
                status = 'success';
            } else {
                console.log(`Ticket ${foundId} NOT FOUND or already paid`);
                status = 'not_found_or_error';
            }
        } else {
            console.log('NO TICKET ID FOUND IN SMS');
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
