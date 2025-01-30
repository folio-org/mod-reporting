# Change history for mod-reporting

## [1.3.0](https://github.com/folio-org/mod-reporting/tree/v1.3.0) (IN PROGRESS)

* Implement `/ldp/db/log`, add tests. This is in the module descriptor, so it's needed for full compatibility. Fixes MODREP-8.
* Upgrade dependency on `crypto` library (v0.27.0 has a vulnerability). Fixes MODREP-16.
* Add three new WSAPI endpoints (`/ldp/db/version`, `/ldp/db/updates`, `/ldp/db/processes`), write tests, update documentation. Provided interface `ldp-query` bumped from v1.3 to v1.4. Fixes MODREP-2.
* Each new FOLIO session gets a new reporting-database connection, causing the current DB config to be re-read. Fixes MODREP-11.

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


