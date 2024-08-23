package main

import "io"
import "strings"
import "testing"
import "github.com/stretchr/testify/assert"
import "net/http/httptest"


func Test_handleConfig(t *testing.T) {
	tests := []testT{
		{
			name: "fetch all configs from table",
			path: "/ldp/config",
			function: handleConfig,
			expected: `\[{"key":"config","tenant":"dummyTenant","value":"v1"}\]`,
		},
		{
			name: "fetch single config",
			path: "/ldp/config/dbinfo",
			function: handleConfigKey,
			expected: `{"key":"dbinfo","tenant":"dummyTenant","value":`,
		},
		{
			name: "non-existent config",
			path: "/ldp/config/not-there",
			function: handleConfigKey,
			errorstr: "no config item with key",
		},
		{
			name: "fetch malformed config",
			path: "/ldp/config/bad",
			function: handleConfigKey,
			errorstr: "could not deserialize",
		},
		{
			name: "translate non-string value",
			path: "/ldp/config/non-string",
			function: handleConfigKey,
			expected: `{"key":"non-string","tenant":"dummyTenant","value":"{\\|"v3\\\":42}"}`,
		},
		{
			name: "failure to reach mod-settings",
			path: "/ldp/config/non-string",
			function: handleConfig,
			errorstr: "could not fetch from mod-settings",
			useBadSession: true,
		},
		{
			name: "write a new config value",
			path: "/ldp/config/foo",
			sendData: `{"key":"foo","tenant":"xxx","value":"{\"user\":\"abc123\"}"}`,
			function: handleConfigKey,
			expected: "abc123",
		},
		{
			name: "rewrite an existing config value",
			path: "/ldp/config/dbinfo",
			sendData: `{"key":"dbinfo","tenant":"xxx","value":"{\"user\":\"abc456\"}"}`,
			function: handleConfigKey,
			expected: "abc456",
		},
		// At this point it seems silly to laboriously chase each individual error case
	}

	ts := MakeDummyModSettingsServer()
	defer ts.Close()
	baseUrl := ts.URL

	server := MakeModReportingServer(nil, nil, "")
	session, err := NewModReportingSession(server, baseUrl, "dummyTenant")
	assert.Nil(t, err)
	badSession, err := NewModReportingSession(server, "x" + baseUrl, "dummyTenant")
	assert.Nil(t, err)

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := "GET"
			var reader io.Reader
			if test.sendData != "" {
				method = "PUT"
				reader = strings.NewReader(test.sendData)
			}
			req := httptest.NewRequest(method, baseUrl + test.path, reader)
			if i == 0 {
				// Just to exercise a code-path, and get slightly more coverage *sigh*
				req.Header.Add("X-Okapi-Token", "dummy")
			}

			var currentSession = session
			if test.useBadSession {
				currentSession = badSession
			}

			w := httptest.NewRecorder()
			err = test.function(w, req, currentSession)
			resp := w.Result()

			if test.errorstr == "" {
				assert.Nil(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Regexp(t, test.expected, string(body))
			} else {
				assert.ErrorContains(t, err, test.errorstr)
			}
		})
	}
}
