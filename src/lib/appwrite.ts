import {
  Client,
  Databases,
  ID,
  Query,
  Models,
} from "node-appwrite";

export interface Env {
  APPWRITE_ENDPOINT: string;
  APPWRITE_PROJECT_ID: string;
  APPWRITE_API_KEY: string;
  APPWRITE_DATABASE_ID: string;
  APPWRITE_COLLECTION_ID: string;
  WEBHOOK_SECRET: string;
  EMAIL_SECRET: string; // shared secret for the Cloudflare email worker webhook
}

export interface Ticket {
  id: string; // $id in Appwrite (THE SECURE INTERNAL ID)
  ticketId: string; // Custom ID we generate  e.g. TICKET1709123456789
  amount: number;
  status: "pending" | "paid" | "cancelled";
  createdAt: string; // ISO string — set by Appwrite ($createdAt)
  senderName?: string;
  rrn?: string; // Reference Retrieval Number from UPI credit mail
  paidAt?: string; // ISO timestamp when email-based payment was confirmed
}

export interface TicketDocument extends Models.Document {
  ticketId: string;
  amount: number;
  status: "pending" | "paid" | "cancelled";
  senderName?: string;
  rrn?: string;
  paidAt?: string;
}

export function toCents(amount: number): number {
  return Math.round(amount * 100);
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
        ticketId,
        {
          ticketId: ticketId,
          amount: amount,
          status: "pending",
        },
      );
    } catch (error) {
      console.error("Appwrite createTicket error:", error);
      throw error;
    }
  }

  /**
   * Mark a ticket as paid, optionally persisting the sender name, RRN,
   * and the timestamp at which payment was confirmed.
   */
  async markAsPaid(ticketId: string, senderName?: string, rrn?: string) {
    try {
      const updatePayload: Record<string, unknown> = {
        status: "paid",
        senderName: senderName,
        paidAt: new Date().toISOString(),
      };

      if (rrn) {
        updatePayload.rrn = rrn;
      }

      console.log("Sending update to Appwrite for ticket:", ticketId);
      console.log("Update Payload:", JSON.stringify(updatePayload, null, 2));

      // We explicitly mapped ticketId to the Document $id
      const updatedDoc = await this.databases.updateDocument<TicketDocument>(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ticketId,
        updatePayload,
      );

      return {
        id: updatedDoc.$id,
        ticketId: updatedDoc.ticketId,
        status: updatedDoc.status,
        amount: updatedDoc.amount,
        senderName: updatedDoc.senderName,
        rrn: updatedDoc.rrn ?? null,
        paidAt: updatedDoc.paidAt ?? null,
        createdAt: updatedDoc.$createdAt,
      };
    } catch (error) {
      console.error("Appwrite markAsPaid error:", error);
      return null;
    }
  }

  async getTicketStatus(ticketId: string) {
    try {
      const response = await this.databases.listDocuments<TicketDocument>(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [Query.equal("ticketId", ticketId)],
      );

      if (response.documents.length === 0) {
        return null;
      }

      const doc = response.documents[0];

      // --- Lazy Cancellation Check ---
      // If the ticket is pending and older than 5 minutes, cancel it immediately.
      let currentStatus = doc.status;
      if (currentStatus === "pending") {
        const ticketTime = new Date(doc.$createdAt).getTime();
        const FIVE_MIN_MS = 5 * 60 * 1000;
        if (Date.now() - ticketTime > FIVE_MIN_MS) {
          currentStatus = "cancelled";
          // Fire-and-forget update to clean the database
          this.databases
            .updateDocument(
              this.env.APPWRITE_DATABASE_ID,
              this.env.APPWRITE_COLLECTION_ID,
              doc.$id,
              { status: "cancelled" },
            )
            .catch((err) => console.error("Lazy cancel DB update error:", err));
        }
      }

      return {
        id: doc.$id,
        ticketId: doc.ticketId,
        status: currentStatus,
        amount: doc.amount,
        senderName: doc.senderName,
        rrn: doc.rrn ?? null,
        paidAt: doc.paidAt ?? null,
        createdAt: doc.$createdAt, // Appwrite built-in timestamp
      };
    } catch (error) {
      console.error("Appwrite getTicketStatus error:", error);
      return null;
    }
  }

  async listRecentPendingTickets(limit: number = 20) {
    try {
      const response = await this.databases.listDocuments<TicketDocument>(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [
          Query.equal("status", "pending"),
          Query.orderDesc("$createdAt"),
          Query.limit(limit),
        ],
      );

      return response.documents.map((doc) => ({
        id: doc.$id,
        ticketId: doc.ticketId,
        status: doc.status,
        amount: doc.amount,
        createdAt: doc.$createdAt,
      }));
    } catch (error) {
      console.error("Appwrite listRecentPendingTickets error:", error);
      return [];
    }
  }

  async getPendingDecimalsForAmount(baseAmount: number): Promise<number[]> {
    try {
      // Fetch recent pending tickets (assume 100 is enough for a 5-min window)
      const candidates = await this.listRecentPendingTickets(100);
      const now = Date.now();
      const FIVE_MIN_MS = 5 * 60 * 1000;
      const decimalsAllocated: number[] = [];
      let cancellationsLeft = 5; // Limit concurrent background cancellations

      for (const ticket of candidates) {
        // Check if within 5 minute window
        const ticketTime = new Date(ticket.createdAt).getTime();
        if (now - ticketTime > FIVE_MIN_MS) {
          // --- Lazy Cancellation Check ---
          // Limit background updates to avoid Cloudflare "Too many subrequests" error
          if (cancellationsLeft > 0) {
            cancellationsLeft--;

            if (ticket.ticketId.startsWith("lock_")) {
              this.databases
                .deleteDocument(
                  this.env.APPWRITE_DATABASE_ID,
                  this.env.APPWRITE_COLLECTION_ID,
                  ticket.id,
                )
                .catch((err) =>
                  console.log(
                    "Lazy cancel batch error (ignorable):",
                    err.message,
                  ),
                );
            } else {
              this.databases
                .updateDocument(
                  this.env.APPWRITE_DATABASE_ID,
                  this.env.APPWRITE_COLLECTION_ID,
                  ticket.id,
                  { status: "cancelled" },
                )
                .catch((err) =>
                  console.log(
                    "Lazy cancel batch error (ignorable):",
                    err.message,
                  ),
                );
            }
          }
          continue;
        }

        // Check if it matches the same base integer amount
        if (Math.floor(ticket.amount) === baseAmount) {
          const decPart = Math.round((ticket.amount - baseAmount) * 100);
          decimalsAllocated.push(decPart);
        }
      }

      return decimalsAllocated;
    } catch (error) {
      console.error("Appwrite getPendingDecimalsForAmount error:", error);
      return [];
    }
  }

  /**
   * Attempts to acquire an absolute, atomic database-level lock for a specific decimal.
   * Uses Appwrite's Document ID uniqueness constraint to guarantee no two concurrent requests
   * can claim the exact same decimal simultaneously.
   */
  async claimDatabaseLock(
    baseAmount: number,
    decimal: number,
  ): Promise<boolean> {
    const lockDocId = `lock_${baseAmount}_${decimal.toString().padStart(2, "0")}`;
    const finalAmount = baseAmount + decimal / 100;

    try {
      await this.databases.createDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        lockDocId,
        {
          ticketId: lockDocId,
          amount: finalAmount,
          status: "pending",
        },
      );
      return true;
    } catch (error: any) {
      if (error.code === 409) {
        // Lock already exists. Check if it is a stale lock (> 5 mins old)
        try {
          const existingLock = await this.databases.getDocument(
            this.env.APPWRITE_DATABASE_ID,
            this.env.APPWRITE_COLLECTION_ID,
            lockDocId,
          );
          const lockTime = new Date(existingLock.$createdAt).getTime();

          if (Date.now() - lockTime > 5 * 60 * 1000) {
            try {
              await this.releaseDatabaseLock(baseAmount, decimal);
            } catch (e) { } // best effort release

            // Retry lock acquisition
            return await this.claimDatabaseLock(baseAmount, decimal);
          }
        } catch (e) {
          // Ignored
        }
        return false; // Someone else has the lock
      }
      throw error; // Not a conflict error, abort
    }
  }

  /**
   * Releases an atomic database-level lock so the decimal can be reused immediately.
   */
  async releaseDatabaseLock(
    baseAmount: number,
    decimal: number,
  ): Promise<void> {
    const lockDocId = `lock_${baseAmount}_${decimal.toString().padStart(2, "0")}`;
    try {
      await this.databases.deleteDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        lockDocId,
      );
    } catch (error) {
      // Safely ignore if the lock doesn't exist
    }
  }
}
