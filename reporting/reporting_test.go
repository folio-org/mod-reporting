package reporting

import "io"
import "strings"
import "fmt"
import "time"
import "testing"
import "encoding/json"
import "github.com/stretchr/testify/assert"
import "github.com/pashagolub/pgxmock/v3"
import "errors"
import "net/http"
import "net/http/httptest"

func Test_makeSql(t *testing.T) {
	uuid := "4f41bd4c-09fb-41a0-8f18-c347f4e81877"

	tests := []testT{
		{
			name:     "empty query",
			sendData: `{}`,
			errorstr: "query must have exactly one table",
		},
		{
			name:     "query with empty tables",
			sendData: `{ "tables": [] }`,
			errorstr: "query must have exactly one table",
		},
		{
			name:     "simplest query",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users" }] }`,
			expected: `SELECT * FROM "folio_users"."users"`,
		},
		{
			name: "query with columns",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				 "showColumns": ["id", "username"] }] }`,
			expected:     `SELECT id, username FROM "folio_users"."users"`,
			expectedArgs: []string{},
		},
		{
			name: "query with empty condition",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [{}] }] }`,
			expected:     `SELECT * FROM "folio_users"."users"`,
			expectedArgs: []string{},
		},
		{
			name: "query with implicit condition",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [
					{ "key": "id", "value": "` + uuid + `" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" WHERE id = $1`,
			expectedArgs: []string{uuid},
		},
		{
			name: "query on invalid column",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [
					{ "key": "xid", "value": "43" }
				] }] }`,
			errorstr: "filter on invalid column",
		},
		{
			name: "query with invalid uuid",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [
					{ "key": "id", "value": "43" }
				] }] }`,
			errorstr: "invalid value for field id",
		},
		{
			name: "query with multiple conditions",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [
					{ "key": "id", "op": "<>", "value": "` + uuid + `" },
					{ "key": "creation_date", "op": ">", "value": "1968-03-12" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" WHERE id <> $1 AND creation_date > $2`,
			expectedArgs: []string{uuid, "1968-03-12"},
		},
		{
			name: "query with real and empty conditions",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"columnFilters": [
					{},
					{ "key": "user", "op": "LIKE", "value": "mi%" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" WHERE user LIKE $2`,
			expectedArgs: []string{"mi%"},
		},
		{
			name: "query with order",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"orderBy": [
					{ "key": "user", "direction": "asc", "nulls": "start" },
					{ "key": "id", "direction": "desc", "nulls": "end" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" ORDER BY user asc NULLS FIRST, id desc NULLS LAST`,
			expectedArgs: []string{},
		},
		{
			name: "query with missing first order",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"orderBy": [
					{ "direction": "asc", "nulls": "start" },
					{ "key": "id", "direction": "desc", "nulls": "end" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" ORDER BY id desc NULLS LAST`,
			expectedArgs: []string{},
		},
		{
			name: "query with missing last order",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users",
				"orderBy": [
					{ "key": "user", "direction": "asc", "nulls": "start" },
					{ "direction": "desc", "nulls": "end" }
				] }] }`,
			expected:     `SELECT * FROM "folio_users"."users" ORDER BY user asc NULLS FIRST`,
			expectedArgs: []string{},
		},
		{
			name:         "query with limit",
			sendData:     `{ "tables": [{ "schema": "folio_users", "tableName": "users", "limit": 99 }] }`,
			expected:     `SELECT * FROM "folio_users"."users" LIMIT 99`,
			expectedArgs: []string{},
		},
		{
			name:         "query with limit expressed a string",
			sendData:     `{ "tables": [{ "schema": "folio_users", "tableName": "users", "limit": "100" }] }`,
			expected:     `SELECT * FROM "folio_users"."users" LIMIT 100`,
			expectedArgs: []string{},
		},
		{
			name:         "make me one with everything",
			sendData:     `{ "tables": [{"limit": 11,"schema": "folio_users","orderBy": [{"direction": "asc","nulls": "end","key": "creation_date"},{"direction": "asc","nulls": "start","key": "__id"}],"showColumns": ["id","creation_date","hrid","title","source"],"columnFilters": [{"key": "creation_date","op": ">=","value": "2022-06-09T19:01:33.757+00:00"},{"key": "id","op": "<>","value": "` + uuid + `"}],"tableName": "users"}]}`,
			expected:     `SELECT id, creation_date, hrid, title, source FROM "folio_users"."users" WHERE creation_date >= $1 AND id <> $2 ORDER BY creation_date asc NULLS LAST, __id asc NULLS FIRST LIMIT 11`,
			expectedArgs: []string{"2022-06-09T19:01:33.757+00:00", uuid},
		},
	}

	ts := MakeMockHTTPServer()
	defer ts.Close()
	mrs, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)
	session, err := NewModReportingSession(mrs, ts.URL, "dummyTenant", "dummyToken")
	assert.Nil(t, err)
	mockPostgres, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(&LoggingMatcher{
		QueryMatcher: pgxmock.QueryMatcherRegexp,
		log:          false,
	}))
	assert.Nil(t, err)
	session.dbConn = mockPostgres

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bytes := []byte(test.sendData)
			var jq jsonQuery
			err := json.Unmarshal(bytes, &jq)
			assert.Nil(t, err)

			err = establishMockForColumns(mockPostgres)
			assert.Nil(t, err)

			sql, params, err := makeSql(jq, session, "")
			if test.errorstr == "" {
				assert.Nil(t, err)
				assert.Equal(t, test.expected, sql)
				assert.Equal(t, len(test.expectedArgs), len(params))
				for i, val := range params {
					assert.EqualValues(t, test.expectedArgs[i], val)
				}
			} else {
				assert.ErrorContains(t, err, test.errorstr)
			}
		})
	}
}

func Test_reportingHandlers(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()
	baseUrl := ts.URL

	// Short enough that the "slow.sql" report times out without slowing the tests much
	savedTimeout := reportFetchTimeout
	reportFetchTimeout = 100 * time.Millisecond
	t.Cleanup(func() { reportFetchTimeout = savedTimeout })

	// URL of a report in the GitLab-style "test/reports" repository that
	// the mock server's "tqrepos" setting places on the mock server itself
	reportUrl := func(name string) string {
		return baseUrl + "/api/v4/projects/test%2Freports/repository/files/reports%2F" + name + "/raw?ref=main"
	}

	tests := []testT{
		{
			name:          "bad DB connection for tables",
			useBadSession: true,
			function:      handleTables,
			errorstr:      "failed to connect",
		},
		{
			name:          "bad DB connection for columns",
			path:          "/ldp/db/columns?schema=folio_users&table=users",
			useBadSession: true,
			function:      handleColumns,
			errorstr:      "failed to connect",
		},
		{
			name:          "bad DB connection for logs",
			useBadSession: true,
			function:      handleLogs,
			errorstr:      "failed to connect",
		},
		{
			name:          "bad DB connection for version",
			useBadSession: true,
			function:      handleVersion,
			errorstr:      "failed to connect",
		},
		{
			name:          "bad DB connection for updates",
			useBadSession: true,
			function:      handleUpdates,
			errorstr:      "failed to connect",
		},
		{
			name:          "bad DB connection for processes",
			useBadSession: true,
			function:      handleProcesses,
			errorstr:      "failed to connect",
		},
		{
			name: "retrieve list of tables",
			path: "/ldp/db/tables",
			establishMock: func(data interface{}) error {
				return establishMockForTables(data.(pgxmock.PgxPoolIface))
			},
			function: handleTables,
			expected: `\[{"tableSchema":"folio_inventory","tableName":"records_instances"},{"tableSchema":"folio_inventory","tableName":"holdings_record"}\]`,
		},
		{
			name:     "list of columns without table",
			path:     "/ldp/db/columns?schema=folio_users",
			function: handleColumns,
			errorstr: "must specify both schema and table",
		},
		{
			name:     "list of columns without schema",
			path:     "/ldp/db/columns?table=users",
			function: handleColumns,
			errorstr: "must specify both schema and table",
		},
		{
			name: "retrieve list of columns",
			path: "/ldp/db/columns?schema=folio_users&table=users",
			establishMock: func(data interface{}) error {
				return establishMockForColumns(data.(pgxmock.PgxPoolIface))
			},
			function: handleColumns,
			expected: `{"columnName":"id","data_type":"uuid","tableSchema":"folio_users","tableName":"users","ordinalPosition":"6"},{"columnName":"user","data_type":"string","tableSchema":"folio_users","tableName":"users","ordinalPosition":"7"},{"columnName":"creation_date","data_type":"timestamp without time zone","tableSchema":"folio_users","tableName":"users","ordinalPosition":"8"}]`,
		},
		{
			name:     "fail non-JSON query",
			path:     "/ldp/db/query",
			sendData: "water",
			function: handleQuery,
			errorstr: "deserialize JSON",
		},
		{
			name:     "fail non-JSON query",
			path:     "/ldp/db/query",
			sendData: `{}`,
			function: handleQuery,
			errorstr: "must have exactly one table",
		},
		{
			name:     "fail JSON query where tables is number",
			path:     "/ldp/db/query",
			sendData: `{ "tables": 42 }`,
			function: handleQuery,
			errorstr: "/cannot unmarshal (JSON )?number/",
		},
		{
			name:     "fail JSON query where tables is string",
			path:     "/ldp/db/query",
			sendData: `{ "tables": "water" }`,
			function: handleQuery,
			errorstr: "/cannot unmarshal (JSON )?string/",
		},
		{
			name:     "fail JSON query with 0 tables",
			path:     "/ldp/db/query",
			sendData: `{ "tables": [] }`,
			function: handleQuery,
			errorstr: "must have exactly one table",
		},
		{
			name:     "fail JSON query with 2 tables",
			path:     "/ldp/db/query",
			sendData: `{ "tables": [{}, {}] }`,
			function: handleQuery,
			errorstr: "must have exactly one table",
		},
		{
			name:     "fail JSON query where table is number",
			path:     "/ldp/db/query",
			sendData: `{ "tables": [42] }`,
			function: handleQuery,
			errorstr: "/cannot unmarshal (JSON )?number/",
		},
		{
			name:     "fail JSON query where table is string",
			path:     "/ldp/db/query",
			sendData: `{ "tables": ["water"] }`,
			function: handleQuery,
			errorstr: "/cannot unmarshal (JSON )?string/",
		},
		{
			name:     "simple query with dummy results",
			path:     "/ldp/db/query",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users" }] }`,
			establishMock: func(data interface{}) error {
				return establishMockForQuery(data.(pgxmock.PgxPoolIface))
			},
			function: handleQuery,
			expected: `\[{"name":"mike","email":"mike@example.com"},{"name":"fiona","email":"fiona@example.com"}\]`,
		},
		{
			// This test doesn't really test anything except my ability to mock PGX errors
			name:     "query with an empty filter",
			path:     "/ldp/db/query",
			sendData: `{ "tables": [{ "schema": "folio_users", "tableName": "users", "columnFilters": [{}] }] }`,
			establishMock: func(data interface{}) error {
				return establishMockForEmptyFilterQuery(data.(pgxmock.PgxPoolIface))
			},
			function: handleQuery,
			errorstr: `ERROR: syntax error at or near "=" (SQLSTATE 42601)`,
		},
		{
			name:     "malformed report",
			path:     "/ldp/db/reports",
			sendData: `a non-JSON string`,
			function: handleReport,
			errorstr: "deserialize JSON",
		},
		{
			name:     "report without URL",
			path:     "/ldp/db/reports",
			sendData: `{}`,
			function: handleReport,
			errorstr: "it is not in any configured report repository",
		},
		{
			name:     "report with 404 URL",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("missing.sql") + `" }`,
			function: handleReport,
			errorstr: "404 Not Found",
		},
		{
			name:     "report URL that redirects",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("redirect.sql") + `" }`,
			function: handleReport,
			errorstr: "could not fetch report from " + reportUrl("redirect.sql") + ": it redirects (307 Temporary Redirect) to " +
				baseUrl + "/reports/loans.sql, and redirects are not followed: " +
				"the report repository may need to be configured with the URL it redirects to",
		},
		{
			name:     "report URL that is too slow to respond",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("slow.sql") + `" }`,
			function: handleReport,
			errorstr: "/could not fetch report from .*Client.Timeout exceeded/",
		},
		{
			name:     "report without function declaration",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("noheader.sql") + `" }`,
			function: handleReport,
			errorstr: "could not extract SQL function name",
		},
		{
			name:     "report that is not valid SQL",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("bad.sql") + `" }`,
			// pgxmock can't spot the badness of the SQL, so we manually cause an error
			establishMock: func(data interface{}) error {
				mock := data.(pgxmock.PgxPoolIface)
				mock.ExpectBegin()
				mock.ExpectExec("--metadb:function users").
					WillReturnError(fmt.Errorf("bad SQL"))
				mock.ExpectRollback()
				return nil
			},
			function: handleReport,
			errorstr: "could not register SQL function: bad SQL",
		},
		{
			name:     "simple report",
			path:     "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("loans.sql") + `" }`,
			establishMock: func(data interface{}) error {
				mock := data.(pgxmock.PgxPoolIface)
				mock.ExpectBegin()
				mock.ExpectExec("--metadb:function count_loans").
					WillReturnResult(pgxmock.NewResult("CREATE FUNCTION", 1))
				mock.ExpectExec(`SET statement_timeout TO 60000`).
					WillReturnResult(pgxmock.NewResult("SET", 1))
				mock.ExpectQuery(`SELECT \* FROM count_loans`).
					WillReturnRows(pgxmock.NewRows([]string{"id", "num"}).
						AddRow("123", 42).
						AddRow("456", 96))
				mock.ExpectRollback()
				return nil
			},
			function: handleReport,
			expected: `{"totalRecords":2,"records":\[{"id":"123","num":42},{"id":"456","num":96}\]}`,
		},
		{
			name: "report with parameters, limit and UUID",
			path: "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("loans.sql") + `",
				     "params": { "end_date": "2023-03-18T00:00:00.000Z" },
				     "limit": 100
				   }`,
			establishMock: func(data interface{}) error {
				return establishMockForReport(data.(pgxmock.PgxPoolIface))
			},
			function: handleReport,
			expected: `{"totalRecords":2,"records":\[{"id":"5a9a92ca-ba05-d72d-f84c-31921f1f7e4d","num":29},{"id":"456","num":3}\]}`,
		},
		{
			name: "report with limit expressed as string",
			path: "/ldp/db/reports",
			sendData: `{ "url": "` + reportUrl("loans.sql") + `",
				     "params": { "end_date": "2023-03-18T00:00:00.000Z" },
				     "limit": "101"
				   }`,
			establishMock: func(data interface{}) error {
				return establishMockForReport(data.(pgxmock.PgxPoolIface))
			},
			function: handleReport,
			expected: `{"totalRecords":2,"records":\[{"id":"5a9a92ca-ba05-d72d-f84c-31921f1f7e4d","num":29},{"id":"456","num":3}\]}`,
		},
		{
			name:         "no match with whitelist",
			use2ndConfig: true,
			path:         "/ldp/db/reports",
			sendData:     `{ "url": "` + baseUrl + `/reports/loans.sql" }`,
			establishMock: func(data interface{}) error {
				return establishMockForReport(data.(pgxmock.PgxPoolIface))
			},
			function: handleReport,
			errorstr: "report URL did not match any whitelist regular expression",
		},
		{
			// The whitelist is checked first, so getting this far shows it matched
			name:         "match with whitelist, but not in any repository",
			use2ndConfig: true,
			path:         "/ldp/db/reports",
			sendData:     `{ "url": "https://gitlab.com/MikeTaylor/metadb-queries/non/existent.sql" }`,
			establishMock: func(data interface{}) error {
				return establishMockForReport(data.(pgxmock.PgxPoolIface))
			},
			function: handleReport,
			errorstr: "it is not in any configured report repository",
		},
		{
			name: "retrieve logs",
			path: "/ldp/db/logs",
			establishMock: func(data interface{}) error {
				return establishMockForLogs(data.(pgxmock.PgxPoolIface))
			},
			function: handleLogs,
			expected: `\[{"log_time":"2023-10-04T23:38:57.662\+01:00","error_severity":"INFO","message":"starting Metadb v1.2.0-beta7"}.*exist"}\]`,
		},
		{
			name: "retrieve version",
			path: "/ldp/db/version",
			establishMock: func(data interface{}) error {
				return establishMockForVersion(data.(pgxmock.PgxPoolIface))
			},
			function: handleVersion,
			expected: `{"rawVersion":"Metadb v1.2.7","version":"1.2.7"}`,
		},
		{
			name: "retrieve updates",
			path: "/ldp/db/updates",
			establishMock: func(data interface{}) error {
				return establishMockForUpdates(data.(pgxmock.PgxPoolIface))
			},
			function: handleUpdates,
			expected: `\[{"tableSchema":"folio_derived","tableName":"agreements_package_content_item","lastUpdate":"2025-01-24T00:59:48.421Z","elapsedRealTime":0.0452}\]`,
		},
		{
			name: "retrieve processes",
			path: "/ldp/db/processes",
			establishMock: func(data interface{}) error {
				return establishMockForProcesses(data.(pgxmock.PgxPoolIface))
			},
			function: handleProcesses,
			expected: `\[{"databaseName":"metadb_indexdata_test","userName":"folio_app","state":"active","realTime":"00:00:04","query":"select a.message, b.message from metadb.log as a, metadb.log as b;"}\]`,
		},
	}

	mrs1, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)
	session1, err := NewModReportingSession(mrs1, baseUrl, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	mrs2, err := MakeConfiguredServer("../etc/silent-with-whitelist.json", ".")
	assert.Nil(t, err)
	session2, err := NewModReportingSession(mrs2, baseUrl, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := "GET"
			var reader io.Reader
			if test.sendData != "" {
				method = "POST"
				reader = strings.NewReader(test.sendData)
			}
			req := httptest.NewRequest(method, baseUrl+test.path, reader)

			mock, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(&LoggingMatcher{
				QueryMatcher: pgxmock.QueryMatcherRegexp,
				log:          false,
			}))
			assert.Nil(t, err)
			defer mock.Close()

			if test.establishMock != nil {
				err = test.establishMock(mock)
				assert.Nil(t, err)
			}

			var session *ModReportingSession
			if !test.use2ndConfig {
				session = session1
			} else {
				session = session2
			}

			if test.useBadSession {
				session.dbConn = nil
			} else {
				session.dbConn = mock
				session.isMDB = true // Mock expectations are as for MetaDB
			}

			w := httptest.NewRecorder()
			err = test.function(w, req, session)
			resp := w.Result()

			if test.errorstr == "" {
				assert.Nil(t, err)
				assert.Equal(t, 200, resp.StatusCode)
				body, _ := io.ReadAll(resp.Body)
				assert.Regexp(t, test.expected, string(body))
				assert.Nil(t, mock.ExpectationsWereMet(), "unfulfilled expections")
			} else if strings.HasPrefix(test.errorstr, "/") && strings.HasSuffix(test.errorstr, "/") {
				regexpstr := test.errorstr[1 : len(test.errorstr)-1]
				assert.Regexp(t, regexpstr, err)
			} else {
				assert.ErrorContains(t, err, test.errorstr)
			}
		})
	}
}

