package gmessages

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/events"
)

const (
	googleReauthRequiredMessage = "Google account authentication expired; refresh the Google login to reconnect"
	googleAuthFailedMessage     = "google account authentication failed; refresh the browser cookies and try again"
	googleAuthVerifyMessage     = "google account authentication could not be verified; try again"
	googleWrongAccountMessage   = "google login belongs to a different account; use the account already paired with this phone"
)

func isGoogleAuthError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, events.ErrInvalidCredentials) {
		return true
	}
	var httpErr events.HTTPError
	if !errors.As(err, &httpErr) || httpErr.Resp == nil {
		return false
	}
	return httpErr.Resp.StatusCode == http.StatusUnauthorized || httpErr.Resp.StatusCode == http.StatusForbidden
}

func (m *Manager) connectionEventsSuppressed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.State == "reauth_required" || m.status.State == "reauthenticating"
}

func (m *Manager) googleAccountPairingExists() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return validPairingSession(m.session) && m.session.IsGoogleAccount()
}

func (m *Manager) handleGoogleAuthFailure(err error) bool {
	m.mu.RLock()
	googleSession := validPairingSession(m.session) && m.session.IsGoogleAccount()
	m.mu.RUnlock()
	if !googleSession || !isGoogleAuthError(err) {
		return false
	}
	m.logger.Warn().Err(err).Msg("Google Messages account authentication requires refresh")
	m.markGoogleReauthRequired()
	return true
}

func (m *Manager) markGoogleReauthRequired() {
	m.mu.Lock()
	client := m.client
	m.status.State = "reauth_required"
	m.status.Paired = validPairingSession(m.session)
	m.status.Connected = false
	m.status.PairingMethod = sessionPairingMethod(m.session)
	m.status.AccountEmail = sessionAccountEmail(m.session)
	m.status.LastError = googleReauthRequiredMessage
	m.mu.Unlock()
	if client != nil {
		// Keep the client pointer until ReauthenticateGoogle retires its event
		// gate. Status() already forces connected=false in reauth_required.
		client.Disconnect()
	}
}

func cloneAuthData(session *libgm.AuthData) (*libgm.AuthData, error) {
	if session == nil {
		return nil, errors.New("cannot clone empty Google Messages session")
	}
	session.CookiesLock.RLock()
	data, err := json.Marshal(session)
	session.CookiesLock.RUnlock()
	if err != nil {
		return nil, err
	}
	var cloned libgm.AuthData
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}

func googleConfigAccount(client *libgm.Client) string {
	if client == nil || client.Config == nil {
		return ""
	}
	return strings.TrimSpace(client.Config.GetDeviceInfo().GetEmail())
}

