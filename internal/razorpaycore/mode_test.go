package razorpaycore

import "testing"

func TestModesRemainStorageAndIdentityIsolated(t *testing.T) {
	testMode := TestMode()
	liveMode := LiveMode()
	if testMode.OrdersCollection == liveMode.OrdersCollection ||
		testMode.EventsCollection == liveMode.EventsCollection ||
		testMode.EventOrderField == liveMode.EventOrderField ||
		testMode.ReceiptPrefix == liveMode.ReceiptPrefix ||
		testMode.ErrorPrefix == liveMode.ErrorPrefix {
		t.Fatalf("Razorpay test/live mode identities must remain isolated: test=%+v live=%+v", testMode, liveMode)
	}
	if testMode.MinOrderPaise <= 0 || liveMode.MinOrderPaise <= 0 ||
		testMode.MaxOrderPaise < testMode.MinOrderPaise || liveMode.MaxOrderPaise < liveMode.MinOrderPaise {
		t.Fatalf("invalid Razorpay amount policy: test=%+v live=%+v", testMode, liveMode)
	}
}
