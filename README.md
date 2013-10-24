track
========================================

[![Build Status](https://travis-ci.org/sourcegraph/track.png?branch=master)](https://travis-ci.org/sourcegraph/track)
[![status](https://sourcegraph.com/api/repos/github.com/sourcegraph/track/badges/status.png)](https://sourcegraph.com/github.com/sourcegraph/track)
[![xrefs](https://sourcegraph.com/api/repos/github.com/sourcegraph/track/badges/xrefs.png)](https://sourcegraph.com/github.com/sourcegraph/track)
[![funcs](https://sourcegraph.com/api/repos/github.com/sourcegraph/track/badges/funcs.png)](https://sourcegraph.com/github.com/sourcegraph/track)
[![top func](https://sourcegraph.com/api/repos/github.com/sourcegraph/track/badges/top-func.png)](https://sourcegraph.com/github.com/sourcegraph/track)
[![library users](https://sourcegraph.com/api/repos/github.com/sourcegraph/track/badges/library-users.png)](https://sourcegraph.com/github.com/sourcegraph/track)

Track tracks user actions and API calls in Web applications that use
[Go](http://golang.org) and [AngularJS](http://angularjs.org/) (with [Angular UI
Router](https://github.com/angular-ui/ui-router)).


Running tests
-------------

1. Create the test database schema. Set the environment vars `PGHOST`, `PGUSER`,
   `PGDATABASE`, etc., so that running `psql` alone opens a DB prompt.
1. Run `go test -test.initdb`

After running the tests with `-test.initdb`, you can omit the flag on future
runs. To drop the DB schema, run `go test -test.dropdb`.