// ReauthenticateGoogle replaces only the Google browser authentication on an
// existing Gaia pairing. The phone pairing, crypto keys and relay identity are
// preserved. Fresh cookies are committed only after Google confirms that they
// belong to the same account that originally paired the phone.
func (m *Manager) ReauthenticateGoogle(cookieInput string) error {
	if !m.cfg.GMessagesEnabled {
		return errors.New("google messages connector is disabled")
	}
	cookies, err := parseGoogleCookieInput(cookieInput)
	if err != nil {
		return err
	}

	m.mu.Lock()
	original := m.session
	if !validPairingSession(original) || !original.IsGoogleAccount() {
		m.mu.Unlock()
		return errors.New("google account pairing is not available to reauthenticate")
	}
	if m.status.State == "pairing" || m.status.State == "reauthenticating" {
		m.mu.Unlock()
		return errors.New("google messages authentication is already being changed")
	}
	expectedAccount := sessionAccountEmail(original)
	if expectedAccount == "" {
		m.mu.Unlock()
		return errors.New("paired Google account identity is missing; unpair and pair again")
	}
	oldClient := m.client
	m.client = nil
	m.status.State = "reauthenticating"
	m.status.Paired = true
	m.status.Connected = false
	m.status.LastError = ""
	baseCtx := m.ctx
	m.mu.Unlock()

	// Drain any handler that already started on the retired client before the
	// replacement can be installed. This prevents a delayed old ClientReady or
	// auth failure from mutating the refreshed connection's state.
	m.retireClient(oldClient)

	candidate, err := cloneAuthData(original)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to prepare Google Messages session for reauthentication")
		m.finishGoogleReauthFailure(original, "Google login refresh could not be prepared; try again")
		return errors.New("google login refresh could not be prepared; try again")
	}
	candidate.SetCookies(cookies)
	client := m.newClient(candidate)

	if baseCtx == nil {
		baseCtx = context.Background()
	}
	probeCtx, cancel := context.WithTimeout(baseCtx, 20*time.Second)
	err = client.FetchConfig(probeCtx)
	cancel()
	if err != nil {
		m.retireClient(client)
		m.logger.Warn().Err(err).Msg("failed to verify refreshed Google Messages cookies")
		message := googleAuthVerifyMessage
		if isGoogleAuthError(err) {
			message = googleAuthFailedMessage
		}
		m.finishGoogleReauthFailure(original, message)
		return errors.New(message)
	}

	account := googleConfigAccount(client)
	if account == "" {
		m.retireClient(client)
		m.finishGoogleReauthFailure(original, googleAuthFailedMessage)
		return errors.New(googleAuthFailedMessage)
	}
	if !strings.EqualFold(account, expectedAccount) {
		m.retireClient(client)
		m.finishGoogleReauthFailure(original, googleWrongAccountMessage)
		return errors.New(googleWrongAccountMessage)
	}

	// The live failure that motivated this flow occurred while libgm's stored
	// Tachyon expiry still claimed the token was valid for many more hours.
	// Clearing only the expiry forces libgm.Connect to use the existing signed
	// refresh key to obtain fresh relay auth, while preserving the phone pairing.
	candidate.TachyonExpiry = time.Time{}

	m.mu.Lock()
	if m.session != original || m.status.State != "reauthenticating" {
		m.mu.Unlock()
		m.retireClient(client)
		return errors.New("google messages pairing changed while authentication was being refreshed; try again")
	}
	if err := saveSession(m.cfg.GMessagesSessionPath, candidate); err != nil {
		m.status.State = "reauth_required"
		m.status.Connected = false
		m.status.LastError = "Refreshed Google authentication could not be persisted"
		m.mu.Unlock()
		m.retireClient(client)
		m.logger.Error().Err(err).Msg("failed to persist reauthenticated Google Messages session")
		return errors.New("refreshed Google authentication could not be persisted")
	}
	m.session = candidate
	m.client = client
	m.status.State = "connecting"
	m.status.Paired = true
	m.status.Connected = false
	m.status.PairingMethod = "google"
	m.status.AccountEmail = expectedAccount
	m.status.LastError = ""
	connectCtx := m.ctx
	m.mu.Unlock()

	if connectCtx != nil {
		go m.connectReauthenticatedClient(connectCtx, client)
	} else {
		m.mu.Lock()
		if m.client == client && m.status.State == "connecting" {
			m.status.State = "disconnected"
		}
		m.mu.Unlock()
	}
	return nil
}

func (m *Manager) connectReauthenticatedClient(ctx context.Context, client *libgm.Client) {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		m.mu.RLock()
		current := m.client == client && m.status.State != "reauth_required" && m.status.State != "reauthenticating"
		m.mu.RUnlock()
		if !current {
			return
		}

		err := client.Connect()
		if err == nil {
			// A nil return only starts libgm's asynchronous listener. ClientReady
			// or ListenRecovered will move the manager to connected.
			return
		}
		if m.handleGoogleAuthFailure(err) {
			return
		}
		m.setError("degraded", err)

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

func (m *Manager) finishGoogleReauthFailure(original *libgm.AuthData, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != original {
		return
	}
	m.status.State = "reauth_required"
	m.status.Paired = validPairingSession(original)
	m.status.Connected = false
	m.status.PairingMethod = "google"
	m.status.AccountEmail = sessionAccountEmail(original)
	m.status.LastError = message
}
