# The module's executable is built from the top level: see ../Makefile

# After upgrading Go, external linting tools also need upgrading:
#	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
#	go install honnef.co/go/tools/cmd/staticcheck@latest
#	go install github.com/kisielk/errcheck@latest
#	go install golang.org/x/tools/cmd/deadcode@latest
#	go install golang.org/x/vuln/cmd/govulncheck@latest
#
lint:
	-go vet ./...
	-go vet -vettool=/Users/mike/go/bin/shadow ./...
	-! egrep -n '([ 	]+$$|if +\(|;[ 	]*$$)' *.go | grep -v ':[A-Z][A-Z][A-Z][A-Z]'
	-staticcheck ./... | (grep -v '^/usr/local/go/src/runtime/' || true)
	-errcheck -exclude .errcheck-exclude ./...
	-ineffassign ./...
	-deadcode ./...
	-cd ..; golangci-lint run
	-govulncheck ./...

test:
	go test -race -v -coverprofile=coverage.out ./...
	go test -json -coverprofile=coverage.out ./... > coverage.json
	@echo "go tool cover -func=coverage.out | sed 's/^github.com\/folio-org\/mod-reporting\/reporting\///'"

test1:
	go test -v -coverprofile=coverage.out . -run Test_makeSql

cover: coverage.out
	go tool cover -html=coverage.out

fmt:
	go fmt ./...

clean:
	rm -f coverage.out coverage.json
