import type Database from "better-sqlite3";
import { randomUUID } from "node:crypto";
import { safeEqual } from "../crypto.js";
import {
  generateAuthenticationOptions,
  generateRegistrationOptions,
  verifyAuthenticationResponse,
  verifyRegistrationResponse,
} from "@simplewebauthn/server";
import type { AuthenticationResponseJSON, RegistrationResponseJSON } from "@simplewebauthn/server";
import type { Config } from "../config.js";
import { AppError } from "../errors.js";

interface Challenge {
  challenge: string;
  expiresAt: number;
  verifiedCode?: boolean;
}

interface AuthenticatorRow {
  id: string;
  public_key: string;
  counter: number;
  transports: string | null;
}

const ADMIN_USER_ID = "admin";
const SESSION_TTL_MS = 8 * 60 * 60 * 1000;
const CHALLENGE_TTL_MS = 5 * 60 * 1000;

export class AuthService {
  private setupCodeVerifiedUntil = 0;
  private registrationChallenge: Challenge | undefined;
  private authChallenges = new Map<string, Challenge>();
  private sessions = new Map<string, number>();

  constructor(
    private readonly db: Database.Database,
    private readonly config: Config,
  ) {}

  setupStatus(): { needs_setup: boolean; has_one_time_code: boolean } {
    return {
      needs_setup: this.authenticatorCount() === 0,
      has_one_time_code: Boolean(this.db.prepare("SELECT code FROM one_time_codes WHERE used = 0 LIMIT 1").get()),
    };
  }

  verifyCode(code: string): void {
    const row = this.db.prepare("SELECT code FROM one_time_codes WHERE used = 0 LIMIT 1").get() as { code: string } | undefined;
    if (!row || !safeEqual(row.code, code)) {
      throw new AppError("ADMIN_UNAUTHORIZED", "Invalid setup code.");
    }
    this.setupCodeVerifiedUntil = Date.now() + CHALLENGE_TTL_MS;
  }

  async beginRegistration(): Promise<unknown> {
    if (this.authenticatorCount() > 0) throw new AppError("ADMIN_UNAUTHORIZED", "Setup is already complete.");
    if (Date.now() > this.setupCodeVerifiedUntil) throw new AppError("ADMIN_UNAUTHORIZED", "Setup code verification expired.");
    const options = await generateRegistrationOptions({
      rpName: "Payment Gateway Admin",
      rpID: this.config.rpId,
      userID: new TextEncoder().encode(ADMIN_USER_ID),
      userName: "admin",
      userDisplayName: "Admin",
      attestationType: "none",
      authenticatorSelection: {
        residentKey: "preferred",
        userVerification: "preferred",
      },
    });
    this.registrationChallenge = { challenge: options.challenge, expiresAt: Date.now() + CHALLENGE_TTL_MS, verifiedCode: true };
    return options;
  }

  async completeRegistration(credential: RegistrationResponseJSON): Promise<string> {
    const challenge = this.registrationChallenge;
    if (!challenge || challenge.expiresAt < Date.now()) throw new AppError("ADMIN_UNAUTHORIZED", "Registration challenge expired.");
    const verification = await verifyRegistrationResponse({
      response: credential,
      expectedChallenge: challenge.challenge,
      expectedOrigin: this.config.publicBaseUrl,
      expectedRPID: this.config.rpId,
      requireUserVerification: false,
    });
    if (!verification.verified || !verification.registrationInfo) {
      throw new AppError("ADMIN_UNAUTHORIZED", "Passkey registration failed.");
    }
    const info = verification.registrationInfo;
    this.db
      .prepare("INSERT INTO authenticators (id, public_key, counter, transports) VALUES (?, ?, ?, ?)")
      .run(info.credential.id, Buffer.from(info.credential.publicKey).toString("base64url"), info.credential.counter, JSON.stringify(info.credential.transports ?? []));
    this.db.prepare("UPDATE one_time_codes SET used = 1 WHERE used = 0").run();
    this.registrationChallenge = undefined;
    return this.createSession();
  }

  async beginLogin(): Promise<{ requestId: string; options: unknown }> {
    const authenticators = this.authenticators();
    if (authenticators.length === 0) throw new AppError("ADMIN_UNAUTHORIZED", "No passkey is registered.");
    const requestId = randomUUID();
    const options = await generateAuthenticationOptions({
      rpID: this.config.rpId,
      allowCredentials: authenticators.map((authenticator) => ({
        id: authenticator.id,
        transports: JSON.parse(authenticator.transports ?? "[]"),
      })),
      userVerification: "preferred",
    });
    this.authChallenges.set(requestId, { challenge: options.challenge, expiresAt: Date.now() + CHALLENGE_TTL_MS });
    return { requestId, options };
  }

  async completeLogin(requestId: string, assertion: AuthenticationResponseJSON): Promise<string> {
    const challenge = this.authChallenges.get(requestId);
    if (!challenge || challenge.expiresAt < Date.now()) throw new AppError("ADMIN_UNAUTHORIZED", "Login challenge expired.");
    const authenticator = this.authenticators().find((item) => item.id === assertion.id);
    if (!authenticator) throw new AppError("ADMIN_UNAUTHORIZED", "Unknown passkey.");
    const verification = await verifyAuthenticationResponse({
      response: assertion,
      expectedChallenge: challenge.challenge,
      expectedOrigin: this.config.publicBaseUrl,
      expectedRPID: this.config.rpId,
      credential: {
        id: authenticator.id,
        publicKey: Buffer.from(authenticator.public_key, "base64url"),
        counter: authenticator.counter,
        transports: JSON.parse(authenticator.transports ?? "[]"),
      },
      requireUserVerification: false,
    });
    if (!verification.verified) throw new AppError("ADMIN_UNAUTHORIZED", "Passkey login failed.");
    this.db.prepare("UPDATE authenticators SET counter = ?, updated_at = datetime('now') WHERE id = ?").run(verification.authenticationInfo.newCounter, authenticator.id);
    this.authChallenges.delete(requestId);
    return this.createSession();
  }

  createSession(): string {
    const token = randomUUID();
    this.sessions.set(token, Date.now() + SESSION_TTL_MS);
    return token;
  }

  verifySession(token: string | undefined): boolean {
    if (!token) return false;
    const expiresAt = this.sessions.get(token);
    if (!expiresAt || expiresAt < Date.now()) {
      this.sessions.delete(token);
      return false;
    }
    return true;
  }

  revokeSession(token: string | undefined): void {
    if (token) this.sessions.delete(token);
  }

  private authenticatorCount(): number {
    return (this.db.prepare("SELECT COUNT(*) AS count FROM authenticators").get() as { count: number }).count;
  }

  private authenticators(): AuthenticatorRow[] {
    return this.db.prepare("SELECT * FROM authenticators ORDER BY created_at ASC").all() as AuthenticatorRow[];
  }
}

