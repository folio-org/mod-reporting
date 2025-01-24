The FOLIO Reporting API provides simple mediated access to a reporting database (LDP Classic or MetaDB) hosted elsewhere. It provides eight entry points, each of them very simple:

1. `/ldp/db/tables`: Request a list of all the tables in their various schemas
2. `/ldp/db/columns`: Request a list of all the columns in a specified table. (The schema and table names are povided as URL query parameters)
3. `/ldp/db/query`: Submit a query
4. `/ldp/db/reports`: Run a report from a repository
5. `/ldp/config` and `/ldp/config/{key}`: Simple key/value configuration store
6. `/ldp/db/version`: Returns the version of Metadb that the module is connected to
7. `/ldp/db/updates`: Shows the time of the most recent updates to tables
8. `/ldp/db/processes`: Gives information on long-running searches

Several types are defined to support these operations:
* The first operation returns [`tables`](tables-schema.json), a list of table-and-schema-name pairs.
* The second operation returns [`columns`](columns-schema.json), a list of column definitions including information such as the column name and type.
* The third operation accepts a [`query`](query-schema.json), a set of parameters such as the table to search in, the criteria, and the columns to return. It returns [`results`](results-schema.json), a list of objects representing rows that satisfy the query, each containing the specified set of columns.
* The fourth operation accepts a [`template query`](template-query-schema.json), specifying where to find the report and what values to substituted into its parameters. It returns [`template results`](template-results-schema.json), a list of result objects together with a result count.
* The fifth operation deals with [`config`](configuration.json) objects and [lists thereof](configuration-list.json)
* The sixth operation returns a simple [`version`](version-schema.json) object.
* The seventh operation returns a [list of table update times](updates-schema.json).
* The eighth operation returns a [list of processes](processes-schema.json).

