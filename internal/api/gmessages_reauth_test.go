package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestGoogleMessagesReauthEndpointRequiresDashboardAuth(t *testing.T) {
	scenarios := []tests.ApiScenario{
		{
			Name: "reauth requires dashboard auth", Method: http.MethodPost,
			URL:             "/api/connector/gmessages/reauth/google",
			Headers:         map[string]string{"Content-Type": "application/json"},
			Body:            strings.NewReader(`{"cookieData":"SID=missing-rest"}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactoryWithGMessages(t) },
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{"Dashboard authentication is required."},
		},
		{
			Name: "payment API key cannot reauthenticate Google Messages", Method: http.MethodPost,
			URL: "/api/connector/gmessages/reauth/google",
			Headers: map[string]string{
				"Authorization": "Bearer api-secret",
				"Content-Type":  "application/json",
			},
			Body:            strings.NewReader(`{"cookieData":"SID=missing-rest"}`),
			TestAppFactory:  func(t testing.TB) *tests.TestApp { return apiTestFactoryWithGMessages(t) },
			ExpectedStatus:  http.StatusUnauthorized,
			ExpectedContent: []string{"Dashboard authentication is required."},
		},
	}
	for i := range scenarios {
		scenarios[i].Test(t)
	}
}
