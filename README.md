# PocketBase for PostgreSQL & MySQL (Enterprise Ready)

This is a heavily customized fork of [PocketBase](https://pocketbase.io), re-engineered for enterprise-grade production environments. The core storage engine has been replaced with **PostgreSQL** and **MySQL** to provide higher performance, scalability, and clustering capabilities.

By moving away from SQLite's write-locking limitations, this project is designed for high-concurrency, multi-node clusters, and complex data environments.

## Core Features

- **Enterprise Performance:** Overcomes SQLite write-locks by leveraging PostgreSQL/MySQL's robust concurrency models.
- **Cluster-Ready Architecture:** Native support for multi-instance deployments. Since it doesn't rely on local database files, you can easily run multiple instances behind a load balancer.
- **Dual Database Engines:**
    - **PostgreSQL:** Default engine with standard DSN support.
    - **MySQL:** Fully compatible via the `mysql://` DSN prefix.
- **Flexible Caching:**
    - **Redis Cache:** Enables distributed caching for cluster environments via `--redisDsn`. When enabled, **SSE Realtime subscriptions also use Redis Pub/Sub**, ensuring message synchronization across all nodes.
    - **In-Memory Cache:** Automatically falls back to high-performance local memory caching if Redis is not configured, ensuring lightning-fast responses for standalone setups.
- **Maintain PocketBase Experience:** 100% compatible with existing PocketBase APIs, Admin UI, and business logic.

## Quick Start

### 1. Prerequisites

- Go 1.26.2+
- PostgreSQL or MySQL instance

### 2. Build

```bash
git clone https://github.com/free/postgresqlbaseapi.git
cd postgresqlbaseapi
# Build binary
go build -o pb ./build/
```

### 3. Run

By default, the application tries to connect to PostgreSQL at `127.0.0.1:5432`.

#### Using PostgreSQL (Recommended for Production)
```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

#### Using MySQL
```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

#### Enable Redis Cache (Enhanced Cluster Performance)
```bash
./pb serve --redisDsn "redis://127.0.0.1:6379/0"
```

## Configuration Flags

- `--dataDsn`: Database connection string.
    - PostgreSQL: `postgres://user:pass@host:port/db?sslmode=disable`
    - MySQL: `mysql://user:pass@tcp(host:port)/db`
- `--redisDsn`: (Optional) Redis connection string. **Defaults to high-performance local memory cache if not provided.**
- `--dir`: Data directory (used for file uploads, backups, etc., but not for the main database).
- `--encryptionEnv`: Name of the environment variable for settings encryption.

## Development

### Building Admin UI

If you modify the Admin UI, rebuild the embedded assets:

```bash
cd ui
npm install
npm run build
```

### Contributing

1. Fork the repository.
2. Create your feature branch (`git checkout -b feature/amazing-feature`).
3. Commit your changes (`git commit -m 'Add some amazing feature'`).
4. Push to the branch (`git push origin feature/amazing-feature`).
5. Open a Pull Request.

## Credits

This project is a fork of [PocketBase](https://pocketbase.io). Special thanks to [Gani Georgiev](https://github.com/ganigeorgiev) for the original amazing work.

## License

Licensed under the [MIT license](https://opensource.org/licenses/MIT).
