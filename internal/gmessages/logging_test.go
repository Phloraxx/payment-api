package gmessages

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestLibGMLoggerSuppressesTraceAndDebug(t *testing.T) {
	var output bytes.Buffer
	base := zerolog.New(&output).Level(zerolog.TraceLevel)
	logger := libgmLogger(base, "libgm")

	logger.Trace().Str("response_body", "sensitive-trace").Msg("trace message")
	logger.Debug().Str("response_body", "sensitive-debug").Msg("debug message")
	logger.Info().Msg("connection lifecycle")
	logger.Warn().Msg("actionable warning")

	got := output.String()
	for _, forbidden := range []string{"sensitive-trace", "sensitive-debug", "trace message", "debug message"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("libgm logger emitted suppressed content %q: %s", forbidden, got)
		}
	}
	for _, wanted := range []string{"connection lifecycle", "actionable warning", `"component":"libgm"`} {
		if !strings.Contains(got, wanted) {
			t.Fatalf("libgm logger missing %q: %s", wanted, got)
		}
	}
}
