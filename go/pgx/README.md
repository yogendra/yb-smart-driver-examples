
## Steps

- Clone this repo since the changes are currently in this repo's branch.

  `git clone git@github.com:ashetkar/pgx.git -b load_balance`

- Clone this repository

  `git clone git@github.com:yugabyte/driver-examples.git -b pgx_load_balance`

- Move to this directory and run

  `go get github.com/yugabyte/pgx/v4`

- Append `replace github.com/yugabyte/pgx/v4 v4.14.1 => <path-to-above-pgx-clone>` to go.mod

- Run the example

  `go build ybsql_load_balance.go`
  `./ybsql_load_balance <path-to-ybdb-installation-dir>`
