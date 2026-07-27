package gmessages

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/events"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

func googleTestSession(t *testing.T) *libgm.AuthData {
	t.Helper()
	session := libgm.NewAuthData()
	session.Browser = &gmproto.Device{}
	session.Mobile = &gmproto.Device{SourceID: "user@example.com"}
	session.DestRegID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	session.TachyonAuthToken = []byte{1, 2, 3}
	cookies, err := parseGoogleCookieInput(testCookieHeader)
	if err != nil {
		t.Fatal(err)
	}
	session.SetCookies(cookies)
	return session
}

func TestGoogleAuthErrorDetection(t *testing.T) {
	unauthorized := events.HTTPError{
		Action: "polling",
		Resp:   &http.Response{StatusCode: http.StatusUnauthorized},
	}
	for name, err := range map[string]error{
		"direct 401":          unauthorized,
		"wrapped 401":         errors.Join(errors.New("listen failed"), unauthorized),
		"invalid credentials": events.ErrInvalidCredentials,
	} {
		t.Run(name, func(t *testing.T) {
			if !isGoogleAuthError(err) {
				t.Fatalf("%v was not classified as Google auth failure", err)
			}
		})
	}
	if isGoogleAuthError(errors.New("temporary network error")) {
		t.Fatal("unrelated error was classified as Google auth failure")
	}
}

func TestMarkGoogleReauthRequiredPreservesPairing(t *testing.T) {
	session := googleTestSession(t)
	manager := &Manager{
		cfg:     config.Config{GMessagesEnabled: true},
		logger:  zerolog.Nop(),
		session: session,
		status: Status{
			Enabled: true, State: "connected", Paired: true, Connected: true,
			PairingMethod: "google", AccountEmail: "user@example.com",
		},
	}
	manager.markGoogleReauthRequired()
	status := manager.Status()
	if manager.session != session {
		t.Fatal("reauth requirement replaced the existing phone pairing")
	}
	if status.State != "reauth_required" || !status.Paired || status.Connected {
		t.Fatalf("status = %#v; want paired reauth_required and disconnected", status)
	}
	if status.AccountEmail != "user@example.com" || status.LastError != googleReauthRequiredMessage {
		t.Fatalf("unexpected reauth status: %#v", status)
	}
}

func TestCloneAuthDataDoesNotShareCookies(t *testing.T) {
	original := googleTestSession(t)
	cloned, err := cloneAuthData(original)
	if err != nil {
		t.Fatal(err)
	}
	cloned.CookiesLock.Lock()
	cloned.Cookies["SID"] = "changed"
	cloned.CookiesLock.Unlock()
	original.CookiesLock.RLock()
	originalSID := original.Cookies["SID"]
	original.CookiesLock.RUnlock()
	if originalSID == "changed" {
		t.Fatal("cloned session shares the original cookie map")
	}
	if sessionAccountEmail(cloned) != sessionAccountEmail(original) || cloned.PairingID != original.PairingID {
		t.Fatal("cloned session did not preserve the existing pairing identity")
	}
}

func TestGoogleConfigAccount(t *testing.T) {
	client := libgm.NewClient(libgm.NewAuthData(), nil, zerolog.Nop())
	client.Config = &gmproto.Config{DeviceInfo: &gmproto.Config_DeviceInfo{Email: " user@example.com "}}
	if got := googleConfigAccount(client); got != "user@example.com" {
		t.Fatalf("googleConfigAccount() = %q", got)
	}
}

func TestReauthenticateGoogleRequiresExistingGooglePairing(t *testing.T) {
	manager := &Manager{cfg: config.Config{GMessagesEnabled: true}, logger: zerolog.Nop()}
	if err := manager.ReauthenticateGoogle(testCookieHeader); err == nil {
		t.Fatal("reauthentication without an existing Google pairing was accepted")
	}
}

func TestStaleReconnectTimerCannotReplaceNewClient(t *testing.T) {
	session := googleTestSession(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := &Manager{
		cfg:     config.Config{GMessagesEnabled: true},
		logger:  zerolog.Nop(),
		session: session,
		ctx:     ctx,
		cancel:  cancel,
		status:  Status{Enabled: true, State: "degraded", Paired: true, PairingMethod: "google"},
	}
	oldClient := manager.newClient(session)
	manager.client = oldClient
	manager.scheduleReconnect()

	replacementSession, err := cloneAuthData(session)
	if err != nil {
		t.Fatal(err)
	}
	replacementClient := manager.newClient(replacementSession)
	manager.mu.Lock()
	manager.session = replacementSession
	manager.client = replacementClient
	manager.status.State = "connecting"
	manager.mu.Unlock()

	time.Sleep(2300 * time.Millisecond)
	manager.mu.RLock()
	current := manager.client
	manager.mu.RUnlock()
	if current != replacementClient {
		t.Fatal("stale reconnect timer replaced the newly installed client")
	}
}

func TestRetiredClientEventsAreIgnored(t *testing.T) {
	session := googleTestSession(t)
	manager := &Manager{
		cfg:     config.Config{GMessagesEnabled: true},
		logger:  zerolog.Nop(),
		session: session,
		status:  Status{Enabled: true, State: "connecting", Paired: true, PairingMethod: "google"},
	}
	retired := manager.newClient(session)
	current := manager.newClient(session)
	manager.client = current

	manager.handleClientEvent(retired, &events.ClientReady{})
	manager.mu.RLock()
	retiredState := manager.status.State
	retiredConnected := manager.status.Connected
	manager.mu.RUnlock()
	if retiredState != "connecting" || retiredConnected {
		t.Fatalf("retired client event changed manager state: state=%s connected=%t", retiredState, retiredConnected)
	}

	manager.handleClientEvent(current, &events.ClientReady{})
	manager.mu.RLock()
	currentState := manager.status.State
	currentConnected := manager.status.Connected
	manager.mu.RUnlock()
	if currentState != "connected" || !currentConnected {
		t.Fatalf("current client event was ignored: state=%s connected=%t", currentState, currentConnected)
	}
}
