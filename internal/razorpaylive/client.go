package razorpaylive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const productionAPIBaseURL = "https://api.razorpay.com/v1"
const maxProviderResponseBytes = 1 << 20

type Client struct {
	KeyID     string
	KeySecret string
	HTTP      *http.Client
	baseURL   string
}

type ProviderOrder struct {
	ID       string `json:"id"`
	Entity   string `json:"entity"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
}

type ProviderPayment struct {
	ID               string `json:"id"`
	Entity           string `json:"entity"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	OrderID          string `json:"order_id"`
	Method           string `json:"method"`
	AmountRefunded   int64  `json:"amount_refunded"`
	Captured         bool   `json:"captured"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
}

func NewClient(keyID, keySecret string) *Client {
	return &Client{
		KeyID: keyID, KeySecret: keySecret,
		HTTP: &http.Client{
			Timeout: 12 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: productionAPIBaseURL,
	}
}

func (c *Client) CreateOrder(ctx context.Context, amountPaise int64, receipt string) (ProviderOrder, error) {
	payload := map[string]any{
		"amount": amountPaise, "currency": "INR", "receipt": receipt,
		"notes": map[string]string{"source": "paygate_razorpay_live"},
	}
	var order ProviderOrder
	if err := c.doJSON(ctx, http.MethodPost, "/orders", payload, &order); err != nil {
		return ProviderOrder{}, err
	}
	if !strings.HasPrefix(order.ID, "order_") || order.Amount != amountPaise || !strings.EqualFold(order.Currency, "INR") {
		return ProviderOrder{}, errors.New("razorpay returned an inconsistent order")
	}
	return order, nil
}

func (c *Client) FetchPayment(ctx context.Context, paymentID string) (ProviderPayment, error) {
	if !strings.HasPrefix(paymentID, "pay_") {
		return ProviderPayment{}, errors.New("invalid razorpay payment id")
	}
	var payment ProviderPayment
	if err := c.doJSON(ctx, http.MethodGet, "/payments/"+paymentID, nil, &payment); err != nil {
		return ProviderPayment{}, err
	}
	if payment.ID != paymentID {
		return ProviderPayment{}, errors.New("razorpay returned an inconsistent payment id")
	}
	return payment, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, requestBody any, responseBody any) error {
	var body io.Reader
	if requestBody != nil {
		raw, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiBaseURL()+path, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.KeyID, c.KeySecret)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("razorpay request failed: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxProviderResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read razorpay response: %w", err)
	}
	if len(raw) > maxProviderResponseBytes {
		return errors.New("razorpay response exceeded 1 MiB")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("razorpay returned HTTP %d: %s", res.StatusCode, providerErrorMessage(raw))
	}
	if err := json.Unmarshal(raw, responseBody); err != nil {
		return fmt.Errorf("decode razorpay response: %w", err)
	}
	return nil
}

func (c *Client) apiBaseURL() string {
	if c.baseURL != "" {
		return strings.TrimRight(c.baseURL, "/")
	}
	return productionAPIBaseURL
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return NewClient(c.KeyID, c.KeySecret).HTTP
}

func providerErrorMessage(raw []byte) string {
	var envelope struct {
		Error struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Error.Description != "" {
		return envelope.Error.Description
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 512 {
		text = text[:512]
	}
	if text == "" {
		return "empty error response"
	}
	return text
}
