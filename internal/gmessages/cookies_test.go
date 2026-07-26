package gmessages

import "testing"

const testCookieHeader = "SID=sid-value; HSID=hsid-value; OSID=osid-value; SSID=ssid-value; APISID=apisid-value; SAPISID=sapisid-value; __Secure-1PSIDTS=optional"

func TestParseGoogleCookieInputFormats(t *testing.T) {
	cases := map[string]string{
		"raw header":       testCookieHeader,
		"cookie header":    "Cookie: " + testCookieHeader,
		"curl header":      "curl 'https://messages.google.com/web/config' -H 'accept: */*' -H 'cookie: " + testCookieHeader + "'",
		"curl cookie flag": "curl 'https://messages.google.com/web/config' --cookie '" + testCookieHeader + "'",
		"json":             `{"SID":"sid-value","HSID":"hsid-value","OSID":"osid-value","SSID":"ssid-value","APISID":"apisid-value","SAPISID":"sapisid-value"}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			cookies, err := parseGoogleCookieInput(input)
			if err != nil {
				t.Fatal(err)
			}
			if got := cookies["SAPISID"]; got != "sapisid-value" {
				t.Fatalf("SAPISID = %q", got)
			}
			if _, exists := cookies["NID"]; exists {
				t.Fatal("unrelated Google cookie was retained")
			}
		})
	}
}

func TestParseGoogleCookieInputRejectsMissingRequired(t *testing.T) {
	if _, err := parseGoogleCookieInput("SID=only-one-cookie"); err == nil {
		t.Fatal("incomplete cookie set was accepted")
	}
}
