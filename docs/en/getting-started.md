# Quick Start

This guide helps you get PostgreBase running in 5 minutes.

## Prerequisites

- **Go 1.26.2+** (for building from source)
- **PostgreSQL** or **MySQL** (optional — SQLite works out of the box)

## Installation

### Method 1: Build from Source

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

For static builds (no CGO):

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### Method 2: Download Binary

Download the latest binary from [GitHub Releases](https://github.com/zhenruyan/postgrebase/releases).

## Running the Server

### SQLite (Local Development)

```bash
# Using sqlite:// prefix
./pb serve --dataDsn "sqlite://./pb_data/dev.db"

# Or just pass a .db file path
./pb serve --dataDsn "./pb_data/dev.db"
```

### PostgreSQL (Recommended for Production)

```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

### MySQL

```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

### With Redis Cache

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://127.0.0.1:6379/0"
```

### Docker

Build the image from the included Dockerfile:

```bash
docker build -t postgrebase .
```

Run with SQLite:

```bash
docker run -p 8090:8090 -v pb_data:/pb/pb_data postgrebase serve --dataDsn "sqlite:///pb/pb_data/dev.db"
```

Run with PostgreSQL:

```bash
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://user:pass@host:5432/dbname?sslmode=disable"
```

Run with Redis:

```bash
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://..." --redisDsn "redis://host:6379/0"
```

The Docker image uses Alpine Linux, exposes port 8090, and stores data in `/pb/pb_data`.

## First Steps

1. Open the Admin UI at `http://127.0.0.1:8090/_/`
2. Create your admin account
3. Create a collection (table) with fields
4. Use the REST API or MCP to interact with your data

## Next Steps

- [Configuration](configuration.md) — All CLI flags and DSN formats
- [MCP Protocol](mcp.md) — Connect AI tools to your data
- [High Availability](high-availability.md) — Deploy in a cluster
