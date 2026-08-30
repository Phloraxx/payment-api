package razorpaylive

import "github.com/Phloraxx/payment-api/internal/razorpaycore"

type Client = razorpaycore.Client
type ProviderOrder = razorpaycore.ProviderOrder
type ProviderPayment = razorpaycore.ProviderPayment

func NewClient(keyID, keySecret string) *Client {
	return razorpaycore.NewClient(keyID, keySecret, "paygate_razorpay_live")
}
