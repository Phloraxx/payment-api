package gmessages

import "go.mau.fi/mautrix-gmessages/pkg/libgm"

// validPairingSession verifies the durable phone/crypto pairing independently
// from Google browser cookies. Google account cookies can expire while the
// underlying phone pairing remains reusable for reauthentication.
func validPairingSession(session *libgm.AuthData) bool {
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
