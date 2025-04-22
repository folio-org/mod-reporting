# mod-reporting

Copyright (C) 2023-2025 The Open Library Foundation

This software is distributed under the terms of the Apache License, Version 2.0. See the file "[LICENSE](LICENSE)" for more information.

<!-- md2toc -l 2 README.md -->
* [Overview](#overview)
* [Compilation and installation](#compilation-and-installation)
* [Configuration](#configuration)
    * [Configuration file](#configuration-file)
    * [Logging](#logging)
    * [FOLIO services and reporting databases](#folio-services-and-reporting-databases)
* [Notes](#notes)
    * [Duplicate permission](#duplicate-permission)
    * [Redundant field in API](#redundant-field-in-api)
    * [CORS problems when running locally](#cors-problems-when-running-locally)
* [See also](#see-also)
* [Additional information](#additional-information)
    * [Issue tracker](#issue-tracker)
    * [Code of Conduct](#code-of-conduct)
    * [ModuleDescriptor](#moduledescriptor)
    * [API documentation](#api-documentation)
    * [Code analysis](#code-analysis)
    * [Download and configuration](#download-and-configuration)
* [Author](#author)



## Overview

`mod-reporting` is a FOLIO module that mediates access to reporting databases -- instances of either MetaDB or LDP Classic. It removes the need to deal directly with a relational database by providing a simple WSAPI that can be used by UI code such as [ui-ldp](https://github.com/folio-org/ui-ldp).

`mod-reporting` started out as a plug-compatible replacement for [`mod-ldp`](https://github.com/folio-org/mod-ldp), using the same API specification ([FOLIO module descriptor](descriptors/ModuleDescriptor-template.json), [RAML file](ramls/ldp.raml) and associated JSON Schemas and examples). It provides the same interface (`ldp-query`) with the same semantics. In more recent releases, it adds new endpoints that were never supported by mod-ldp, but remains strictly backwards compatible.

As well as the machine-readable API specification, [a human-readable overview](ramls/overview.md) is provided.

**Personal data.**
This module does not store any personal data. See the file [`PERSONAL_DATA_DISCLOSURE.md`](PERSONAL_DATA_DISCLOSURE.md) for details.

**Contributing.**
See the file [`CONTRIBUTING.md`](CONTRIBUTING.md) for details.



## Compilation and installation

`mod-reporting` is written in Go.

Compilation is controlled by a good old-fashioned [`Makefile`](Makefile)
* `make target/ModuleDescriptor.json` to compile the module descriptor
* `make target/mod-reporting` to build to module
* `make test` to run tests
* `make lint` to audit code quality and security

Invocation takes a single argument, the name of a configuration file (see [below](#configuration-file)). Its behaviour can also be affected by environment variables (see [below](#folio-services-and-reporting-databases)).

Containerization is supported:
* `docker build -t mod-reporting .` to create a container
* `docker run -p 12369:12369 mod-reporting` to run the container with its default port wired out to the host

GitHub workflows exist to build the code and run tests, and to create a Docker container.



## Configuration

### Configuration file

The configuration file is written in JSON. An example can be found in [`etc/config.json`](etc/config.json):

```
{
  "logging": {
    "categories": "config,listen,path,error",
    "prefix": "",
    "timestamp": false
  },
  "listen": {
    "host": "0.0.0.0",
    "port": 12369
  },
  "reportUrlWhitelist": [
    "^https://gitlab.com/MikeTaylor/metadb-queries/",
    "^https://raw.githubusercontent.com/metadb-project/"
  ]
}
```

Three top-level stanzas are supported:
* `logging` specifies how the system's [categorical logger](https://github.com/MikeTaylor/catlogger) should be configured:
  * `categories` is a comma-separated list of logging categories for which output should be emitted: see [below](#logging)
  * `prefix` is an optional string which will be emitted at the start of each logging line. This can help to differentiate logging output from other outputs.
  * `timestamp` is a boolean indicating whether each logged line should be timestamped.
* `listen` specifies where the running server should listen for connections:
  * `host` is an IP address or DNS-resolvable hostname. `0.0.0.0` (all interfaces) should usually be used
  * `port` is an IP port number
* `reportUrlWhitelist` is an optional list of regular expressions. If this is specified, then only report URLs that match one of these regular expressions are accepted.

The port specified in the `listen` stanza can be overridden at run-time by setting the `SERVER_PORT` environment variable. This is useful when invoking the service from a container whose contents (i.e. the configuration file) cannot easily be modified, but whose environment can be specified.


### Logging

The following categories of logging information may be emitted, depending on how the logger is configured:

* `config` -- logs the contents of the configuration file
* `listen` -- indicates when the server has started listening, and on what host and port
* `path` -- notes each path requested by a client
* `db` -- emits information about each reporting database and notes when successful connections are made
* `sql` -- logs the generated SQL for each JSON query submitted via the `/ldp/db/query` endpoint
* `validate` -- logs checks of report URLs against the specified whitelist regular expressions
* `error` -- emits error messages returned to the client in HTTP responses

Access to the FOLIO database is performed using [the foliogo client library](https://github.com/indexdata/foliogo) which also uses categorical logger. See its documentation for information on the categories `service`, `session`, `op`, `auth`, `curl`, `status` and `response`.


### FOLIO services and reporting databases

In normal operation, each incoming request is serviced by reference to the Okapi instance that sent it. For development, however, it's possible to override this behaviour and have every outgoing request go to a nominated Okapi instance. This is specified by the environment variables `OKAPI_URL` (e.g https://folio-snapshot-okapi.dev.folio.org) and `OKAPI_TENANT` (e.g. `diku`). When using an Okapi service specified in this way, authentication onto this instance is done using the values specifid by the environment variables `OKAPI_USER` and `OKAPI_PW`.

Similarly, in normal operation, mod-reporting determines which underlying reporting database to connect to on the basis of the information configured in FOLIO -- specifically, in the mod-settings record with scope `ui-ldp.config` and key `dbinfo`, as managed by the "Database configuration" settings page of the Reporting app. However, these configured settings can be overridden if mod-reporting is run with all three of the following environment variables set:

* `REPORTING_DB_URL` -- The full URL of the PostgreSQL database to connect to, e.g. `postgres://localhost:5432/metadb`
* `REPORTING_DB_USER` -- The name of the PostgreSQL user to act as when accessing this database
* `REPORTING_DB_PASS` -- The password to use for nominated user



## Notes


### Duplicate permission

For reasons discussed in [MODREP-25](https://folio-org.atlassian.net/browse/MODREP-25), [the module descriptor](descriptors/ModuleDescriptor-template.json) contains a permission -- `mod-settings.global.write.ui-ldp.admin` -- that is also defined, identically, in [ui-ldp](https://github.com/folio-org/ui-ldp). In time, the ui-ldp version will be removed, but until then **be careful not to make changes to this permission**.

### Redundant field in API

In the response from `/ldp/db/reports`, there is a numeric element `totalRecords`. Note that this is a count of the number of records included in the `records` array -- _not_ the total number of hits in the database. (That information is not available from PostgreSQL). The provided field is redundant, and would have been better omitted, but we retain it for backwards compatibility.


### CORS problems when running locally

If running `mod-reporting` locally, you will likely run into CORS problems with Stripes refusing to make GET and POST requests to it because OPTIONS requests don't return the necessary `Access-control-allow-origin` header. To work around this, you can run a CORS-permissive HTTP proxy such as [`local-cors-anywhere`](https://github.com/dkaoster/local-cors-anywhere) -- which by default listens on port 8080 -- and access the running `mod-reporting` at http://localhost:8080/http://localhost:12369.



## See also

* [The `ldp-config-tool` utility](config-tool)
* The old [`mod-ldp`](https://github.com/folio-org/mod-ldp) for which this is a replacement
* FOLIO's Reporting app, [`ui-ldp`](https://github.com/folio-org/ui-ldp), which uses this module
* [The FOLIO Module Acceptance assessment](doc/MODULE_EVALUATION_TEMPLATE.MD)

## Additional information

### Issue tracker

See project [MODREP](https://issues.folio.org/browse/MODREP)
at the [FOLIO issue tracker](https://dev.folio.org/guidelines/issue-tracker).

### Code of Conduct

Refer to the Wiki
[FOLIO Code of Conduct](https://wiki.folio.org/display/COMMUNITY/FOLIO+Code+of+Conduct).

### ModuleDescriptor

See the [ModuleDescriptor](descriptors/ModuleDescriptor-template.json)
for the interfaces that this module requires and provides, the permissions,
and the additional module metadata.

### API documentation

API descriptions:

 * [RAML](ramls/ldp.raml) and the included [Overview](ramls/overview.md).
 * [JSON Schemas](ramls/)

Generated [API documentation](https://dev.folio.org/reference/api/#mod-reporting).

### Code analysis

[SonarCloud analysis](https://sonarcloud.io/project/overview?id=org.folio:mod-reporting).

### Download and configuration

The built artifacts for this module are available.
See [configuration](https://dev.folio.org/download/artifacts) for repository access,
and the Docker images for [released versions](https://hub.docker.com/r/folioorg/mod-reporting/)
and for [snapshot versions](https://hub.docker.com/r/folioci/mod-reporting/).

## Author

Mike Taylor, [Index Data ApS](https://www.indexdata.com/).
mike@indexdata.com

