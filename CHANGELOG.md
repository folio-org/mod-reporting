# Change history for mod-reporting

## [1.5.0](https://github.com/folio-org/mod-reporting/tree/v1.5.0) (2025-11-25)

* Postgres query timeout is configurable both as `queryTimeout` in the config file and (overriding this) in the `MOD_REPORTING_QUERY_TIMEOUT` environment variable. Defaults to 60 seconds if neither is specified. Also, fixes a bug where reports running for more than 30 seconds would result in the connection to the client being silently dropped. Fixes MODREP-42.
* Upgrade to Go version 1.25 (specifically, v1.25.4). Fixes MODREP-44.

## [1.4.2](https://github.com/folio-org/mod-reporting/tree/v1.4.2) (2025-11-21)

* When writing an HTTP response fails, do not attempt to inform the client by writing the HTTP response again. Avoids "superfluous response.WriteHeader call" warning. Fixes MODREP-37.
* Upgrade to Go v1.24.6 to get patches for some vulnerabilities. Fixes MODREP-41.
* Redact passwords in log-file. Requires v0.1.8 of `foliogo` and v0.0.3 of `catlogger`. Fixes MODREP-40.
* Remove `LOGCAT` environment-variable setting from `Dockerfile`; add less verbose/more secure setting to launch descriptor. Fixes MODREP-43.
* Disable the report-URL whitelist in the example config. Fixes MODREP-46.
* Accept `limit` parameter (for JSON queries and reports) as either string or number. Fixes MODREP-45.
* Upgrade Go version to 1.24.10 to avoid vulnerabilities in earlier versions' libraries. Fixes MODREP-47.

## [1.4.1](https://github.com/folio-org/mod-reporting/tree/v1.4.1) (2025-06-03)

* URL-encode double quotes to `%22` sequences in CQL queries to mod-settings. Fixes MODREP-35.

## [1.4.0](https://github.com/folio-org/mod-reporting/tree/v1.4.0) (2025-05-16)

* Smaller base images in Dockerfile. Fixes MODREP-24.
* Upgrade golang from 1.23 to 1.24. Fixes MODREP-30.
* Support validation of report URLs by reference to a whitelist of regexps in the configuration file. Fixes MODREP-4.
* Politely reject invalid UUID values. Fixes MODREP-29.
* In JSON queries, ignore order-by elements when the fieldname is empty. Fixes MODREP-28.
* When returning a report or JSON query response, the fields of the results are ordered as specified in query. Fixes MODREP-31.
* Add `MikeTaylor`'s GitLab repositories to the default set of whitelist regexps. Fixes MODREP-32.

## [1.3.1](https://github.com/folio-org/mod-reporting/tree/v1.3.1) (2025-03-12)

* Prevent unescape-HTML attack in `server.go` error responses. Fixes MODREP-23.

## [1.3.0](https://github.com/folio-org/mod-reporting/tree/v1.3.0) (2025-03-05)

* Implement `/ldp/db/log`, add tests. This is in the module descriptor, so it's needed for full compatibility. Fixes MODREP-8.
* Upgrade dependency on `crypto` library (v0.27.0 has a vulnerability). Fixes MODREP-16.
* Add three new WSAPI endpoints (`/ldp/db/version`, `/ldp/db/updates`, `/ldp/db/processes`), write tests, update documentation. Provided interface `ldp-query` bumped from v1.3 to v1.4. Fixes MODREP-2.
* Each new FOLIO session gets a new reporting-database connection, causing the current DB config to be re-read. Fixes MODREP-11.
* Metadb-only features fail more politely (HTTP status 501) when run against LDP Classic. Fixes MODREP-17.
* When fetching `/ldp/config/dbinfo`, the reporting-database password is replaced by `********`. Fixes MODREP-18.
* Listening port can be set at runtime by the `SERVER_PORT` environment variable. Fixes MODREP-21.

## [1.2.0](https://github.com/folio-org/mod-reporting/tree/v1.2.0) (2024-10-29)

* Upgrade required Go version to current (from 1.21.3 to 1.23.2), to allow more up-to-date vulnerability checking. Fixes MODREP-13.
* Add `govulncheck` to `make lint` target, patch detected vulnerabilities. Fixes MODREP-14.
* Format all code according to `gofmt` standard, add Makefile rule. Fixes MODREP-15.

## [1.1.0](https://github.com/folio-org/mod-reporting/tree/v1.1.0) (2024-10-18)

* Modify permission names for the convenience of Eureka. Each endpoint is now governed my its own permission instead of `ldp.read` governing logs/tables/columns/query/reports. For backwards-compatibility, `ldp.read` is retained as an umbrella that contains all five new fine-grained permissions. Fixes MODREP-7.
* The `ldp-query` interface version number is incremented from 1.2 to 1.3. (A minor version bump suffices, as the current interface is backwards-compatible with the old.)
* `ldp-config-tool` now serializes object values to strings. This means if you now fetch an object-valued config from mod-ldp, you can set it directly into mod-reporting. Fixes MODREP-10.
* Bump default memory allocation from 20 Mb to 100 Mb to allow headroom for bigger query-result sets. Fixes MODREP-12.

## [1.0.4](https://github.com/folio-org/mod-reporting/tree/v1.0.4) (2024-09-17)

* Remove `env` section from the module-descriptor. This was causing FOLIO snapshot deployment to use a hardwired incorrect specification of the reporting database parameters.

## [1.0.3](https://github.com/folio-org/mod-reporting/tree/v1.0.3) (2024-09-16)

* Change launch-descriptor `portBindings` to match the port exposed by `Dockerfile`. Fixes DEVOPS-3304.

## [1.0.2](https://github.com/folio-org/mod-reporting/tree/v1.0.2) (2024-09-14)

* Re-release to exercise updated GitHub actions. No code changes.

## [1.0.1](https://github.com/folio-org/mod-reporting/tree/v1.0.1) (2024-09-13)

* Changes to GitHub CI workflows. Supports deployment of mod-reporting to folio-snapshot reference environments. Part of DEVOPS-3304.

## [1.0.0](https://github.com/folio-org/mod-reporting/tree/v1.0.0) (2024-08-27)

* First release.


