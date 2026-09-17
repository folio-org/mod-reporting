# See also: ramls/Makefile (used only for validation and documentation)

SRC=main.go $(wildcard reporting/*.go)
TARGET=target/mod-reporting

**default**: target/ModuleDescriptor.json $(TARGET)

target/ModuleDescriptor.json:
	(cd target; make)

$(TARGET): $(SRC)
	go build -o $@ .

run: $(TARGET)
	env LOGCAT=listen,path,db,error,sql,op,curl,sql,status,response $(TARGET) etc/config.json

run-local: $(TARGET)
	env LOGCAT=listen,path,db,error,sql,op,curl,status,response OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org OKAPI_TENANT=diku OKAPI_USER=diku_admin $(TARGET) etc/config.json

lint:
	(cd reporting; make lint)

# The central FOLIO CI workflow uploads the coverage reports from the
# hardcoded path src/coverage.* (see folio-org/.github go-build.yml), so
# copy them there until that workflow can be told where to look.
#
test:
	(cd reporting; make test)
	mkdir -p src && cp reporting/coverage.out reporting/coverage.json src/

clean:
	(cd target; make clean)
	(cd reporting; make clean)
	rm -rf src
