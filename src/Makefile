SRC=main.go configured-server.go config-file.go getdbinfo.go http-error.go server.go session.go ldp-config.go reporting.go
TESTSRC=config-file_test.go ldp-config_test.go mod-reporting_test.go
TARGET=../target/mod-reporting

$(TARGET): $(SRC)
	go build -o $@

run: $(TARGET)
	cd ..; env LOGCAT=listen,path,db,error,sql,op,curl,sql,status,response src/$(TARGET) etc/config.json

run-local: $(TARGET)
	cd ..; env LOGCAT=listen,path,db,error,sql,op,curl,status,response OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org OKAPI_TENANT=diku OKAPI_USER=diku_admin src/$(TARGET) etc/config.json

lint:
	-go vet ./...
	-go vet -vettool=/Users/mike/go/bin/shadow ./...
	-! egrep -n '([ 	]+$$|if +\(|;[ 	]*$$)' *.go | grep -v ':[A-Z][A-Z][A-Z][A-Z]'
	-staticcheck ./... | (grep -v '^/usr/local/go/src/runtime/' || true)
	-errcheck -exclude .errcheck-exclude ./...
	-ineffassign ./...
	-deadcode
	-govulncheck ./...

test:
	go test -v -coverprofile=coverage.out ./...
	go test -json -coverprofile=coverage.out ./... > coverage.json
	@echo "go tool cover -func=coverage.out | sed 's/^github.com\/folio-org\/mod-reporting\/src\///'"

test1:
	go test -v -coverprofile=coverage.out . -run Test_reporting

cover: coverage.out
	go tool cover -html=coverage.out

clean:
	rm -f $(TARGET) coverage.out coverage.json

