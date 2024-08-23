package main

import "context"
import "strings"
import "fmt"
import "github.com/indexdata/foliogo"
import "github.com/jackc/pgx/v5"
import "github.com/jackc/pgx/v5/pgxpool"
import "github.com/jackc/pgx/v5/pgconn" // just for the data-type pgconn.CommandTag


type PgxIface interface {
	Begin(context.Context) (pgx.Tx, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Close()
}

type ModReportingSession struct {
	server *ModReportingServer // back-reference
	url string
	tenant string
	folioSession foliogo.Session
	dbConn PgxIface
	isMDB bool
}


/*
 * There are two valid cases:
 * 1. The url and tenant parameters are defined: we have received a request from Okapi, and the new FOLIO session should be pointed to the specified URL+tenant
 * 2. The url and tenant parameters are not defined: we have a request directly from a browser, curl command or similar, and need ot make a default FOLIO session (which will use environment variables such as OKAPI_URL)
 *
 * There is also one common INvalid case:
 * 3. The url parameter is not defined but the tenant parameter is. This arises when running a curl command copied from a browser session, as the url parameter is added only by Okapi. We explicitly catch this are reject it.
 */

func NewModReportingSession(server *ModReportingServer, url string, tenant string) (*ModReportingSession, error) {
	if url == "" && tenant != "" {
		return nil, fmt.Errorf("no URL provided with tenant: responding to a request with no X-Okapi-Url header?")
	}

	session := ModReportingSession{
		server: server,
		url: url,
		tenant: tenant,
	}

	if url != "" {
		// A request that has arrived via Okapi (or been faked to look that way)
		service := foliogo.NewService(url, server.GetLogger())
		folioSession, err := service.ResumeSession(tenant)
		session.folioSession = folioSession
		if err != nil {
			return nil, fmt.Errorf("could not resume existing FOLIO session: %w", err)
		}

		return &session, nil
	}

	// Probably a request from command-line, not via Okapi
	// In this case, we use a FOLIO service specified in the environment
	folioSession, err := foliogo.NewDefaultSession(server.GetLogger())
	if err != nil {
		return nil, fmt.Errorf("could not create new FOLIO session: %w", err)
	}

	session.folioSession = folioSession
	return &session, nil
}


func (session *ModReportingSession)Log(cat string, args ...string) {
	session.server.Log(cat, args...)
}


func (session *ModReportingSession)makeDbConn(token string) (PgxIface, bool, error) {
	dbUrl, dbUser, dbPass, err := getDbInfo(session.folioSession, token)
	if err != nil {
		return nil, false, fmt.Errorf("cannot extract data from 'dbinfo': %w", err)
	}
	session.Log("db", "url=" + dbUrl + ", user=" + dbUser)

	// For historical reasons, database connection configuration is often JDBCish
	dbUrl = strings.Replace(dbUrl, "jdbc:postgresql://", "", 1)
	dbUrl = strings.Replace(dbUrl, "postgres://", "", 1)
	// We may need `?sslmode=require` on the end of the URL.
	dbConn, err := pgxpool.New(context.Background(), "postgres://" + dbUser + ":" + dbPass + "@" + dbUrl)
	if err != nil {
		return nil, false, fmt.Errorf("cannot connect to DB: %w", err)
	}

	session.Log("db", "connected to DB", dbUrl)
	isMDB, err := isMetaDB(dbConn)
	if err != nil {
		return nil, false, fmt.Errorf("cannot determine whether reporting DB is MetaDB: %w", err)
	}

	session.Log("db", fmt.Sprintf("isMetaDB=%v", isMDB))
	return dbConn, isMDB, nil
}


func (session *ModReportingSession) findDbConn(token string) (PgxIface, error) {
	if session.dbConn == nil {
		dbConn, isMDB, err := session.makeDbConn(token)
		if err != nil {
			return nil, err
		}
		session.dbConn = dbConn
		session.isMDB = isMDB
	}

	return session.dbConn, nil
}
