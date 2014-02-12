appmon
========================================

[![Build Status](https://travis-ci.org/sourcegraph/appmon.png?branch=master)](https://travis-ci.org/sourcegraph/appmon)
[![status](https://sourcegraph.com/api/repos/github.com/sourcegraph/appmon/badges/status.png)](https://sourcegraph.com/github.com/sourcegraph/appmon)
[![xrefs](https://sourcegraph.com/api/repos/github.com/sourcegraph/appmon/badges/xrefs.png)](https://sourcegraph.com/github.com/sourcegraph/appmon)
[![funcs](https://sourcegraph.com/api/repos/github.com/sourcegraph/appmon/badges/funcs.png)](https://sourcegraph.com/github.com/sourcegraph/appmon)
[![top func](https://sourcegraph.com/api/repos/github.com/sourcegraph/appmon/badges/top-func.png)](https://sourcegraph.com/github.com/sourcegraph/appmon)
[![library users](https://sourcegraph.com/api/repos/github.com/sourcegraph/appmon/badges/library-users.png)](https://sourcegraph.com/github.com/sourcegraph/appmon)

Appmon tracks API calls in Web applications that use [Go](http://golang.org).


Running tests
-------------

1. Create the test database schema. Set the environment vars `PGHOST`, `PGUSER`,
   `PGDATABASE`, etc., so that running `psql` alone opens a DB prompt.
1. Run `go test -test.initschema` to create the DB schema.

After running the tests with `-test.initschema`, you can omit the flag on future
runs. To drop the DB schema, run `go test -test.dropschema`.
