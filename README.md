<p align="center">
    <a href="https://pocketbase.io" target="_blank" rel="noopener">
        <img src="https://i.imgur.com/5qimnm5.png" alt="PocketBase - open source backend in 1 file" />
    </a>
</p>

<p align="center">
    <a href="https://github.com/pocketbase/pocketbase/actions/workflows/release.yaml" target="_blank" rel="noopener"><img src="https://github.com/pocketbase/pocketbase/actions/workflows/release.yaml/badge.svg" alt="build" /></a>
    <a href="https://github.com/pocketbase/pocketbase/releases" target="_blank" rel="noopener"><img src="https://img.shields.io/github/release/pocketbase/pocketbase.svg" alt="Latest releases" /></a>
    <a href="https://pkg.go.dev/github.com/pocketbase/pocketbase" target="_blank" rel="noopener"><img src="https://godoc.org/github.com/ganigeorgiev/fexpr?status.svg" alt="Go package documentation" /></a>
</p>

[PocketBase](https://pocketbase.io) is an open source Go backend, consisting of:

The author of PB is really excellent, and PB is a great work.

But we really need to run PB online in high pressure and complex environments. Sqlite cannot handle this scenario, so I am working hard to make it support cockroachdb  and postgreSQL for cluster and standalone environments.

Current modification progress, running cockroachdb  by default as a single node by default

Then run PB to install and perform curd operations normally.

run cockroachdb 

```bash

cockroach start-single-node --insecure

cockroach sql --insecure

CREATE DATABASE logs;
CREATE DATABASE data;

```

build  pb

```bash

go build .\examples\base\

```

run pb --help

```
PocketBase CLI

Usage:
  base.exe [command]

Available Commands:
  admin       Manages admin accounts
  migrate     Executes app DB migration scripts
  serve       Starts the web server (default to 127.0.0.1:8090)
  update      Automatically updates the current PocketBase executable with the latest available version

Flags:
      --automigrate            enable/disable auto migrations (default true)
      --dataDsn string         store data postgresql dsn(default  postgresql://root@127.0.0.1:26257/data?sslmode=disable) (default "postgresql://root@127.0.0.1:26257/data?sslmode=disable")
      --debug                  enable debug mode, aka. showing more detailed logs
      --dir string             the PocketBase data directory (default "D:\\src\\postgresqlbaseapi\\pb_data")
      --encryptionEnv string   the env variable whose value of 32 characters will be used
                               as encryption key for the app settings (default none)
  -h, --help                   help for base.exe
      --hooksDir string        the directory with the JS app hooks
      --hooksPool int          the total prewarm goja.Runtime instances for the JS app hooks execution (default 50)
      --hooksWatch             auto restart the app on pb_hooks file change (default true)
      --indexFallback          fallback the request to index.html on missing static path (eg. when pretty urls are used with SPA) (default true)
      --logDsn string          store logs postgresql dsn(default postgresql://root@127.0.0.1:26257/logs?sslmode=disable) (default "postgresql://root@127.0.0.1:26257/logs?sslmode=disable")
      --migrationsDir string   the directory with the user defined migrations
      --publicDir string       the directory to serve static files (default "D:\\src\\postgresqlbaseapi\\pb_public")
      --queryTimeout int       the default SELECT queries timeout in seconds (default 30)
  -v, --version                version for base.exe

Use "base.exe [command] --help" for more information about a command.

```


release build

```
goreleaser.exe release --skip-publish  --snapshot  --rm-dist
```