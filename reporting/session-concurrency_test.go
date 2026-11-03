package reporting

import "fmt"
import "sync"
import "testing"
import "github.com/stretchr/testify/assert"

// findSession runs in the goroutine that net/http spawns for each request, so
// concurrent requests for keys that are not yet cached read and write the
// session map at the same time. Run this under -race to see the problem; it can
// also abort the whole test binary with "fatal error: concurrent map writes".
func Test_findSessionIsConcurrencySafe(t *testing.T) {
	ts := MakeMockHTTPServer()
	defer ts.Close()

	cfg, err := readConfig("../etc/silent.json")
	assert.Nil(t, err)
	server := MakeModReportingServer(cfg, nil, "")

	const nroutines = 50
	const ntenants = nroutines / 2

	sessions := make([]*ModReportingSession, nroutines)
	errs := make([]error, nroutines)

	var wg sync.WaitGroup
	for i := 0; i < nroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each tenant is requested by two nroutines, so that both the
			// concurrent-insert and the concurrent-lookup paths are exercised.
			tenant := fmt.Sprintf("tenant%d", i%ntenants)
			sessions[i], errs[i] = server.findSession(ts.URL, tenant, "dummyToken")
		}(i)
	}
	wg.Wait()

	for i := 0; i < nroutines; i++ {
		assert.Nil(t, errs[i])
		assert.NotNil(t, sessions[i])
	}
	assert.Equal(t, ntenants, len(server.sessions))
}
