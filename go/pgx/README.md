
## Pre-requisites

- Make sure you have YugabyteDB installed on the local file system.

## Steps

- Clone this repository

  `git clone git@github.com:yugabyte/driver-examples.git`

- Move to the directory `go/pgx` in this repo and setup the modules

  ```
  cd driver-examples/go/pgx
  go mod init main
  go mod tidy
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
