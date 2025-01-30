package main

import "os"
import "testing"
import "github.com/stretchr/testify/assert"

func Test_session(t *testing.T) {
	ts := MakeDummyModSettingsServer()
	defer ts.Close()

	server, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)

	badUrl := "http://made.up.hostname.abc123:9000"

	t.Run("bad URL/tenant combo", func(t *testing.T) {
		session, err := NewModReportingSession(server, "", "diku", "dummyToken")
		assert.Nil(t, session)
		assert.ErrorContains(t, err, "no URL provided with tenant")
	})

	t.Run("resume Okapi session", func(t *testing.T) {
		session, err := NewModReportingSession(server, badUrl, "", "dummyToken")
		assert.Nil(t, err)
		assert.Equal(t, badUrl, session.url)
	})

	t.Run("session for command-line without env", func(t *testing.T) {
		session, err := NewModReportingSession(server, "", "", "dummyToken")
		assert.Nil(t, session)
		assert.ErrorContains(t, err, "missing environment variables")
	})

	t.Run("session for command-line with env but no service", func(t *testing.T) {
		os.Setenv("OKAPI_URL", badUrl)
		os.Setenv("OKAPI_TENANT", "diku")
		os.Setenv("OKAPI_USER", "mike")
		os.Setenv("OKAPI_PW", "swordfish")
		session, err := NewModReportingSession(server, "", "", "dummyToken")
		assert.Nil(t, session)
		assert.ErrorContains(t, err, "no such host")
	})

	makeGoodSession := func(t *testing.T) *ModReportingSession {
		os.Setenv("OKAPI_URL", ts.URL)
		os.Setenv("OKAPI_TENANT", "diku")
		os.Setenv("OKAPI_USER", "mike")
		os.Setenv("OKAPI_PW", "swordfish")
		session, err := NewModReportingSession(server, "", "", "dummyToken")
		assert.Nil(t, err)
		return session
	}

	t.Run("session for command-line with env and service", func(t *testing.T) {
		session := makeGoodSession(t)
		session.Log("x", "exercise the logging function just for code-coverage")
	})

	// Removing this test for now, as making the is-this-MetaDB
	// check work correctly would involve a lot of work to
	// establish a pgxmock
	/*
		t.Run("find reporting database connection", func(t *testing.T) {
			session := makeGoodSession(t)
			db, error := session.findDbConn("")
			assert.Nil(t, error)
			assert.NotNil(t, db) // That's all we can ask about this opaque object
		})
	*/
}
