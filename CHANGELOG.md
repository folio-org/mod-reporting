# Change history for mod-reporting

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


