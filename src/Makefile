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
	-! egrep -n '([ 	]+$$|if +\(|;[ 	]*$$)' *.go 
	-staticcheck ./...
	-errcheck -exclude .errcheck-exclude ./...
	-ineffassign ./...
	-deadcode

test:
	go test -v -coverprofile=c.out ./...
	@echo "go tool cover -func=c.out | sed 's/^github.com\/folio-org\/mod-reporting\/src\///'"

test1:
	go test -v -coverprofile=c.out . -run Test_reporting

cover: c.out
	go tool cover -html=c.out

clean:
	rm -f $(TARGET) c.out

