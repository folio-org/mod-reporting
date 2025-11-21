# ldp-config-tool.js - read from, or write to, the /ldp/config endpoint

<!-- md2toc -l 2 README.md -->
* [Background](#background)
* [Building](#building)
* [Running](#running)
    * [Setting directly into mod-settings](#setting-directly-into-mod-settings)


## Background

The new module mod-reporting provides exactly the same WSAPI as the old mod-ldp, described by the `ldp` interface in [the module descriptor](../descriptors/ModuleDescriptor-template.json).

However, mod-ldp's use of the FOLIO database violated conventions in using the "public" schema not enforcing tenant separation. That bug is fixed in mod-reporting -- instead of making its own database accesses, it does configuration using mod-settings. But as a result, old configuration elements are no longer visible after switching from mod-ldp to mod-reporting.

This tool exists to allow the old values to be read from the configuration store before upgrading, and to write them back afterwards.


## Building

`ldp-config-tool.js` written in Node, using [the FolioJS library](https://github.com/indexdata/foliojs). All dependencies are captured by the package file in this directory, so to build it you just need to run `yarn install`.

This software was developed using Node v18.17.1 and Yarn v1.22.19, but there is no reason to think that earlier version won't work just fine. We're not doing anything unusual here that would need bleeding-edge versions.


## Running

The FOLIO system and user to be used by the `ldp-config-tool.js` script are specified by the standard environment variables
`OKAPI_URL`,
`OKAPI_TENANT`,
`OKAPI_USER`,
and
`OKAPI_PW`.

You will need to run `ldp-config-tool.js` once in read mode, to extract the configuration from the old mod-ldp; then perform the upgrade; then run `ldp-config-tool.js` again in write mode, to put the old configuration values back:
```
$ export OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org
$ export OKAPI_TENANT=diku
$ export OKAPI_USER=diku_admin
$ export OKAPI_PW=********
$ ldp-config-tool.js get > config-dump.json
```
Then disable mod-ldp for your tenant, and enable mod-reporting. Then:
```
$ ldp-config-tool.js set < config-dump.json
```

### Setting directly into mod-settings

Sometimes in practice it's necessary to copy mod-ldp configuration into mod-settings before mod-reporting is available to write through. So there is also a new `write-settings` mode for this script which will insert directly into mod-settings.


<!--
env OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org OKAPI_TENANT=diku OKAPI_USER=diku_admin OKAPI_PW= LOGCAT=auth,op,status ./l
dp-config-tool.js get > config-dump.json
env OKAPI_URL=https://folio-snapshot-okapi.dev.folio.org OKAPI_TENANT=diku OKAPI_USER=diku_admin OKAPI_PW= LOGCAT=auth,op,status ./ldp-config-tool.js write-settings < config-dump.json
-->
