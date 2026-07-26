package gmessages

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var requiredGoogleCookies = []string{"SID", "HSID", "OSID", "SSID", "APISID", "SAPISID"}
var allowedGoogleCookies = map[string]struct{}{
	"SID": {}, "HSID": {}, "OSID": {}, "SSID": {}, "APISID": {}, "SAPISID": {}, "__Secure-1PSIDTS": {},
}

var (
	curlHeaderCookieRE = regexp.MustCompile(`(?is)(?:-H|--header)\s+(?:'([^']*cookie\s*:[^']*)'|"([^"]*cookie\s*:[^"]*)")`)
	curlCookieRE       = regexp.MustCompile(`(?is)(?:-b|--cookie)\s+(?:'([^']*)'|"([^"]*)")`)
)

func parseGoogleCookieInput(input string) (map[string]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, errors.New("google cookie data is required")
	}

	var cookies map[string]string
	if strings.HasPrefix(input, "{") {
		if err := json.Unmarshal([]byte(input), &cookies); err != nil {
			return nil, errors.New("cookie JSON is invalid")
		}
	} else {
		header := extractCookieHeader(input)
		cookies = parseCookieHeader(header)
	}
	if err := validateGoogleCookies(cookies); err != nil {
		return nil, err
	}
	filtered := make(map[string]string, len(allowedGoogleCookies))
	for name := range allowedGoogleCookies {
		if value := cookies[name]; value != "" {
			filtered[name] = value
		}
	}
	return filtered, nil
}

func extractCookieHeader(input string) string {
	if match := curlHeaderCookieRE.FindStringSubmatch(input); len(match) > 0 {
		header := firstNonEmpty(match[1], match[2])
		if idx := strings.Index(strings.ToLower(header), "cookie:"); idx >= 0 {
			return strings.TrimSpace(header[idx+len("cookie:"):])
		}
	}
	if match := curlCookieRE.FindStringSubmatch(input); len(match) > 0 {
		return strings.TrimSpace(firstNonEmpty(match[1], match[2]))
	}
	if idx := strings.Index(strings.ToLower(input), "cookie:"); idx >= 0 {
		line := input[idx+len("cookie:"):]
		if end := strings.IndexAny(line, "\r\n"); end >= 0 {
			line = line[:end]
		}
		return strings.Trim(strings.TrimSpace(line), "'\"")
	}
	return input
}

func parseCookieHeader(header string) map[string]string {
	cookies := make(map[string]string)
	for _, part := range strings.Split(header, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || name == "" {
			continue
		}
		cookies[name] = strings.TrimSpace(value)
	}
	return cookies
}

func validateGoogleCookies(cookies map[string]string) error {
	var missing []string
	for _, name := range requiredGoogleCookies {
		if strings.TrimSpace(cookies[name]) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing required Google cookies: %s", strings.Join(missing, ", "))
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
