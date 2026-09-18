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

test:
	(cd reporting; make test)

clean:
	(cd target; make clean)
	(cd reporting; make clean)
