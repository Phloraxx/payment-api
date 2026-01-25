import { Client, Databases, ID, Query } from 'node-appwrite';

// Environment variables should be defined in wrangler.json or set in Cloudflare dashboard
// In Hono + Cloudflare Workers, we access env via Context, but for this helper
// we might need to pass the config or initialize it per request.
// However, since we want to keep it simple, we can export a function that takes the env.

export interface Env {
    APPWRITE_ENDPOINT: string;
    APPWRITE_PROJECT_ID: string;
    APPWRITE_API_KEY: string;
    APPWRITE_DATABASE_ID: string;
    APPWRITE_COLLECTION_ID: string;
    WEBHOOK_SECRET: string;
}

export interface Ticket {
    id: string; // $id in Appwrite
    ticketId: string; // Custom ID we generate (or we can use $id if we want)
    amount: number;
    status: 'pending' | 'paid';
    createdAt: string;
    senderName?: string;
}

export class AppwriteService {
    client: Client;
    databases: Databases;
    env: Env;

    constructor(env: Env) {
        this.env = env;
        this.client = new Client()
            .setEndpoint(env.APPWRITE_ENDPOINT)
            .setProject(env.APPWRITE_PROJECT_ID)
            .setKey(env.APPWRITE_API_KEY);
        this.databases = new Databases(this.client);
    }

    async createTicket(ticketId: string, amount: number) {
        try {
            return await this.databases.createDocument(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                ID.unique(), // Appwrite ID
                {
                    ticketId: ticketId,
                    amount: amount,
                    status: 'pending',

                }
            );
        } catch (error) {
            console.error('Appwrite createTicket error:', error);
            throw error;
        }
    }

    async markAsPaid(ticketId: string, senderName?: string) {
        try {
            // First we need to find the document by our ticketId
            const response = await this.databases.listDocuments(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                [Query.equal('ticketId', ticketId)]
            );

            if (response.documents.length === 0) {
                return false;
            }

            const docId = response.documents[0].$id;

            await this.databases.updateDocument(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                docId,
                {
                    status: 'paid',
                    senderName: senderName
                }
            );
            return true;
        } catch (error) {
            console.error('Appwrite markAsPaid error:', error);
            return false;
        }
    }

    async getTicketStatus(ticketId: string) {
        try {
            const response = await this.databases.listDocuments(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                [Query.equal('ticketId', ticketId)]
            );

            if (response.documents.length === 0) {
                return null;
            }

            const doc = response.documents[0];
            return {
                ticketId: doc.ticketId,
                status: doc.status,
                amount: doc.amount,
                senderName: doc.senderName
            };
        } catch (error) {
            console.error('Appwrite getTicketStatus error:', error);
            return null;
        }
    }
}
