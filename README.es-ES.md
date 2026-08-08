

# PostgreBase — Herramienta de desarrollo de API sin código nativa de IA

PostgreBase es una plataforma de desarrollo de API sin código nativa de IA, reescrita sobre [PocketBase](https://pocketbase.io). Incluye un servidor MCP (Model Context Protocol) que permite a las herramientas de IA (Claude, Cursor, Windsurf) interactuar directamente con tus datos. Soporta PostgreSQL, MySQL y SQLite, proporcionando APIs REST instantáneas, suscripciones en tiempo real y una interfaz de administración (Admin UI), permitiendo poner en marcha tu servicio backend en 5 minutos.

## Características

- **Tres Motores de Base de Datos:**
    - **PostgreSQL** (predeterminado) — concurrencia de nivel producción, clustering y consultas complejas.
    - **MySQL** — totalmente compatible mediante el prefijo DSN `mysql://`.
    - **SQLite** (Go puro, sin CGO) — desarrollo y pruebas locales sin dependencias mediante el DSN `sqlite://` o la ruta del archivo `.db`.
- **Arquitectura lista para clústeres:** La falta de dependencia de archivos de base de datos locales permite que múltiples instancias funcionen detrás de un balanceador de carga con un backend compartido de PostgreSQL/MySQL.
- **Caché Híbrido:**
    - **Caché Redis** (`--redisDsn`) — caché distribuido para entornos de clúster; las suscripciones SSE en tiempo real se sincronizan entre nodos mediante Redis Pub/Sub.
    - **Caché en Memoria** — retorno automático a la caché de memoria local de alto rendimiento cuando Redis no está configurado.
- **Servidor MCP (Model Context Protocol):**
    - Protocolo JSON-RPC 2.0 que permite a las herramientas de IA (Claude Desktop, Cursor, Windsurf) interactuar con tus datos.
    - 8 herramientas integradas: `list_collections`, `get_collection`, `list_records`, `get_record`, `create_record`, `update_record`, `delete_record`, `search_records`.
    - Recursos: `postgrebase://collections`, `postgrebase://settings`.
    - Tres modos de transporte: SSE (HTTP), HTTP transmisible (Streamable HTTP) y Stdio.
    - Tokens API específicos para MCP con soporte de expiración, gestionables desde la Admin UI.
- **Cero Dependencias Externas para SQLite:** Utiliza `modernc.org/sqlite` (transpilación de SQLite en Go puro), compila con `CGO_ENABLED=0`.

## Inicio Rápido

### Prerequisitos

- Go 1.26.2+
- PostgreSQL, MySQL o ninguno (SQLite funciona de forma nativa)

### Compilación

```bash
git clone https://github.com/zhenruyan/postgrebase.git
cd postgrebase
go build -o pb ./build/
```

Para compilaciones estáticas (sin CGO):

```bash
CGO_ENABLED=0 go build -o pb ./build/
```

### Ejecución

#### SQLite (Desarrollo Local)

```bash
# Using sqlite:// prefix
./pb serve --dataDsn "sqlite://./pb_data/dev.db"

# Or just pass a .db file path
./pb serve --dataDsn "./pb_data/dev.db"
```

#### PostgreSQL (Recomendado para Producción)

```bash
./pb serve --dataDsn "postgresql://user:password@127.0.0.1:5432/dbname?sslmode=disable"
```

#### MySQL

```bash
./pb serve --dataDsn "mysql://user:password@tcp(127.0.0.1:3306)/dbname"
```

#### Con Caché Redis

```bash
./pb serve --dataDsn "postgres://..." --redisDsn "redis://127.0.0.1:6379/0"
```

#### Docker

```bash
# Build the image
docker build -t postgrebase .

# Run with SQLite
docker run -p 8090:8090 -v pb_data:/pb/pb_data postgrebase serve --dataDsn "sqlite:///pb/pb_data/dev.db"

# Run with PostgreSQL
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://user:pass@host:5432/db?sslmode=disable"

# Run with Redis
docker run -p 8090:8090 postgrebase serve --dataDsn "postgres://..." --redisDsn "redis://host:6379/0"
```

Al iniciar, el servidor muestra sus puntos de conexión:

```
├─ REST API: http://127.0.0.1:8090/api/
├─ MCP SSE:  http://127.0.0.1:8090/api/mcp/sse
└─ Admin UI: http://127.0.0.1:8090/_/
```

## Parámetros de Configuración

| Parámetro | Descripción | Ejemplo |
|------|-------------|---------|
| `--dataDsn` | Cadena de conexión a la base de datos | `sqlite://pb_data/data.db` |
| `--redisDsn` | (Opcional) Cadena de conexión a Redis. Si se omite, utiliza caché en memoria. | `redis://localhost:6379/0` |
| `--dir` | Directorio de datos para subidas de archivos, copias de seguridad, etc. | `./pb_data` |
| `--encryptionEnv` | Nombre de la variable de entorno para la clave de cifrado de la configuración | `PB_ENCRYPTION` |
| `--debug` | Activar modo de depuración con registros detallados | _(parámetro)_ |

### Formatos DSN

| Motor | Formato |
|--------|--------|
| PostgreSQL | `postgres://user:pass@host:port/db?sslmode=disable` o `postgresql://...` |
| MySQL | `mysql://user:pass@tcp(host:port)/db` |
| SQLite | `sqlite:///path/to/file.db`, `sqlite3://...`, o simplemente `./file.db` |

## MCP (Model Context Protocol)

PostgreBase incluye un servidor MCP integrado que permite a las herramientas de IA interactuar con tus datos a través de un protocolo JSON-RPC 2.0 estandarizado.

### Modos de Transporte

#### SSE (Eventos Enviados por el Servidor) — HTTP

Disponible al ejecutar `./pb serve`. Conecta a:

```
GET  http://localhost:8090/api/mcp/sse      # SSE event stream
POST http://localhost:8090/api/mcp/message   # Send JSON-RPC requests
POST http://localhost:8090/api/mcp/stream    # Streamable HTTP (single request/response)
```

#### Stdio — CLI

Ejecuta el servidor MCP como un proceso independiente que se comunica por stdin/stdout:

```bash
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-token "YOUR_TOKEN"

# Or disable auth (NOT recommended for production)
./pb mcp --dataDsn "sqlite://./dev.db" --mcp-no-auth
```

| Parámetro | Descripción |
|------|-------------|
| `--mcp-token` | Token JWT de administrador o token específico de MCP para autenticación |
| `--mcp-no-auth` | Deshabilitar autenticación (solo para desarrollo) |

### Configuración de Claude Desktop

```json
{
  "mcpServers": {
    "postgrebase": {
      "command": "/path/to/pb",
      "args": ["mcp", "--dataDsn", "sqlite:///path/to/dev.db", "--mcp-no-auth"]
    }
  }
}
```

### Tokens MCP

Para entornos de producción, crea tokens MCP dedicados (con prefijo `mcp_`) desde la Admin UI en **Ajustes → Tokens MCP**. Estos tokens:

- Son independientes de los tokens JWT de administrador.
- Pueden revocarse individualmente.
- Soportan fechas de expiración opcionales.
- Se muestran completamente solo una vez al momento de su creación.

### Herramientas Disponibles

| Herramienta | Descripción |
|------|-------------|
| `list_collections` | Lista todas las colecciones |
| `get_collection` | Obtiene el esquema y la configuración de una colección |
| `list_records` | Lista registros con paginación, filtrado y ordenación |
| `get_record` | Obtiene un solo registro por ID |
| `create_record` | Crea un nuevo registro |
| `update_record` | Actualiza un registro existente |
| `delete_record` | Elimina un registro |
| `search_records` | Busca registros utilizando expresiones de filtro de PostgreBase |

### Recursos Disponibles

| URI | Descripción |
|-----|-------------|
| `postgrebase://collections` | Todas las colecciones con sus esquemas |
| `postgrebase://settings` | Configuración de la aplicación (sanitizada, sin secretos) |

## Desarrollo

### Compilación de la Admin UI

Si modificas la Admin UI, recompila los activos integrados antes de compilar el binario Go:

```bash
cd ui
npm install
npm run build
cd ..
go build -o pb ./build/
```

### Estructura del Proyecto

```
postgrebase/
├── build/           # Main server entry point (main.go)
├── core/            # Application logic, DB connection, caching
├── daos/            # Data access objects (CRUD operations)
├── models/          # Data models and schema definitions
├── apis/            # HTTP API handlers (REST + MCP routes)
├── mcp/             # MCP server: protocol, tools, resources, transports
├── migrations/      # Database migrations (Postgres/MySQL/SQLite)
├── cmd/             # CLI commands (serve, mcp, admin)
├── dbx/             # Database query builder (fork of ozzo-dbx)
├── tools/           # Shared utilities (security, types, search, etc.)
├── ui/              # Admin UI (Svelte + Vite)
├── vendor/          # Go dependencies (preserved for offline builds)
└── postgrebase.go  # Root package: CLI setup, config, bootstrap
```

### Ejecución de Pruebas

```bash
go test ./tools/... ./models/... ./daos/...
```

### Contribuir

1. Haz un fork del repositorio.
2. Crea tu rama de características (`git checkout -b feature/amazing-feature`).
3. Confirma tus cambios (`git commit -m 'Add some amazing feature'`).
4. Sube a la rama (`git push origin feature/amazing-feature`).
5. Abre una Pull Request.

## Créditos

Este proyecto es un fork de [PocketBase](https://pocketbase.io). Un agradecimiento especial a [Gani Georgiev](https://github.com/ganigeorgiev) por el trabajo original increíble.

## Licencia

Licenciado bajo la [licencia MIT](https://opensource.org/licenses/MIT).
