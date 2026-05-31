import type { Config } from "../config.js";
import type { Ticket } from "../../types/index.js";
import type { LoggerService } from "./logger.service.js";

export class AppwriteService {
  constructor(
    private readonly config: Config,
    private readonly logger: LoggerService,
  ) {}

  syncTicket(ticket: Ticket): void {
    if (!this.config.appwrite.enabled) return;
    void this.upsert(ticket).catch((error: unknown) => {
      this.logger.error("Appwrite sync failure", {
        ticket_id: ticket.id,
        error: error instanceof Error ? error.message : String(error),
      });
    });
  }

  async fullSync(tickets: Ticket[]): Promise<{ attempted: number; failed: number }> {
    if (!this.config.appwrite.enabled) return { attempted: 0, failed: 0 };
    let failed = 0;
    for (const ticket of tickets) {
      try {
        await this.upsert(ticket);
      } catch (error) {
        failed += 1;
        this.logger.error("Appwrite full sync item failure", {
          ticket_id: ticket.id,
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }
    return { attempted: tickets.length, failed };
  }

  async reachable(): Promise<boolean> {
    if (!this.config.appwrite.enabled) return false;
    try {
      const response = await fetch(`${this.config.appwrite.endpoint}/health`, { method: "GET" });
      return response.ok;
    } catch {
      return false;
    }
  }

  private async upsert(ticket: Ticket): Promise<void> {
    const { endpoint, projectId, apiKey, databaseId, collectionId } = this.config.appwrite;
    const url = `${endpoint}/v1/databases/${databaseId}/collections/${collectionId}/documents/${encodeURIComponent(ticket.id)}`;
    const headers = {
      "Content-Type": "application/json",
      "X-Appwrite-Project": projectId ?? "",
      "X-Appwrite-Key": apiKey ?? "",
    };
    const body = JSON.stringify({
      documentId: ticket.id,
      data: ticket,
      permissions: [],
    });
    let response = await fetch(url, { method: "PATCH", headers, body });
    if (response.status === 404) {
      response = await fetch(`${endpoint}/v1/databases/${databaseId}/collections/${collectionId}/documents`, {
        method: "POST",
        headers,
        body,
      });
    }
    if (!response.ok) {
      throw new Error(`Appwrite returned ${response.status}`);
    }
  }
}
