package main

import "os"
import "testing"
import "github.com/stretchr/testify/assert"

func Test_getDbInfo(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()
	baseUrl := ts.URL

	cfg, err := readConfig("../etc/silent.json");
	assert.Nil(t, err)
	server := MakeModReportingServer(cfg, nil, "")
	session, err := NewModReportingSession(server, baseUrl, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	t.Run("info from environment", func(t *testing.T) {
		url := "http://metadb.example.com:12345/db"
		os.Setenv("REPORTING_DB_URL", url)
		os.Setenv("REPORTING_DB_USER", "mike")
		os.Setenv("REPORTING_DB_PASS", "swordfish")
		dburl, dbuser, dbpass, err := getDbInfo(session.folioSession, "")
		assert.Nil(t, err)
		assert.Equal(t, url, dburl)
		assert.Equal(t, "mike", dbuser)
		assert.Equal(t, "swordfish", dbpass)
	})

	t.Run("info from FOLIO", func(t *testing.T) {
		os.Setenv("REPORTING_DB_URL", "")
		os.Setenv("REPORTING_DB_USER", "")
		os.Setenv("REPORTING_DB_PASS", "")
		dburl, dbuser, dbpass, err := getDbInfo(session.folioSession, "")
		assert.Nil(t, err)
		assert.Equal(t, "dummyUrl", dburl)
		assert.Equal(t, "fiona", dbuser)
		assert.Equal(t, "pw", dbpass)
	})
}
