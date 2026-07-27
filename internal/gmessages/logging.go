package gmessages

import "github.com/rs/zerolog"

// ProductionLogger suppresses Trace/Debug output before a logger is handed to
// libgm. Upstream uses those levels for encoded HTTP/protobuf response bodies,
// which can contain private Google Messages data and must not be retained in
// production logs. Info and above keep lifecycle and actionable failures visible.
func ProductionLogger(logger zerolog.Logger) zerolog.Logger {
	return logger.Level(zerolog.InfoLevel)
}
