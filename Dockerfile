# Build with: docker build -t mod-reporting .
# Run with: docker run -p 12369:12369 -e OKAPI_PW=[DIKU_ADMIN-PASSWORD] mod-reporting

FROM golang:1.21

ENV APP_DIR=/app

# Create user/group 'folio'
RUN addgroup folio && \
    adduser --disabled-password --gecos "" --home ${APP_DIR} --ingroup folio folio && \
    chown -R folio:folio ${APP_DIR}

# Run as this user
USER folio

# Set destination for COPY
WORKDIR ${APP_DIR}

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy sources
COPY src etc htdocs ./

# Build
RUN go build -o mod-reporting ./...

EXPOSE 12369

# Run
ENV LOGCAT=listen,op,curl,status,response,db,path

# These are not needed in production, as requests will come from Okapi with authentication tokens
#ENV OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org
#ENV OKAPI_TENANT=diku
#ENV OKAPI_USER=diku_admin
#ENV OKAPI_PW=swordfish

# These are also not needed in production, as reporting-DB details will be configured in /ldp/config
#ENV REPORTING_DB_URL=postgres://id-test-metadb.folio.indexdata.com:5432/metadb_indexdata_test
#ENV REPORTING_DB_USER=miketaylor
#ENV REPORTING_DB_PASS=swordfish

CMD ["./mod-reporting", "config.json"]
