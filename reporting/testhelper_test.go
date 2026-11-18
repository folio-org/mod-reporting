package reporting

import "errors"
import "time"
import "fmt"
import "strings"
import "net/http"
import "net/http/httptest"
import "github.com/pashagolub/pgxmock/v3"

func Must[T any](ret T, err error) T {
	if err != nil {
		panic(err)
	}
	return ret
}

// This is useful when expectations are not met but we don't understand why.
// Uncomment the fmt.Printf line below to see the details of each match.
type LoggingMatcher struct {
	pgxmock.QueryMatcher
	log bool
}

func (m *LoggingMatcher) Match(expected string, actual string) error {
	err := m.QueryMatcher.Match(expected, actual)
	if m.log {
		fmt.Printf("[pgxmock] Matching query:\n --> expected: %s\n AND actual: %s\n  --> error: %v\n", expected, actual, err)
	}
	return err
}

// Various parts of this structure are used by different files' tests
type testT struct {
	name          string
	path          string
	sendData      string
	establishMock func(data interface{}) error
	status        int // Used only in server_test.go
	function      func(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error
	expected      string
	expectedArgs  []string // Used only in reporting_test.go/Test_makeSql
	errorstr      string
	useBadSession bool
	use2ndConfig  bool
}

// Dummy HTTP server used by multiple tests
func MakeMockHTTPServer() *httptest.Server {
	// The server's own URL, which the "tqrepos" setting below refers to.
	// It is taken from the listener before the server starts, so that it
	// is set before any request can read it, and is not derived from the
	// request.
	var serverUrl string
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Raw-file requests for the GitLab-style "test/..." repositories
		// configured in "tqrepos" below are served from the same files as
		// plain /reports/... requests. (The %2Fs in such URLs have already
		// been decoded in req.URL.Path.)
		if rest, ok := strings.CutPrefix(req.URL.Path, "/api/v4/projects/test/"); ok {
			if _, p, ok := strings.Cut(rest, "/repository/files/"); ok {
				if p, ok = strings.CutSuffix(p, "/raw"); ok {
					req.URL.Path = "/" + p
				}
			}
		}

		if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22" {
			_, _ = w.Write([]byte(`
			  {
			    "items": [
			      {
				"id": "75c12fcb-ba6c-463f-a5fc-cb0587b7d43b",
				"scope": "ui-ldp.admin",
				"key": "config",
				"value": "v1"
			      }
			    ],
			    "resultInfo": {
			      "totalRecords": 1,
			      "diagnostics": []
			    }
			  }
			`))
		} else if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22+and+key==%22dbinfo%22" {
			// XXX note that this specific value is also required by the getDbInfo test
			_, _ = w.Write([]byte(`
			  {
			    "items": [
			      {
				"id": "75c12fcb-ba6c-463f-a5fc-cb0587b7d43c",
				"scope": "ui-ldp.admin",
				"key": "dbinfo",
				"value": {
				  "url": "dummyUrl",
				  "user": "fiona",
				  "pass": "pw"
				}
			      }
			    ],
			    "resultInfo": {
			      "totalRecords": 1,
			      "diagnostics": []
			    }
			  }
			`))
		} else if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22+and+key==%22tqrepos%22" &&
			req.Header.Get("X-Okapi-Tenant") == "noRepos" {
			// Used to test what happens when the repositories cannot be loaded
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "mod-settings is unwell")
		} else if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22+and+key==%22tqrepos%22" {
			// The last two repositories are on this mock server, so that
			// reports can be fetched from them. Of these, "test/reports" is
			// open to all and "test/restricted" is not.
			_, _ = w.Write([]byte(strings.ReplaceAll(`
			  {
			    "items": [
			      {
				"id": "75c12fcb-ba6c-463f-a5fc-cb0587b7d43d",
				"scope": "ui-ldp.admin",
				"key": "tqrepos",
				"value": [
				  {
				    "type": "github",
				    "user": "seneyl",
				    "repo": "test_reports",
				    "branch": "main",
				    "dir": "circulation2",
				    "authorizedUsers": [ "11111111-1111-1111-1111-111111111111" ]
				  },
				  {
				    "type": "gitlab",
				    "baseUrl": "https://gitlab.example.com",
				    "user": "mike",
				    "repo": "reports",
				    "branch": "dev",
				    "dir": "/"
				  },
				  {
				    "type": "gitlab",
				    "baseUrl": "MOCK-SERVER",
				    "user": "test",
				    "repo": "reports",
				    "branch": "main",
				    "dir": "reports"
				  },
				  {
				    "type": "gitlab",
				    "baseUrl": "MOCK-SERVER",
				    "user": "test",
				    "repo": "restricted",
				    "branch": "main",
				    "dir": "reports",
				    "authorizedUsers": [ "22222222-2222-2222-2222-222222222222" ]
				  }
				]
			      }
			    ],
			    "resultInfo": {
			      "totalRecords": 1,
			      "diagnostics": []
			    }
			  }
			`, "MOCK-SERVER", serverUrl)))
		} else if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22+and+key==%22non-string%22" {
			_, _ = w.Write([]byte(`
			  {
			    "items": [
			      {
				"id": "75c12fcb-ba6c-463f-a5fc-cb0587b7d43c",
				"scope": "ui-ldp.admin",
				"key": "non-string",
				"value": { "v3": 42 }
			      }
			    ],
			    "resultInfo": {
			      "totalRecords": 1,
			      "diagnostics": []
			    }
			  }
			`))
		} else if req.URL.Path == "/settings/entries" &&
			req.URL.RawQuery == "query=scope==%22ui-ldp.admin%22+and+key==%22bad%22" {
			_, _ = w.Write([]byte("some bit of text"))
		} else if req.URL.Path == "/settings/entries" {
			// Searching for some other setting, e.g. "score" before trying to write to it
			_, _ = w.Write([]byte(`
			  {
			    "items": [],
			    "resultInfo": {
			      "totalRecords": 0,
			      "diagnostics": []
			    }
			  }
			`))
		} else if req.URL.Path == "/settings/entries/75c12fcb-ba6c-463f-a5fc-cb0587b7d43c" {
			// Nothing to do
		} else if after, ok := strings.CutPrefix(req.URL.Path, "/repos/seneyl/test_reports/compare/"); ok {
			// The GitHub compare API, as used by commitIsOnBranch. The
			// result depends on the first character of the commit SHA.
			base, head, _ := strings.Cut(after, "...")
			status := ""
			switch head[0] {
			case 'a':
				status = "behind"
			case 'b':
				status = "identical"
			case 'c':
				status = "ahead"
			case 'd':
				status = "diverged"
			case 'e':
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintln(w, `{"message":"Not Found"}`)
				return
			case 'f':
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintln(w, `{"message":"API rate limit exceeded"}`)
				return
			case '9':
				_, _ = w.Write([]byte("not JSON"))
				return
			}
			// Commits are on "feature/x" only if they are identical to its head
			if base == "feature/x" && status != "identical" {
				status = "diverged"
			}
			fmt.Fprintf(w, `{"status":%q}`, status)
		} else if req.URL.Path == "/reports/redirect.sql" {
			// A relative redirect, as mod-reporting should report the absolute URL
			http.Redirect(w, req, "/reports/loans.sql", http.StatusTemporaryRedirect)
		} else if req.URL.Path == "/reports/slow.sql" {
			// Outlasts the shortened reportFetchTimeout that tests use
			time.Sleep(500 * time.Millisecond)
			_, _ = w.Write([]byte(`--metadb:function slow`))
		} else if req.URL.Path == "/reports/noheader.sql" {
			_, _ = w.Write([]byte(`this is a bad report`))
		} else if req.URL.Path == "/reports/bad.sql" {
			_, _ = w.Write([]byte(`--metadb:function users\nthis is bad SQL`))
		} else if req.URL.Path == "/reports/loans.sql" {
			_, _ = w.Write([]byte(`--metadb:function count_loans

DROP FUNCTION IF EXISTS count_loans;

CREATE FUNCTION count_loans(
    start_date date DEFAULT '1000-01-01',
    end_date date DEFAULT '3000-01-01')
RETURNS TABLE(
    item_id uuid,
    loan_count bigint)
AS $$
SELECT item_id,
       count(*) AS loan_count
    FROM folio_circulation.loan__t
    WHERE start_date <= loan_date AND loan_date < end_date
    GROUP BY item_id
$$
LANGUAGE SQL
STABLE
PARALLEL SAFE;
`))
		} else if req.URL.Path == "/authn/login-with-expiry" {
			// Attempted login to create new FOLIO session
			fmt.Fprintln(w, `{"accessTokenExpiration":"2023-12-22T12:35:47Z"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, "Not found")
		}
	}))
	serverUrl = "http://" + ts.Listener.Addr().String()
	ts.Start()
	return ts
}

// Functions to establish pgxmock expectations, used by multiple tests
func establishMockForTables(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery("SELECT schema_name, table_name FROM metadb.base_table").WillReturnRows(
		pgxmock.NewRows([]string{"schema_name", "table_name"}).
			AddRow("folio_inventory", "records_instances").
			AddRow("folio_inventory", "holdings_record"))
	return nil
}

func establishMockForColumns(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery(`SELECT column_name, data_type, ordinal_position, table_schema, table_name FROM information_schema.columns`).
		WithArgs("folio_users", "users", "data").
		WillReturnRows(pgxmock.NewRows([]string{"column_name", "data_type", "ordinal_position", "table_schema", "table_name"}).
			AddRow("id", "uuid", "6", "folio_users", "users").
			AddRow("user", "string", "7", "folio_users", "users").
			AddRow("creation_date", "timestamp without time zone", "8", "folio_users", "users"))
	return nil
}

func establishMockForQuery(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery(`SELECT \* FROM "folio_users"."users"`).
		WillReturnRows(pgxmock.NewRows([]string{"name", "email"}).
			AddRow("mike", "mike@example.com").
			AddRow("fiona", "fiona@example.com"))
	return nil
}

func establishMockForEmptyFilterQuery(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery(`SELECT \* FROM "folio_users"."users"`).
		WillReturnError(errors.New(`ERROR: syntax error at or near "=" (SQLSTATE 42601)`))
	return nil
}

func establishMockForReport(mock pgxmock.PgxPoolIface) error {
	mock.ExpectBegin()
	mock.ExpectExec("--metadb:function count_loans").
		WillReturnResult(pgxmock.NewResult("CREATE FUNCTION", 1))
	mock.ExpectExec(`SET statement_timeout TO 60000`).
		WillReturnResult(pgxmock.NewResult("SET", 1))
	id := [16]uint8{90, 154, 146, 202, 186, 5, 215, 45, 248, 76, 49, 146, 31, 31, 126, 77}
	mock.ExpectQuery(`SELECT \* FROM count_loans\(end_date => \$1\)`).
		WithArgs("2023-03-18T00:00:00.000Z").
		WillReturnRows(pgxmock.NewRows([]string{"id", "num"}).
			AddRow(id, 29).
			AddRow("456", 3))
	mock.ExpectRollback()
	return nil
}

func establishMockForLogs(mock pgxmock.PgxPoolIface) error {
	ts1, _ := time.Parse(time.RFC3339, "2023-10-04T23:38:57.662+01:00")
	ts2, _ := time.Parse(time.RFC3339, "2023-10-05T00:40:25.571+01:00")
	ts3, _ := time.Parse(time.RFC3339, "2023-10-05T00:34:14.76+01:00")

	mock.ExpectQuery("SELECT log_time, error_severity, message FROM metadb.log").WillReturnRows(
		pgxmock.NewRows([]string{"log_time", "error_severity", "message"}).
			AddRow(ts1, "INFO", "starting Metadb v1.2.0-beta7").
			AddRow(ts2, "INFO", "source \"folio\" snapshot complete").
			AddRow(ts3, "WARNING", "runsql: operator does not exist"))
	return nil
}

func establishMockForVersion(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery(`SELECT mdbversion()`).
		WillReturnRows(pgxmock.NewRows([]string{"mdbversion"}).
			AddRow("Metadb v1.2.7"))
	return nil
}

func establishMockForUpdates(mock pgxmock.PgxPoolIface) error {
	ts1 := Must(time.Parse(time.RFC3339, "2025-01-24T00:59:48.421+00:00"))

	mock.ExpectQuery(`SELECT schema_name, table_name, last_update, elapsed_real_time ` +
		`FROM metadb.table_update ORDER BY elapsed_real_time DESC`).
		WillReturnRows(pgxmock.NewRows([]string{"schema_name", "table_name", "last_update", "elapsed_real_time"}).
			AddRow("folio_derived", "agreements_package_content_item", ts1, float32(0.0452)))
	return nil
}

func establishMockForProcesses(mock pgxmock.PgxPoolIface) error {
	mock.ExpectQuery(`SELECT dbname, username, state, realtime, query FROM ps\(\) ORDER BY realtime DESC`).
		WillReturnRows(pgxmock.NewRows([]string{"dbname", "username", "state", "realtime", "query"}).
			AddRow("metadb_indexdata_test", "folio_app", "active", "00:00:04", "select a.message, b.message from metadb.log as a, metadb.log as b;"))
	return nil
}
