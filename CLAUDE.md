# PocketBase (Enterprise-Ready PostgreSQL & MySQL Fork)

## Overview
This project is a high-performance, enterprise-ready fork of [PocketBase](https://pocketbase.io). The core storage engine has been replaced with **PostgreSQL** and **MySQL** to handle high-concurrency, clustered, and complex data environments that SQLite cannot support. It features a hybrid caching mechanism that supports both **Redis** (for clusters) and **In-Memory** (for standalone) caching.

## Tech Stack
- **Language:** Go (1.18+)
- **Database Driver:** `github.com/lib/pq` (PostgreSQL), `github.com/go-sql-driver/mysql` (MySQL)
- **Caching:** `github.com/redis/go-redis/v9` (Redis) / Built-in memory store
- **Architecture:** Modified PocketBase `core`, `daos`, `models`, `apis` for SQL dialect compatibility and hybrid caching.

## Project Structure
- `build/`: Main server entry point (`main.go`).
- `core/`: Application logic. `db_postgresql.go` handles DSN parsing; `base.go` contains hybrid cache initialization.
- `vendor/`: **Must be preserved.** Contains all dependencies for offline/strict environment builds.
- `pocketbase.go`: Main package file; handles command-line flags like `--dataDsn` and `--redisDsn`.

## Setup & Local Development

### 1. Database Setup
- **PostgreSQL:** `CREATE DATABASE postgres;`
- **MySQL:** `CREATE DATABASE pb_data;`

### 2. Build the Server
```bash
go build -o pb ./build/
```

### 3. Run the Server
- **PostgreSQL (Default):** `./pb serve --dataDsn "postgres://user:pass@127.0.0.1:5432/db?sslmode=disable"`
- **MySQL:** `./pb serve --dataDsn "mysql://user:pass@tcp(127.0.0.1:3306)/db"`
- **With Redis:** `./pb serve --redisDsn "redis://localhost:6379/0"` (Falls back to Memory Cache if omitted).

## Key Implementation Details
- **Hybrid Caching:** If `--redisDsn` is empty, `core.BaseApp` uses an internal memory store. If provided, it initializes a Redis client.
- **Preserved Vendor:** The `/vendor` folder is critical and must not be deleted.
- **DSN Prefixing:** Uses `mysql://` vs `postgres://` or `postgresql://` to switch between drivers in `core/db_postgresql.go`.

## Maintenance Notes
- **Linter:** `golangci-lint` (config in `golangci.yml`).
- **CI/CD:** GitHub Actions in `.github/workflows/` (Build Check and GoReleaser).
- **Versioning:** Version is set via `ldflags` in `.goreleaser.yaml`.
