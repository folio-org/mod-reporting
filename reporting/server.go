package reporting

import "os"
import "errors"
import "fmt"
import "net/http"
import "time"
import "strconv"
import "sync"
import "html"
import "github.com/MikeTaylor/catlogger"

type HTTPError struct {
	status  int
	message string
}

func (m *HTTPError) Error() string {
	return m.message
}

type handlerFn func(w http.ResponseWriter, req *http.Request, session *ModReportingSession) error

type ModReportingServer struct {
	config       *config
	logger       *catlogger.Logger
	root         string
	server       http.Server
	sessionTTL   time.Duration
	sessionMutex sync.Mutex
	sessions     map[string]*ModReportingSession
}

// FOLIO rotates access tokens every 10 minutes by default, and each new token
// yields a new session, so sessions must expire or they accumulate forever.
// The default is a little longer than the token lifespan, so that a session is
// never discarded while its token is still valid.
const defaultSessionTTL = 11 * time.Minute

func sessionTTL(logger *catlogger.Logger) time.Duration {
	s := os.Getenv("MOD_REPORTING_SESSION_TIMEOUT")
	if s == "" {
		return defaultSessionTTL
	}

	seconds, err := strconv.Atoi(s)
	if err != nil || seconds <= 0 {
		logger.Log("error", fmt.Sprintf("ignoring bad MOD_REPORTING_SESSION_TIMEOUT '%s'", s))
		return defaultSessionTTL
	}

	return time.Duration(seconds) * time.Second
}

func MakeModReportingServer(cfg *config, logger *catlogger.Logger, root string) *ModReportingServer {
	mux := http.NewServeMux()
	var server = ModReportingServer{
		config: cfg,
		logger: logger,
		root:   root,
		server: http.Server{
			// Set timeouts a minute longer than those at Postgres level to allow for overhead
			ReadTimeout:  time.Duration(cfg.QueryTimeout+60) * time.Second,
			WriteTimeout: time.Duration(cfg.QueryTimeout+60) * time.Second,
			Handler:      mux,
		},
		sessionTTL: sessionTTL(logger),
		sessions:   map[string]*ModReportingSession{},
	}

	// Every request is logged, whichever route it lands on.
	server.server.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		server.Log("path", req.URL.Path)
		mux.ServeHTTP(w, req)
	})

	mux.HandleFunc("/{$}", handleRoot)
	fs := http.FileServer(http.Dir(root + "/htdocs"))
	mux.Handle("/htdocs/", http.StripPrefix("/htdocs/", fs))
	mux.Handle("/favicon.ico", fs)
	mux.HandleFunc("/admin/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, "Behold! I live!!")
	})
	mux.HandleFunc("/ldp/config", server.handler(handleConfig))
	mux.HandleFunc("/ldp/config/", server.handler(handleConfigKey))
	mux.HandleFunc("/ldp/db/tables", server.handler(handleTables))
	mux.HandleFunc("/ldp/db/columns", server.handler(handleColumns))
	mux.HandleFunc("POST /ldp/db/query", server.handler(handleQuery))
	mux.HandleFunc("POST /ldp/db/reports", server.handler(handleReport))
	mux.HandleFunc("/ldp/db/log", server.handler(handleLogs))
	mux.HandleFunc("/ldp/db/version", server.handler(handleVersion))
	mux.HandleFunc("/ldp/db/updates", server.handler(handleUpdates))
	mux.HandleFunc("/ldp/db/processes", server.handler(handleProcesses))

	return &server
}

// Intended only for ModReportingSession to pass the session logger though to foliogo
func (server *ModReportingServer) GetLogger() *catlogger.Logger {
	return server.logger
}

func (server *ModReportingServer) Log(cat string, args ...string) {
	server.logger.Log(cat, args...)
}

func (server *ModReportingServer) Launch() error {
	cfg := server.config

	var port int
	serverPortString := os.Getenv("SERVER_PORT")
	if serverPortString != "" {
		port, _ = strconv.Atoi(serverPortString)
	} else {
		port = cfg.Listen.Port
	}

	hostspec := cfg.Listen.Host + ":" + fmt.Sprint(port)
	server.server.Addr = hostspec
	server.Log("listen", "listening on", hostspec)
	err := server.server.ListenAndServe()
	server.Log("listen", "finished listening on", hostspec)
	return err
}

// We maintain a map of tenant:url to session
func (server *ModReportingServer) findSession(url string, tenant string, token string) (*ModReportingSession, error) {
	key := sessionKey(url, tenant, token)

	server.sessionMutex.Lock()
	server.expireSessions()
	session := server.sessions[key]
	server.sessionMutex.Unlock()
	if session != nil {
		return session, nil
	}

	// The lock is not held across session creation, which makes network
	// calls: a slow FOLIO login must not block requests for other sessions.
	session, err := NewModReportingSession(server, url, tenant, token)
	if err != nil {
		return nil, fmt.Errorf("could not create session for key '%s': %w", key, err)
	}

	server.sessionMutex.Lock()
	defer server.sessionMutex.Unlock()
	existing := server.sessions[key]
	if existing != nil {
		// Another request created this session while we were building ours
		return existing, nil
	}

	session.created = time.Now()
	server.sessions[key] = session
	return session, nil
}

// Discards sessions older than the TTL, so that a later request builds a new
// one and re-reads the reporting-database configuration. Sessions are keyed on
// an access token, which expires anyway.
func (server *ModReportingServer) expireSessions() {
	deadline := time.Now().Add(-server.sessionTTL)
	for key, session := range server.sessions {
		if session.created.Before(deadline) {
			delete(server.sessions, key)
			// Closing blocks until in-flight queries end: not under the lock
			go session.close()
		}
	}
}

func handleRoot(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintln(w, `
This is <a href="https://github.com/folio-org/mod-reporting">mod-reporting</a>. Try:
<ul>
  <li><a href="/admin/health">Health check</a></li>
  <li><a href="/htdocs/">Static area</a></li>
  <li><a href="/ldp/config">Legacy configuration WSAPI</a></li>
  <li><a href="/ldp/config/dbinfo">Legacy configuration 'dbinfo'</a></li>
  <li><a href="/ldp/db/tables">List tables from reporting database</a></li>
  <li><a href="/ldp/db/columns?schema=folio_users&table=users">List columns for "users" table</a></li>
  <li><a href="/ldp/db/log">Logs</a></li>
  <li><a href="/ldp/db/version">Version</a></li>
  <li><a href="/ldp/db/updates">Updates</a></li>
  <li><a href="/ldp/db/processes">Processes</a></li>
</ul>`)
}

// handler adapts a handlerFn into an http.HandlerFunc for route registration.
func (server *ModReportingServer) handler(f handlerFn) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		runWithErrorHandling(w, req, server, f)
	}
}

func runWithErrorHandling(w http.ResponseWriter, req *http.Request, server *ModReportingServer, f handlerFn) {
	host := req.Header.Get("X-Okapi-Url")
	tenant := req.Header.Get("X-Okapi-Tenant")
	token := req.Header.Get("X-Okapi-Token")
	session, err := server.findSession(host, tenant, token)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "could not make session: %s\n", html.EscapeString(err.Error()))
		server.Log("error", fmt.Sprintf("%s: %s", req.RequestURI, err.Error()))
		return
	}

	err = f(w, req, session)
	if err != nil {
		status := http.StatusInternalServerError
		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.status
		}
		w.WriteHeader(status)
		fmt.Fprintln(w, html.EscapeString(err.Error()))
		session.Log("error", fmt.Sprintf("%s: %s", req.RequestURI, err.Error()))
	}
}
