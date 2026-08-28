export type Page = "dashboard" | "payments" | "reviews" | "reconciliation" | "sms" | "email" | "alerts" | "refunds" | "webhooks" | "audit" | "razorpay_test" | "settings";

export type Connector = {
  enabled: boolean;
  state: string;
  paired: boolean;
  connected: boolean;
  phoneResponsive: boolean;
  pairingMethod?: string;
  pairingEmoji?: string;
  accountEmail?: string;
  lastConnectedAt?: string;
  lastMessageAt?: string;
  lastError?: string;
};


export type EvidenceShadowMetrics = {
  windowStart: string;
  windowDays: number;
  androidObserved: number;
  androidParseable: number;
  androidComplete: number;
  libgmObserved: number;
  libgmComplete: number;
  exactMatches: number;
  androidOnlyComplete: number;
  libgmOnlyComplete: number;
  referenceCoveragePercent: number;
  exactParityPercent: number;
  removalReady: boolean;
  removalGate: string;
};

export type RelayStatus = {
  ready: boolean;
  enabledDevices: number;
  activeDevices: number;
  legacyGraceDevices: number;
  powerUnhealthyDevices: number;
  staleAfterSeconds: number;
  lastSeenAt?: string | null;
  lastHeartbeatAt?: string | null;
  lastEventAt?: string | null;
  lastMatchedAt?: string | null;
  recentErrorCount: number;
  pendingQueueCount: number;
  failedQueueCount: number;
};

export type RelayDevice = {
  id: string;
  deviceId: string;
  name: string;
  enabled: boolean;
  appVersion: string;
  androidVersion: string;
  deviceModel: string;
  lastSeenAt?: string | null;
  lastHeartbeatAt?: string | null;
  heartbeatGraceUntil?: string | null;
  notificationAccess: boolean;
  listenerConnected: boolean;
  powerHealthReported: boolean;
  batteryOptimizationExempt: boolean;
  powerSaveMode: boolean;
  backgroundRestricted: boolean;
  foregroundService: boolean;
  powerHealthy: boolean;
  pendingCount: number;
  failedCount: number;
  lastClientError?: string;
  lastDeliveryAt?: string | null;
  lastEventAt?: string | null;
  lastMatchedAt?: string | null;
  lastMatchedPaymentId?: string;
  recentErrorCount: number;
  active: boolean;
};

export type CapacityPool = {
  requestedAmountPaise: number;
  requestedAmount: string;
  pending: number;
  quarantined: number;
  blocked: number;
  available: number;
  utilizationPercent: number;
  level: "normal" | "warning" | "critical";
};

export type CapacitySnapshot = {
  pools: CapacityPool[];
  warningPools: number;
  criticalPools: number;
};

export type BackupStatus = {
  enabled: boolean;
  cron?: string;
  maxKeep: number;
  offsite: boolean;
  backupCount: number;
  latest?: { name: string; size: number; modTime: string };
  latestVerified: boolean;
  verificationError?: string;
  error?: string;
};

export type DashboardData = {
  stats: Record<string, number>;
  connector: Connector;
  relay?: RelayStatus;
  capacity?: CapacitySnapshot;
  openReviewCount?: number;
  openAlertCount?: number;
  backup?: BackupStatus;
};

export type OperatorPaymentSummary = {
  id: string;
  paymentAccount: "kotak" | "slice" | "paytm";
  requestedAmountPaise: number;
  payableAmountPaise: number;
  status: "pending" | "paid" | "expired" | "cancelled" | "late";
  createdAt: string;
  expiresAt: string;
  paidAt?: string;
};

export type OperatorPaymentDetail = OperatorPaymentSummary & {
  externalId?: string;
  payerName?: string;
  upiId?: string;
  rrn?: string;
  evidenceSource?: string;
  evidenceReference?: string;
  resolvedAt?: string;
};

export type OperatorOverviewResponse = {
  overview: {
    paymentCounts: Record<string, number>;
    openReviews: number;
    openAlerts: number;
    recentPayments: OperatorPaymentSummary[];
  };
  connector: Connector;
  relay?: RelayStatus;
  capacity?: CapacitySnapshot;
  backup?: BackupStatus;
};

export type PaymentCreateResponse = {
  id: string;
  paymentAccount: "kotak" | "slice" | "paytm";
  paymentAccountLabel: string;
  verificationMethod: "sms" | "email" | "notification";
  requestedAmount: number;
  requestedAmountPaise: number;
  payableAmount: string;
  payableAmountPaise: number;
  status: string;
  expiresAt: string;
  paidAt: string | null;
  externalId: string;
  paymentFlow: "upi_intent" | "merchant_qr" | "qr_only";
  upiUri?: string;
  qrPayload?: string;
};

export type OperatorEvidenceDetail = {
  kind: "sms" | "email" | "reconciliation";
  id: string;
  source?: string; sender?: string; subject?: string;
  amountPaise?: number; reference?: string; upiId?: string; payerName?: string;
  occurredAt?: string; description?: string; status?: string; notes?: string;
};
export type OperatorReviewSummary = {
  id: string; kind: string; status: "open" | "resolved" | "dismissed"; severity: string;
  paymentId?: string; candidatePaymentIds?: string[]; reason: string; openedAt: string;
};
export type OperatorReviewDetail = OperatorReviewSummary & {
  resolution?: string; resolutionNote?: string; resolvedAt?: string; evidence?: OperatorEvidenceDetail;
};
export type OperatorAlertSummary = {
  id: string; kind: string; status: "open" | "resolved"; severity: string; message: string;
  occurrenceCount: number; firstSeenAt: string; lastSeenAt: string; notificationStatus?: string;
  notificationAttempts?: number; notificationLastError?: string; notificationDeliveredAt?: string;
};

export type OperatorReconciliationRun = {
  id: string; filename: string; status: string; totalRows: number; matchedRows: number; unmatchedRows: number; duplicateRows: number; conflictRows: number; invalidRows: number; error?: string; startedAt: string; completedAt?: string;
};
export type OperatorReconciliationEntry = {
  id: string; rowNumber: number; transactionTime?: string; amountPaise?: number; reference?: string; description?: string; status: string; paymentId?: string; notes?: string;
};
export type OperatorRefund = {
  id: string; paymentId: string; amountPaise: number; status: string; reason?: string; reference?: string; externalId?: string; requestedAt: string; completedAt?: string;
};

export type RazorpayTestConfig = {
  enabled: boolean;
  keyId: string;
  displayName: string;
  mode: "test";
};

export type OperatorRazorpayOrder = {
  id: string; amountPaise: number; currency: string; status: string; externalId?: string; razorpayOrderId?: string; razorpayPaymentId?: string; providerStatus?: string; paymentMethod?: string; amountRefunded?: number; error?: string; createdAt: string; capturedAt?: string;
};

export type RazorpayTestOrderResponse = {
  id: string;
  amountPaise: number;
  currency: string;
  status: string;
  externalId: string;
  razorpayOrderId: string;
  razorpayPaymentId: string;
  providerStatus: string;
  paymentMethod: string;
  amountRefunded: number;
  error: string;
  createdAt: string;
  capturedAt: string;
  keyId: string;
  displayName: string;
};
