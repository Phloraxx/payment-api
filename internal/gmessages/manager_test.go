package gmessages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Phloraxx/payment-api/internal/config"

	"go.mau.fi/mautrix-gmessages/pkg/libgm"
	"go.mau.fi/mautrix-gmessages/pkg/libgm/gmproto"
)

func TestSessionPersistenceUsesRestrictedPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "gmessages")
	path := filepath.Join(dir, "session.json")
	session := libgm.NewAuthData()
	if err := saveSession(path, session); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("session mode = %o; want 600", got)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got&0o077 != 0 {
		t.Fatalf("session directory is group/world accessible: %o", got)
	}
	loaded, err := loadSession(path)
	if err != nil || loaded == nil || loaded.RefreshKey == nil || loaded.RequestCrypto == nil {
		t.Fatalf("loaded session = %#v, %v", loaded, err)
	}
}

func TestMessageBodyJoinsTextPartsOnly(t *testing.T) {
	wrapped := &libgm.WrappedMessage{Message: &gmproto.Message{MessageInfo: []*gmproto.MessageInfo{
		{Data: &gmproto.MessageInfo_MessageContent{MessageContent: &gmproto.MessageContent{Content: "Received Rs.100.01"}}},
		{Data: &gmproto.MessageInfo_MediaContent{MediaContent: &gmproto.MediaContent{MediaName: "ignored.jpg"}}},
		{Data: &gmproto.MessageInfo_MessageContent{MessageContent: &gmproto.MessageContent{Content: "UPI Ref:123456789012"}}},
	}}}
	got := messageBody(wrapped)
	want := "Received Rs.100.01\nUPI Ref:123456789012"
	if got != want {
		t.Fatalf("messageBody() = %q; want %q", got, want)
	}
}

func TestNormalizeTimestampHandlesMicrosAndMillis(t *testing.T) {
	if got := normalizeTimestampMS(1_700_000_000_123); got != 1_700_000_000_123 {
		t.Fatalf("millis changed: %d", got)
	}
	if got := normalizeTimestampMS(1_700_000_000_123_000); got != 1_700_000_000_123 {
		t.Fatalf("micros = %d", got)
	}
}

func TestOutgoingStatusDetectionIsConservative(t *testing.T) {
	for _, status := range []string{"OUTGOING_COMPLETE", "SENT_BY_ME"} {
		if !isFromMe(status) {
			t.Errorf("%q should be outgoing", status)
		}
	}
	if isFromMe("INCOMING_COMPLETE") {
		t.Fatal("incoming message classified as outgoing")
	}
}

func TestBeginPairRefusesToReplaceExistingSession(t *testing.T) {
	session := libgm.NewAuthData()
	session.Browser = &gmproto.Device{}
	session.Mobile = &gmproto.Device{}
	session.TachyonAuthToken = []byte{1}
	manager := &Manager{
		cfg:     config.Config{GMessagesEnabled: true},
		session: session,
	}
	if _, err := manager.BeginPair(); err == nil || !strings.Contains(err.Error(), "already paired") {
		t.Fatalf("BeginPair() error = %v; want already paired guard", err)
	}
}

func TestGoogleSessionRequiresCookies(t *testing.T) {
	session := libgm.NewAuthData()
	session.Browser = &gmproto.Device{}
	session.Mobile = &gmproto.Device{SourceID: "user@example.com"}
	session.TachyonAuthToken = []byte{1}
	session.DestRegID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if validSession(session) {
		t.Fatal("Google-account session without cookies was accepted")
	}
	cookies, err := parseGoogleCookieInput(testCookieHeader)
	if err != nil {
		t.Fatal(err)
	}
	session.SetCookies(cookies)
	if !validSession(session) {
		t.Fatal("complete Google-account session with required cookies was rejected")
	}
	if got := sessionPairingMethod(session); got != "google" {
		t.Fatalf("pairing method = %q", got)
	}
}

func TestQRPairRefusesDuringGooglePairing(t *testing.T) {
	manager := &Manager{
		cfg:    config.Config{GMessagesEnabled: true},
		status: Status{Enabled: true, State: "pairing", PairingMethod: "google"},
	}
	if _, err := manager.BeginPair(); err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("BeginPair() error = %v; want in-progress guard", err)
	}
}

func TestValidSessionRejectsPartialCredentialState(t *testing.T) {
	partial := &libgm.AuthData{Browser: &gmproto.Device{}, TachyonAuthToken: []byte{1}}
	if validSession(partial) {
		t.Fatal("partial session without mobile/crypto/refresh key was accepted")
	}
	complete := libgm.NewAuthData()
	complete.Browser = &gmproto.Device{}
	complete.Mobile = &gmproto.Device{}
	complete.TachyonAuthToken = []byte{1}
	if !validSession(complete) {
		t.Fatal("complete session was rejected")
	}
}
