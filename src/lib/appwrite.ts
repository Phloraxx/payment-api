import { Client, Databases, ID, Query } from 'node-appwrite';

export interface Env {
    APPWRITE_ENDPOINT: string;
    APPWRITE_PROJECT_ID: string;
    APPWRITE_API_KEY: string;
    APPWRITE_DATABASE_ID: string;
    APPWRITE_COLLECTION_ID: string;
    WEBHOOK_SECRET: string;
    EMAIL_SECRET: string;   // shared secret for the Cloudflare email worker webhook
}

export interface Ticket {
    id: string;         // $id in Appwrite
    ticketId: string;   // Custom ID we generate  e.g. TICKET1709123456789
    amount: number;
    status: 'pending' | 'paid';
    createdAt: string;  // ISO string — set by Appwrite ($createdAt)
    senderName?: string;
    rrn?: string;       // Reference Retrieval Number from UPI credit mail
    paidAt?: string;    // ISO timestamp when email-based payment was confirmed
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
                ID.unique(),
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

    /**
     * Mark a ticket as paid, optionally persisting the sender name, RRN,
     * and the timestamp at which payment was confirmed.
     */
    async markAsPaid(ticketId: string, senderName?: string, rrn?: string) {
        try {
            const response = await this.databases.listDocuments(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                [Query.equal('ticketId', ticketId)]
            );

            if (response.documents.length === 0) {
                return false;
            }

            const docId = response.documents[0].$id;

            const updatePayload: Record<string, unknown> = {
                status: 'paid',
                senderName: senderName,
                paidAt: new Date().toISOString(),
            };

            if (rrn) {
                updatePayload.rrn = rrn;
            }

            await this.databases.updateDocument(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                docId,
                updatePayload
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
                senderName: doc.senderName,
                rrn: doc.rrn ?? null,
                paidAt: doc.paidAt ?? null,
                createdAt: doc.$createdAt,   // Appwrite built-in timestamp
            };
        } catch (error) {
            console.error('Appwrite getTicketStatus error:', error);
            return null;
        }
    }

    async listRecentPendingTickets(limit: number = 20) {
        try {
            const response = await this.databases.listDocuments(
                this.env.APPWRITE_DATABASE_ID,
                this.env.APPWRITE_COLLECTION_ID,
                [
                    Query.equal('status', 'pending'),
                    Query.orderDesc('$createdAt'),
                    Query.limit(limit)
                ]
            );

            return response.documents.map(doc => ({
                id: doc.$id,
                ticketId: doc.ticketId,
                status: doc.status,
                amount: doc.amount,
                createdAt: doc.$createdAt
            }));
        } catch (error) {
            console.error('Appwrite listRecentPendingTickets error:', error);
            return [];
        }
    }
}
