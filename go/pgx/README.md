
## Steps

- Clone this repo since the changes are currently in this repo's branch.

  `git clone git@github.com:ashetkar/pgx.git -b load_balance`

- Clone this repository

  `git clone git@github.com:yugabyte/driver-examples.git -b pgx_load_balance`

- Move to the directory `go/pgx` in this repo and setup the modules

  ```
  cd driver-examples/go/pgx
  go mod init main
  echo "replace github.com/yugabyte/pgx/v4 v4.14.1 => ../../../pgx" >> go.mod
  go get github.com/yugabyte/pgx/v4@v4.14.1
  go get github.com/yugabyte/pgx/v4/pgxpool@v4.14.1
  ```

- Build the example

  ```
  go build ybsql_load_balance.go ybsql_load_balance_pool.go
  ```

- Run the example

  To run the example which demonstrates `pgx.Connect()`, run

  ```
  ./ybsql_load_balance <path-to-ybdb-installation-dir>
  ```

  For interactive experience, run with `-i` option

  ```
  ./ybsql_load_balance <path-to-ybdb-installation-dir> -i
  ```

  To run the example which demonstrates `pgxpool.Connect()` interactively, run

  ```
  ./ybsql_load_balance <path-to-ybdb-installation-dir> --pool -i
  ```
