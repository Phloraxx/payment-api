package razorpaytest

import (
	"github.com/Phloraxx/payment-api/internal/razorpaycore"
	"github.com/pocketbase/pocketbase/core"
)

type ProviderClient = razorpaycore.ProviderClient
type Service = razorpaycore.Service
type CreateInput = razorpaycore.CreateInput
type VerifyInput = razorpaycore.VerifyInput
type WebhookResult = razorpaycore.WebhookResult

func NewService(app core.App, client ProviderClient, keyID, keySecret, webhookSecret, displayName string) *Service {
	return razorpaycore.NewService(app, client, keyID, keySecret, webhookSecret, displayName, razorpaycore.TestMode())
}

var Sign = razorpaycore.Sign
var OrderResponse = razorpaycore.OrderResponse
