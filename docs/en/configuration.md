# Configuration

PostgreBase is configured via CLI flags passed to the `serve` command.

## CLI Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--dataDsn` | Database connection string | `sqlite://pb_data/data.db` |
| `--redisDsn` | (Optional) Redis connection string. Falls back to in-memory cache if omitted | `redis://localhost:6379/0` |
| `--dir` | Data directory for file uploads, backups, etc. | `./pb_data` |
| `--encryptionEnv` | Environment variable name for settings encryption key | `PB_ENCRYPTION` |
| `--debug` | Enable debug mode with detailed logs | _(flag)_ |

## DSN Formats

### PostgreSQL

```
postgres://user:pass@host:port/dbname?sslmode=disable
postgresql://user:pass@host:port/dbname?sslmode=disable
```

### MySQL

```
mysql://user:pass@tcp(host:port)/dbname
```

### SQLite

```
sqlite:///path/to/file.db
sqlite3:///path/to/file.db
./path/to/file.db          # shorthand (detected by .db suffix)
```

## Driver Detection

The DSN prefix determines which database driver is used:

| Prefix | Driver |
|--------|--------|
| `postgres://` | PostgreSQL |
| `postgresql://` | PostgreSQL |
| `mysql://` | MySQL |
| `sqlite://` | SQLite |
| `sqlite3://` | SQLite |
| `*.db` (suffix) | SQLite |
| _(no prefix)_ | PostgreSQL (default) |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PB_ENCRYPTION` | Encryption key for settings (used by `--encryptionEnv`) |

## Examples

### Local Development

```bash
./pb serve --dataDsn "sqlite://./pb_data/dev.db" --debug
```

### Production with PostgreSQL

```bash
./pb serve \
  --dataDsn "postgres://app:secret@db.example.com:5432/postgrebase?sslmode=disable" \
  --dir /var/data/postgrebase
```

### Production with MySQL + Redis

```bash
./pb serve \
  --dataDsn "mysql://app:secret@tcp(db.example.com:3306)/postgrebase" \
  --redisDsn "redis://redis.example.com:6379/0" \
  --dir /var/data/postgrebase
```

### HA Cluster Node

```bash
./pb serve \
  --dataDsn "postgres://app:secret@pg-cluster:5432/postgrebase?sslmode=disable" \
  --redisDsn "redis://redis-cluster:6379/0"
```
