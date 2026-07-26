package gmessages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/sms"
	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/events"
	"rsc.io/qr"
)

type Status struct {
	Enabled         bool       `json:"enabled"`
	State           string     `json:"state"`
	Paired          bool       `json:"paired"`
	Connected       bool       `json:"connected"`
	PhoneResponsive bool       `json:"phoneResponsive"`
	LastConnectedAt *time.Time `json:"lastConnectedAt,omitempty"`
	LastMessageAt   *time.Time `json:"lastMessageAt,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
}

type IngestFunc func(sms.Input) error

type Manager struct {
	cfg    config.Config
	ingest IngestFunc
	logger zerolog.Logger

	mu           sync.RWMutex
	session      *libgm.AuthData
	client       *libgm.Client
	status       Status
	ctx          context.Context
	cancel       context.CancelFunc
	reconnecting bool
}

func NewManager(cfg config.Config, logger zerolog.Logger, ingest IngestFunc) *Manager {
	m := &Manager{cfg: cfg, ingest: ingest, logger: logger}
	m.status = Status{Enabled: cfg.GMessagesEnabled, State: "disabled"}
	if !cfg.GMessagesEnabled {
		return m
	}

	session, err := loadSession(cfg.GMessagesSessionPath)
	if err != nil {
		m.status = Status{Enabled: true, State: "unpaired", LastError: "stored session could not be loaded"}
		logger.Warn().Err(err).Msg("ignoring invalid Google Messages session")
		return m
	}
	if !validSession(session) {
		m.status = Status{Enabled: true, State: "unpaired"}
		return m
	}
	m.session = session
	m.status = Status{Enabled: true, State: "disconnected", Paired: true}
	return m
}

func (m *Manager) Start(parent context.Context) {
	if !m.cfg.GMessagesEnabled {
		return
	}
	m.mu.Lock()
	if m.ctx != nil {
		m.mu.Unlock()
		return
	}
	m.ctx, m.cancel = context.WithCancel(parent)
	ctx := m.ctx
	paired := validSession(m.session)
	m.mu.Unlock()
	if paired {
		go m.connectWithBackoff(ctx)
	}
}

func (m *Manager) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	client := m.client
	m.cancel = nil
	m.ctx = nil
	m.client = nil
	m.status.Connected = false
	if m.status.State == "connected" {
		m.status.State = "disconnected"
	}
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Disconnect()
	}
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := m.status
	if m.client != nil && status.Paired {
		status.Connected = m.client.IsConnected()
		if status.Connected && status.State == "disconnected" {
			status.State = "connected"
		}
	}
	return status
}

func (m *Manager) connectWithBackoff(ctx context.Context) {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}

		m.mu.Lock()
		if !validSession(m.session) {
			m.status.State = "unpaired"
			m.status.Paired = false
			m.mu.Unlock()
			return
		}
		if m.client == nil {
			m.client = m.newClient(m.session)
		}
		client := m.client
		m.status.State = "connecting"
		m.mu.Unlock()

		if err := client.Connect(); err == nil {
			m.markConnected()
			return
		} else {
			m.setError("degraded", err)
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if backoff < time.Minute {
			backoff *= 2
			if backoff > time.Minute {
				backoff = time.Minute
			}
		}
	}
}

func (m *Manager) newClient(session *libgm.AuthData) *libgm.Client {
	client := libgm.NewClient(session, nil, m.logger.With().Str("component", "libgm").Logger())
	client.SetEventHandler(m.handleEvent)
	return client
}

func (m *Manager) handleEvent(raw any) {
	switch event := raw.(type) {
	case *events.ClientReady:
		m.markConnected()
	case *events.PairSuccessful:
		m.mu.Lock()
		m.status.Paired = true
		m.status.State = "connected"
		m.status.Connected = true
		m.status.PhoneResponsive = true
		m.mu.Unlock()
		if err := m.saveCurrentSession(); err != nil {
			m.logger.Error().Err(err).Msg("failed to persist Google Messages session")
		}
	case *events.AuthTokenRefreshed:
		if err := m.saveCurrentSession(); err != nil {
			m.logger.Error().Err(err).Msg("failed to persist refreshed Google Messages auth")
		}
	case *events.PhoneNotResponding:
		m.mu.Lock()
		m.status.PhoneResponsive = false
		m.status.State = "degraded"
		m.mu.Unlock()
	case *events.PhoneRespondingAgain:
		m.mu.Lock()
		m.status.PhoneResponsive = true
		m.status.State = "connected"
		m.status.Connected = true
		m.mu.Unlock()
	case *events.ListenTemporaryError:
		m.setError("degraded", event.Error)
	case *events.ListenRecovered:
		m.markConnected()
	case *events.ListenFatalError:
		m.setError("degraded", event.Error)
		m.scheduleReconnect()
	case *events.PingFailed:
		m.setError("degraded", event.Error)
	case *events.GaiaLoggedOut:
		m.markLoggedOut()
	case *libgm.WrappedMessage:
		m.handleMessage(event)
	}
}

func (m *Manager) handleMessage(wrapped *libgm.WrappedMessage) {
	if wrapped == nil || wrapped.Message == nil {
		return
	}
	message := wrapped.Message
	if isFromMe(message.GetMessageStatus().GetStatus().String()) {
		return
	}
	body := messageBody(wrapped)
	if body == "" || !sms.LooksLikeBankCredit(body) {
		// Privacy by default: don't copy unrelated personal messages into PayGate.
		return
	}

	timestampMS := normalizeTimestampMS(message.GetTimestamp())
	messageAt := time.Time{}
	if timestampMS > 0 {
		messageAt = time.UnixMilli(timestampMS).UTC()
	}
	sender := strings.TrimSpace(message.GetParticipantID())
	if participant := message.GetSenderParticipant(); participant != nil {
		if participant.GetFormattedNumber() != "" {
			sender = participant.GetFormattedNumber()
		} else if participant.GetFullName() != "" {
			sender = participant.GetFullName()
		}
	}

	now := time.Now().UTC()
	m.mu.Lock()
	m.status.LastMessageAt = &now
	m.status.PhoneResponsive = true
	m.mu.Unlock()
	if m.ingest == nil {
		return
	}
	if err := m.ingest(sms.Input{
		Source:        "gmessages",
		SourceEventID: message.GetMessageID(),
		Sender:        sender,
		Body:          body,
		MessageTime:   messageAt,
		RawPayload: map[string]any{
			"messageId":      message.GetMessageID(),
			"conversationId": message.GetConversationID(),
			"isOld":          wrapped.IsOld,
		},
	}); err != nil {
		m.logger.Error().Err(err).Str("message_id", message.GetMessageID()).Msg("failed to ingest Google Messages bank SMS")
	}
}

func messageBody(wrapped *libgm.WrappedMessage) string {
	var parts []string
	for _, info := range wrapped.Message.GetMessageInfo() {
		if content := info.GetMessageContent(); content != nil && strings.TrimSpace(content.GetContent()) != "" {
			parts = append(parts, content.GetContent())
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func isFromMe(status string) bool {
	return strings.Contains(status, "OUTGOING") || strings.Contains(status, "SENT_BY_ME")
}

func normalizeTimestampMS(timestamp int64) int64 {
	if timestamp > 100_000_000_000_000 {
		return timestamp / 1000
	}
	return timestamp
}

func (m *Manager) scheduleReconnect() {
	m.mu.Lock()
	if m.reconnecting || m.ctx == nil || !validSession(m.session) {
		m.mu.Unlock()
		return
	}
	m.reconnecting = true
	ctx := m.ctx
	client := m.client
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			m.reconnecting = false
			m.mu.Unlock()
		}()
		if client != nil {
			client.Disconnect()
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		m.mu.Lock()
		m.client = nil
		m.mu.Unlock()
		m.connectWithBackoff(ctx)
	}()
}

func (m *Manager) Reconnect() error {
	if !m.cfg.GMessagesEnabled {
		return errors.New("google messages connector is disabled")
	}
	m.mu.RLock()
	client := m.client
	paired := validSession(m.session)
	m.mu.RUnlock()
	if !paired {
		return errors.New("google messages is not paired")
	}
	if client == nil {
		m.scheduleReconnect()
		return nil
	}
	if err := client.Reconnect(); err != nil {
		m.setError("degraded", err)
		return err
	}
	m.markConnected()
	return m.saveCurrentSession()
}

func (m *Manager) BeginPair() (string, error) {
	if !m.cfg.GMessagesEnabled {
		return "", errors.New("google messages connector is disabled")
	}
	m.mu.RLock()
	alreadyPaired := validSession(m.session)
	m.mu.RUnlock()
	if alreadyPaired {
		return "", errors.New("google messages is already paired; unpair it before starting a new pairing")
	}
	m.mu.Lock()
	oldClient := m.client
	m.session = libgm.NewAuthData()
	m.client = m.newClient(m.session)
	client := m.client
	m.status = Status{Enabled: true, State: "pairing"}
	m.mu.Unlock()
	if oldClient != nil {
		oldClient.Disconnect()
	}
	qrURL, err := client.StartLogin()
	if err != nil {
		m.setError("degraded", err)
		return "", err
	}
	return qrURL, nil
}

func (m *Manager) RefreshPair() (string, error) {
	m.mu.RLock()
	client := m.client
	state := m.status.State
	m.mu.RUnlock()
	if client == nil || state != "pairing" {
		return "", errors.New("pairing has not started")
	}
	return client.RefreshPhoneRelay()
}

func (m *Manager) Unpair() error {
	m.mu.Lock()
	client := m.client
	m.client = nil
	m.session = nil
	m.status = Status{Enabled: m.cfg.GMessagesEnabled, State: "unpaired"}
	m.mu.Unlock()
	if client != nil {
		if _, err := client.UnpairBugle(); err != nil {
			m.logger.Warn().Err(err).Msg("remote Google Messages unpair failed; deleting local session anyway")
		}
		client.Disconnect()
	}
	if err := os.Remove(m.cfg.GMessagesSessionPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *Manager) markConnected() {
	now := time.Now().UTC()
	m.mu.Lock()
	m.status.State = "connected"
	m.status.Paired = true
	m.status.Connected = true
	m.status.PhoneResponsive = true
	m.status.LastConnectedAt = &now
	m.status.LastError = ""
	m.mu.Unlock()
}

func (m *Manager) markLoggedOut() {
	m.mu.Lock()
	client := m.client
	m.client = nil
	m.session = nil
	m.status = Status{Enabled: m.cfg.GMessagesEnabled, State: "unpaired", LastError: "Google Messages session was logged out"}
	m.mu.Unlock()
	if client != nil {
		client.Disconnect()
	}
	_ = os.Remove(m.cfg.GMessagesSessionPath)
}

func (m *Manager) setError(state string, err error) {
	m.mu.Lock()
	m.status.State = state
	m.status.Connected = false
	if err != nil {
		m.status.LastError = err.Error()
	}
	m.mu.Unlock()
}

func (m *Manager) saveCurrentSession() error {
	// Keep the manager read lock for the whole atomic save so Unpair cannot
	// delete the session and then have this goroutine resurrect it afterwards.
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.session == nil {
		return nil
	}
	return saveSession(m.cfg.GMessagesSessionPath, m.session)
}

func validSession(session *libgm.AuthData) bool {
	return session != nil &&
		session.Browser != nil &&
		session.Mobile != nil &&
		session.RequestCrypto != nil &&
		len(session.RequestCrypto.AESKey) == 32 &&
		len(session.RequestCrypto.HMACKey) > 0 &&
		session.RefreshKey != nil &&
		len(session.RefreshKey.D) > 0 &&
		len(session.RefreshKey.X) > 0 &&
		len(session.RefreshKey.Y) > 0 &&
		len(session.TachyonAuthToken) > 0
}

func loadSession(path string) (*libgm.AuthData, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var session libgm.AuthData
	if err := json.NewDecoder(file).Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveSession(path string, session *libgm.AuthData) error {
	if session == nil {
		return errors.New("cannot save empty Google Messages session")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	// AuthData.Cookies is mutated by libgm HTTP handling and has its own lock.
	session.CookiesLock.RLock()
	encodeErr := encoder.Encode(session)
	session.CookiesLock.RUnlock()
	if encodeErr != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return encodeErr
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, 0o600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// PairConsole performs the explicit operator QR flow. QR tokens are refreshed
// every 25 seconds because the Messages Web pairing token is short-lived.
func PairConsole(ctx context.Context, sessionPath, qrPNG string, logger zerolog.Logger) error {
	session := libgm.NewAuthData()
	client := libgm.NewClient(session, nil, logger.With().Str("component", "libgm-pair").Logger())
	paired := make(chan struct{}, 1)
	fatal := make(chan error, 1)
	client.SetEventHandler(func(raw any) {
		switch event := raw.(type) {
		case *events.PairSuccessful:
			if err := saveSession(sessionPath, session); err != nil {
				select {
				case fatal <- err:
				default:
				}
				return
			}
			select {
			case paired <- struct{}{}:
			default:
			}
		case *events.ListenFatalError:
			select {
			case fatal <- event.Error:
			default:
			}
		}
	})
	defer client.Disconnect()

	render := func(qrURL string) error {
		fmt.Fprint(os.Stderr, "\nScan this QR in Google Messages → Device pairing → Switch to QR pairing:\n\n")
		qrterminal.GenerateHalfBlock(qrURL, qrterminal.L, os.Stderr)
		if qrPNG != "" {
			code, err := qr.Encode(qrURL, qr.L)
			if err != nil {
				return err
			}
			if err := os.WriteFile(qrPNG, code.PNG(), 0o600); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "QR PNG updated: %s\n", qrPNG)
		}
		return nil
	}

	qrURL, err := client.StartLogin()
	if err != nil {
		return fmt.Errorf("start Google Messages pairing: %w", err)
	}
	if err := render(qrURL); err != nil {
		return err
	}

	refresh := time.NewTicker(25 * time.Second)
	defer refresh.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-fatal:
			return fmt.Errorf("google messages pairing failed: %w", err)
		case <-paired:
			fmt.Fprintf(os.Stderr, "Paired. Session saved to %s\n", sessionPath)
			return nil
		case <-refresh.C:
			fresh, err := client.RefreshPhoneRelay()
			if err != nil {
				logger.Warn().Err(err).Msg("failed to refresh Google Messages pairing QR")
				continue
			}
			if err := render(fresh); err != nil {
				return err
			}
		}
	}
}
