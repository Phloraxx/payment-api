export interface ExtractedEmailData {
  paidAmount: number | null;
  intPart: number | null;
  decPart: number | null;
  rrn: string | null;
  senderName: string;
  upiId?: string | null;
}

export class EmailParser {
  /**
   * Extracts the payment amount, decimal parts, RRN, and Sender Name from a Slice UPI email.
   */
  static parseSliceEmail(
    subject: string,
    textOrHtmlBody: string,
  ): ExtractedEmailData {
    const result: ExtractedEmailData = {
      paidAmount: null,
      intPart: null,
      decPart: null,
      rrn: null,
      senderName: "UNKNOWN",
    };

    // Normalise Quoted-Printable encoding and non-breaking spaces
    const normalised = textOrHtmlBody
      .replace(/=\r?\n/g, "")
      .replace(/=3D/g, "=")
      .replace(/=E2=82=B9/gi, "₹") // QP-encoded ₹
      .replace(/&nbsp;/g, " ");

    // --- Extract Amount ---
    // Prefer extracting from Subject to avoid catching account balances in body
    let amountMatch = subject.match(/₹\s*(\d+)(?:\.(\d{2}))?/i);

    if (!amountMatch) {
      // Fallback to body text "received ₹5 via" or the first amount found
      amountMatch =
        normalised.match(/received\s*₹\s*(\d+)(?:\.(\d{2}))?\s*via/i) ||
        normalised.match(/₹\s*(\d+)(?:\.(\d{2}))?/);
    }

    if (amountMatch) {
      result.intPart = parseInt(amountMatch[1], 10);
      result.decPart = amountMatch[2] ? parseInt(amountMatch[2], 10) : 0;
      result.paidAmount = parseFloat(
        `${result.intPart}.${amountMatch[2] || "00"}`,
      );
    }

    // --- Extract RRN ---
    // RRN appears usually after "RRN" label in the table. The table splits "RRN" and the number into two columns.
    const rrnMatch =
      normalised.match(/RRN\s*<\/td>\s*<td[^>]*>\s*(\d{9,15})/i) ||
      normalised.match(/RRN\D{0,10}(\d{9,15})/i);
    if (rrnMatch) {
      result.rrn = rrnMatch[1];
    }

    // --- Extract Sender Name ---
    // Sender name appears after "From" label in the transaction table
    const senderMatch =
      normalised.match(/From\s*<\/td>\s*<td[^>]*>\s*([A-Z0-9 ]+)\s*</i) ||
      normalised.match(/From\s*[\|\t:]\s*([A-Z0-9 ]+)/i);

    if (senderMatch) {
      result.senderName = senderMatch[1].trim();
    }

    return result;
  }

  /**
   * Extracts the payment amount, decimal parts, RRN, and Sender Name from a Kotak SMS.
   * Example: "Confirmed payment for Received Rs.3.00 in your Kotak Bank AC X4959 from drvijayapalliyil@oksbi on 08-03-26.UPI Ref:606703736479."
   */
  static parseKotakSms(smsBody: string): ExtractedEmailData | null {
    if (!smsBody.toUpperCase().includes("KOTAK")) {
      return null;
    }

    const result: ExtractedEmailData = {
      paidAmount: null,
      intPart: null,
      decPart: null,
      rrn: null,
      senderName: "UNKNOWN",
      upiId: null,
    };

    // Amount: search for "Rs.3.00"
    const amountMatch = smsBody.match(/Rs\.?\s*(\d+)(?:\.(\d{2}))?/i);
    if (amountMatch) {
      result.intPart = parseInt(amountMatch[1], 10);
      result.decPart = amountMatch[2] ? parseInt(amountMatch[2], 10) : 0;
      result.paidAmount = parseFloat(`${result.intPart}.${amountMatch[2] || "00"}`);
    } else {
      // If we can't find an amount, it's not a valid payment SMS
      return null;
    }

    // UPI ID: comes after "from " and ends at the space or period.
    // E.g., from drvijayapalliyil@oksbi on 08-03-26
    const upiIdMatch = smsBody.match(/from\s+([a-zA-Z0-9.\-_]+@[a-zA-Z0-9.\-_]+)/i);
    if (upiIdMatch) {
      result.upiId = upiIdMatch[1].trim();
    }

    // RRN: after "UPI Ref:" or similar
    const rrnMatch = smsBody.match(/UPI Ref[:\s]*(\d+)/i) || smsBody.match(/Ref\.?No\.?[:\s]*(\d+)/i) || smsBody.match(/RRN[:\s]*(\d+)/i);
    if (rrnMatch) {
      result.rrn = rrnMatch[1];
    }

    return result;
  }
}
