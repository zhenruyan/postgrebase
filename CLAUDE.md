# PocketBase (PostgreSQL & MySQL Fork)

## Overview
This project is a modified version (fork) of [PocketBase](https://pocketbase.io). The primary difference is that **SQLite has been replaced with PostgreSQL and MySQL**. The author created this fork to allow PocketBase to handle high-pressure and complex clustered environments that SQLite struggles with. 

By default, the project connects to a **PostgreSQL** database via standard Data Source Names (DSN), but **MySQL** is also fully supported through a `mysql://` DSN prefix.

## Tech Stack
- **Language:** Go (1.18+)
- **Database Driver:** `github.com/lib/pq` (PostgreSQL), `github.com/go-sql-driver/mysql` (MySQL) / local `dbx`
- **Core Architecture:** Follows the original PocketBase `core`, `daos`, `models`, `apis` architecture with modifications made for postgres/mysql compliance.

## Project Structure
- `build/`: Contains the `main.go` entrypoint for the server (in the upstream this was `examples/base/`).
- `core/`: Contains the core application logic. `db_postgresql.go` holds the new `postgres` connection logic.
- `pocketbase.go`: Modified to accept `--dataDsn` instead of the local SQLite database directory.
- `pb_data/`: Still exists but does not contain `.db` files anymore.

## Setup & Local Development

### 1. Database Setup
By default, the application expects a local PostgreSQL instance. You can also use MySQL.

For PostgreSQL:
```sql
CREATE DATABASE postgres;
```

For MySQL:
```sql
CREATE DATABASE pb_data;
```

### 2. Build the Server
The `README.md` references an outdated `examples/base` path. The actual entry point is in the `build/` directory.

```bash
# Compile the binary
go build -o pb ./build/
```

### 3. Run the Server
You can run the application with the default connection string (which points to local PostgreSQL `127.0.0.1:5432` without SSL):

```bash
./pb serve
```

To run with a custom PostgreSQL DSN:
```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/postgres?sslmode=disable"
```

To run with a custom MySQL DSN:
```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/pb_data"
```

## Key Code Differences from Upstream PocketBase
1. **No SQLite:** The `github.com/mattn/go-sqlite3` driver is gone.
2. **Database Drivers:** Uses `github.com/lib/pq` and `github.com/go-sql-driver/mysql`.
3. **DSN Initialization:** `core/db_postgresql.go` parses the prefix (`postgres://` vs `mysql://`) to initialize `dbx.Open(driver, dsn)`.
4. **Command Line Flags:** Added `--dataDsn` to replace local database file configurations.

## Maintenance Notes
- If modifying schemas or queries, ensure compatibility with both PostgreSQL and MySQL. 
- Schema field definitions and `init` migrations use conditional dialect checks via `dao.DB().DriverName()`.
- The Makefile only includes a `lint` target (`golangci-lint`).
- `.goreleaser.yaml` is configured for deployment, but currently points to a missing `examples/base/` main path. Ensure you adjust it to `./build` if releasing.