// The rejections made when a report's repository cannot be established are
// checked separately, as the status of the error matters too.
func Test_reportRepoChecks(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()

	server, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)

	runReport := func(t *testing.T, tenant string, reportUrl string, userId string) error {
		session, err := NewModReportingSession(server, ts.URL, tenant, "dummyToken")
		assert.Nil(t, err)
		mock, err := pgxmock.NewPool()
		assert.Nil(t, err)
		defer mock.Close()
		session.dbConn = mock
		session.isMDB = true

		req := httptest.NewRequest("POST", ts.URL+"/ldp/db/reports", strings.NewReader(`{ "url": "`+reportUrl+`" }`))
		if userId != "" {
			req.Header.Set("X-Okapi-User-Id", userId)
		}
		w := httptest.NewRecorder()
		err = handleReport(w, req, session)
		// Rejection must happen before anything is sent to the database
		assert.Nil(t, mock.ExpectationsWereMet())
		return err
	}

	t.Run("report not in any repository", func(t *testing.T) {
		url := ts.URL + "/reports/loans.sql"
		err := runReport(t, "dummyTenant", url, "")
		assert.EqualError(t, err, "cannot run report "+url+": it is not in any configured report repository")
		var httpErr *HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusForbidden, httpErr.status)
	})

	t.Run("repositories cannot be loaded", func(t *testing.T) {
		url := ts.URL + "/api/v4/projects/test%2Freports/repository/files/reports%2Floans.sql/raw?ref=main"
		err := runReport(t, "noRepos", url, "")
		assert.ErrorContains(t, err, "cannot run report "+url+": could not load the configured report repositories from mod-settings: ")
		assert.ErrorContains(t, err, "mod-settings is unwell")
		var httpErr *HTTPError
		assert.False(t, errors.As(err, &httpErr), "should be reported as an internal server error")
	})

	// In the following tests, a report that is allowed to run fails at
	// the next step, as noheader.sql has no function declaration: this
	// shows that it got past the access check without needing the
	// database.
	restrictedUrl := ts.URL + "/api/v4/projects/test%2Frestricted/repository/files/reports%2Fnoheader.sql/raw?ref=main"
	openUrl := ts.URL + "/api/v4/projects/test%2Freports/repository/files/reports%2Fnoheader.sql/raw?ref=main"
	authorized := "22222222-2222-2222-2222-222222222222"
	unauthorized := "33333333-3333-3333-3333-333333333333"

	t.Run("open repository, no user", func(t *testing.T) {
		err := runReport(t, "dummyTenant", openUrl, "")
		assert.ErrorContains(t, err, "could not extract SQL function name")
	})

	t.Run("restricted repository, authorized user", func(t *testing.T) {
		err := runReport(t, "dummyTenant", restrictedUrl, authorized)
		assert.ErrorContains(t, err, "could not extract SQL function name")
	})

	t.Run("restricted repository, unauthorized user", func(t *testing.T) {
		err := runReport(t, "dummyTenant", restrictedUrl, unauthorized)
		assert.EqualError(t, err, "cannot run report "+restrictedUrl+": user "+unauthorized+
			" is not authorized to run reports from repository gitlab:"+ts.URL+" test/restricted@main:reports")
		var httpErr *HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusForbidden, httpErr.status)
	})

	t.Run("commit not on branch", func(t *testing.T) {
		savedApiUrl := githubApiUrl
		githubApiUrl = ts.URL
		t.Cleanup(func() { githubApiUrl = savedApiUrl })
		// The mock server says that commits starting with "c" are not on the branch
		sha := strings.Repeat("c", 40)
		url := "https://raw.githubusercontent.com/seneyl/test_reports/" + sha + "/circulation2/circ.sql"
		err := runReport(t, "dummyTenant", url, "11111111-1111-1111-1111-111111111111")
		assert.EqualError(t, err, "cannot run report "+url+": commit "+sha+
			" is not on the branch of any configured report repository that contains the report")
		var httpErr *HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusForbidden, httpErr.status)
	})

	t.Run("cannot check whether commit is on branch", func(t *testing.T) {
		savedApiUrl := githubApiUrl
		githubApiUrl = ts.URL
		t.Cleanup(func() { githubApiUrl = savedApiUrl })
		// The mock server refuses requests about commits starting with "f"
		sha := strings.Repeat("f", 40)
		url := "https://raw.githubusercontent.com/seneyl/test_reports/" + sha + "/circulation2/circ.sql"
		err := runReport(t, "dummyTenant", url, "11111111-1111-1111-1111-111111111111")
		assert.EqualError(t, err, "cannot run report "+url+": could not ask GitHub whether commit "+sha+
			" is on branch main of seneyl/test_reports: 403 Forbidden")
		var httpErr *HTTPError
		assert.False(t, errors.As(err, &httpErr), "should be reported as an internal server error")
	})

	t.Run("restricted repository, no user", func(t *testing.T) {
		err := runReport(t, "dummyTenant", restrictedUrl, "")
		assert.EqualError(t, err, "cannot run report "+restrictedUrl+": request has no X-Okapi-User-Id header, "+
			"so cannot check whether the user is authorized to run reports from repository gitlab:"+ts.URL+" test/restricted@main:reports")
		var httpErr *HTTPError
		assert.True(t, errors.As(err, &httpErr))
		assert.Equal(t, http.StatusForbidden, httpErr.status)
	})
}
