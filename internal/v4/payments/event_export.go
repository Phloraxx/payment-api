package payments

import "time"

// BuildEventPayloadAt builds the canonical merchant webhook envelope for an
// already-materialized payment state. Higher-level v4 services use it so
// operator corrections and automatic transitions share one wire contract.
func BuildEventPayloadAt(eventID, eventType string, payment Payment, eventAt time.Time) ([]byte, error) {
	return paymentEventPayloadAt(eventID, eventType, payment, eventAt)
}
