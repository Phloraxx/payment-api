package gmessages

import "github.com/rs/zerolog"

// libgmLogger deliberately suppresses libgm Trace/Debug output. Upstream uses
// those levels for encoded HTTP/protobuf response bodies, which can contain
// private Google Messages data and should never be retained in production logs.
// Info and above keep connection lifecycle and actionable failures visible.
func libgmLogger(logger zerolog.Logger, component string) zerolog.Logger {
	return logger.Level(zerolog.InfoLevel).With().Str("component", component).Logger()
}
