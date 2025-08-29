# Build with: docker build -t mod-reporting .
# Run with: docker run -p 12369:12369 -e OKAPI_PW=[DIKU_ADMIN-PASSWORD] mod-reporting

# also update the version in go.mod
FROM golang:1.24-alpine AS build

# Install latest patch versions of alpine packages: https://pythonspeed.com/articles/security-updates-in-docker/
RUN apk upgrade --no-cache

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY src etc htdocs ./

# Build
RUN CGO_ENABLED=0 go build -o mod-reporting ./...

# https://github.com/GoogleContainerTools/distroless/tree/main/base
FROM gcr.io/distroless/base:nonroot

COPY --from=build /app/mod-reporting /app/config.json ./

EXPOSE 12369

# Run
ENV LOGCAT=listen,op,curl,status,response,db,path,error

# These are not needed in production, as requests will come from Okapi with authentication tokens
#ENV OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org
#ENV OKAPI_TENANT=diku
#ENV OKAPI_USER=diku_admin
#ENV OKAPI_PW=swordfish

# These are also not needed in production, as reporting-DB details will be configured in /ldp/config
#ENV REPORTING_DB_URL=postgres://id-test-metadb.folio.indexdata.com:5432/metadb_indexdata_test
#ENV REPORTING_DB_USER=miketaylor
#ENV REPORTING_DB_PASS=swordfish

ENTRYPOINT ["./mod-reporting"]
CMD ["config.json"]
