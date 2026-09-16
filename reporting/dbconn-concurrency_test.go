package reporting

import "sync"
import "testing"
import "github.com/pashagolub/pgxmock/v3"
import "github.com/stretchr/testify/assert"

// A session is shared by every concurrent request for the same tenant, so
// findDbConn's lazy initialization -- test dbConn for nil, then assign it --
// runs in several goroutines at once. Run this under -race.
//
// Dummy makeConn stands in for makeDbConn, which would need a live Postgres.
func Test_findDbConnIsConcurrencySafe(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()

	server, err := MakeConfiguredServer("../etc/silent.json", ".")
	assert.Nil(t, err)
	session, err := NewModReportingSession(server, ts.URL, "dummyTenant", "dummyToken")
	assert.Nil(t, err)

	var made sync.Mutex
	var nmade int
	session.overrideMakeConn = func(token string) (PgxIface, bool, error) {
		made.Lock()
		defer made.Unlock()
		nmade++
		mock, err := pgxmock.NewPool()
		return mock, true, err
	}

	const nroutines = 50

	conns := make([]PgxIface, nroutines)
	errs := make([]error, nroutines)

	// The goroutines wait on this channel so that they all reach findDbConn at
	// once. Without such a barrier, the first goroutine finishes the whole
	// check-and-assign before later ones are even spawned, and the window in
	// which the bug shows is missed however many goroutines we start.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < nroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			conns[i], errs[i] = session.findDbConn("dummyToken")
		}(i)
	}
	close(start)
	wg.Wait()

	for i := 0; i < nroutines; i++ {
		assert.Nil(t, errs[i])
		assert.NotNil(t, conns[i])
		// Every request must see the one connection the session cached:
		// a second pool would be a leak, as nothing ever closes it.
		assert.Equal(t, conns[0], conns[i])
	}
	assert.Equal(t, 1, nmade)
}
