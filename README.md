# Gin API Template

A production-ready, feature-based Go REST API starter built with **Gin**, **PostgreSQL** (`pgx/v5`), **sqlc** (type-safe queries), and **goose** (migrations). It includes cookie-based session authentication with Google OAuth support, role-based authorization, structured logging, graceful shutdown, and seamless Docker / Podman support.

---

## Table of Contents

- [Features](#features)
- [Project Layout](#project-layout)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
  - [1. Clone & Rename Module](#1-clone--rename-module)
  - [2. Configure Environment](#2-configure-environment)
  - [3a. Run with Containers (Recommended)](#3a-run-with-containers-recommended)
  - [3b. Run Locally on Host](#3b-run-locally-on-host)
- [API Conventions](#api-conventions)
- [API Usage & Examples](#api-usage--examples)
  - [Health Probes](#health-probes)
  - [User Registration & Authentication](#user-registration--authentication)
  - [Protected & Admin Endpoints](#protected--admin-endpoints)
  - [Session Management](#session-management)
- [How to Add a New Feature](#how-to-add-a-new-feature)
- [Adding New API Versions](#adding-new-api-versions)
- [Authentication & Authorization](#authentication--authorization)
- [Database & Migration Workflow](#database--migration-workflow)
- [Environment Configuration](#environment-configuration)
- [Makefile Reference](#makefile-reference)
- [Docker & Podman Notes](#docker--podman-notes)
- [Production Notes & Best Practices](#production-notes--best-practices)
- [License](#license)

---

## Features

- **Feature-Based Architecture**: Business logic is divided into feature slices (`internal/features/*`), each managing its own handlers, services, DTOs, and SQL queries.
- **Unified Type-Safe Queries (`sqlc`)**: Generates type-safe Go database code directly into `internal/platform/dbgen` with Go interfaces enabled for straightforward unit testing and mocking.
- **Database Migrations (`goose`)**: Schema defined in SQL migrations (`db/migrations/`), serving as the schema source of truth for `sqlc`.
- **Authentication & Sessions**:
  - Secure password hashing using `bcrypt`.
  - HTTP-only cookie-based session management stored in PostgreSQL (`user_sessions`).
  - Google OAuth ID token verification and extensible third-party provider registry.
  - Multi-device session tracking and per-session revocation or global sign-out.
  - Automatic background worker cleaning up expired sessions every 24 hours.
- **Role-Based Authorization (RBAC)**: Context-aware middleware for authentication (`Authenticate`) and administrator privileges (`RequireAdmin`).
- **Structured Logging (`log/slog`)**: Pretty text logs for local development, structured JSON for production, with contextual `request_id` correlation.
- **Production Middleware**: Request ID generation/propagation (`X-Request-ID`), structured request logger, panic recovery, security headers, and CORS.
- **Standardized JSON Envelope**: Consistent envelopes for data (`data`, `meta`), errors (`error.code`, `error.message`, `error.details`), and validation errors mapped to JSON property names.
- **Operational Probes**: Unversioned `/healthz` (liveness) and `/readyz` (readiness with database ping), plus a compiled container health check (`api healthcheck`).
- **Container-First**:
  - Multi-stage `Dockerfile` with minimal distroless runtime and unprivileged user.
  - `compose.yaml` with database health check, automatic Goose migration container, the API server, and **Adminer** database management UI (`http://localhost:8088`).
  - Native support for both **Docker** and **Podman**.
- **Development Tooling**: Hot-reload via `air`, `golangci-lint`, race-enabled testing, and automated Makefile targets.

---

## Project Layout

```
cmd/
  api/
    main.go                   # Application entrypoint & container healthcheck CLI
db/
  migrations/                 # Goose SQL migrations (source of truth for DB schema)
internal/
  config/                     # Environment configuration loader & validation
  features/                   # Feature-based business modules
    auth/                     # Session & OAuth authentication
      auth.go                 # DTOs, interfaces, and types
      handler.go              # HTTP handlers for login/logout/sessions
      service.go              # Session lifecycle, password & token logic
      provider.go             # OAuth provider registry & Google token verifier
      query.sql               # sqlc SQL queries for sessions & credentials
      route.go                # Route registration (/api/v1/auth/*)
    health/                   # Operational health probes (/healthz, /readyz)
      health.go               # Handlers & router registration
    user/                     # User management feature
      user.go                 # User DTOs and models
      handler.go              # HTTP handlers for user CRUD
      service.go              # User business logic
      query.sql               # sqlc SQL queries for users
      route.go                # Route registration (/api/v1/users/*)
  platform/                   # Shared technical infrastructure (no business logic)
    database/                 # pgxpool PostgreSQL connection pool initializer
    dbgen/                    # sqlc-generated code (db, models, querier interface)
    httpx/                    # Response envelopes, error wrappers, validator helpers
    logger/                   # slog logger initialization
    middleware/               # RequestID, logging, recovery, security headers, auth & RBAC
  server/
    server.go                 # http.Server lifecycle, graceful shutdown, session cleaner
    router.go                 # Middleware pipeline, dependency injection & route mounting
  version/                    # Build-time version and git commit metadata
scripts/
  rename.sh                   # Module renaming utility
.air.toml                     # Air hot reload configuration
.env.example                  # Environment variable template
compose.yaml                  # Multi-container stack (db, migrate, api, adminer)
Dockerfile                    # Multi-stage container build
Makefile                      # Development and orchestration commands
sqlc.yaml                     # sqlc compilation configuration
```

### Dependency Flow

```
HTTP Request → Router → Middleware → Feature Handler → Feature Service → dbgen.Querier (PostgreSQL)
```

- Features **never** import other features directly. Shared utilities, middleware, and common models reside in `internal/platform/`.
- Features define their persistence queries in local `query.sql` files, which `sqlc` compiles into the shared `internal/platform/dbgen` package.

---

## Prerequisites

- **Go**: 1.23 or newer
- **Make**: GNU Make
- **Container Engine**: Docker (`docker` + `docker compose`) or Podman (`podman` + `podman compose`)
- *(Optional for local host development)*: `goose`, `sqlc`, `air`, `golangci-lint` (can be installed with `make tools`)

---

## Getting Started

### 1. Clone & Rename Module

After cloning or creating your repository from this template, run the rename script to update the Go module path across all files:

```bash
# Replace with your module path (e.g., github.com/username/my-service)
./scripts/rename.sh github.com/myuser/my-api

# Update dependencies and lockfile
go mod tidy
```

### 2. Configure Environment

Copy the example environment file and customize it if needed:

```bash
cp .env.example .env
```

The defaults work out-of-the-box for local containerized development.

### 3a. Run with Containers (Recommended)

Spin up the entire stack (PostgreSQL, database migrations, the API, and Adminer):

```bash
# With Docker:
make up CONTAINER=docker COMPOSE="docker compose"

# With Podman (default):
make up
```

Services started:
- **API**: `http://localhost:8080`
- **PostgreSQL**: `localhost:5432`
- **Adminer DB UI**: `http://localhost:8088` (Server: `db`, Username: `app`, Password: `app`, Database: `app`)

To view live logs:
```bash
make logs
```

To stop the containers:
```bash
make down
```

### 3b. Run Locally on Host

If you prefer running the Go application on your host with hot reload:

1. **Install required development tools**:
   ```bash
   make tools
   ```
   *(Installs `goose`, `sqlc`, `air`, and `golangci-lint` to `$GOPATH/bin`)*

2. **Start the PostgreSQL container**:
   ```bash
   make db-up
   ```

3. **Apply database migrations**:
   ```bash
   make migrate-up
   ```

4. **Start the API server**:
   ```bash
   # With hot reload (recommended):
   make dev

   # ...or standard go run:
   make run
   ```

---

## API Conventions

### Success Envelope

Single resource:
```json
{
  "data": {
    "id": "e3b0c442-98fc-1c14-9afb-4c759dd18417",
    "email": "jane@example.com",
    "name": "Jane Doe"
  }
}
```

Paginated resource:
```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42
  }
}
```

### Error Envelope

```json
{
  "error": {
    "code": "validation_error",
    "message": "Request validation failed",
    "details": {
      "email": "failed 'email' validation"
    },
    "request_id": "8a83d78c-687f-4318-bd88-0f1352e825a1"
  }
}
```

### Common Error Codes

| Status Code | Code | Description |
|---|---|---|
| 400 | `invalid_request`, `invalid_id` | Malformed JSON body or invalid UUID param |
| 401 | `unauthorized`, `invalid_credentials` | Missing/invalid session or incorrect credentials |
| 403 | `forbidden`, `email_not_verified` | Insufficient permissions (e.g. admin required) |
| 404 | `not_found`, `user_not_found` | Resource or route does not exist |
| 405 | `method_not_allowed` | HTTP method not supported for route |
| 409 | `email_taken` | Unique constraint conflict (e.g. email exists) |
| 422 | `validation_error` | Request body validation failed |
| 500 | `internal_error` | Server error (internal details logged, hidden from client) |

Clients can supply an `X-Request-ID` header; it will be echoed in responses and attached to all log lines. If omitted, the server generates a UUID automatically.

---

## API Usage & Examples

A ready-to-run `.http` file is provided in the project root for testing with VS Code REST Client or JetBrains HTTP Client. Below are standard `curl` examples:

### Health Probes

```bash
# Liveness probe (returns 200 OK)
curl http://localhost:8080/healthz

# Readiness probe (pings database, returns 200 OK when ready)
curl http://localhost:8080/readyz
```

### User Registration & Authentication

1. **Register a new user**:
   ```bash
   curl -i -X POST http://localhost:8080/api/v1/users \
     -H "Content-Type: application/json" \
     -d '{
       "email": "user@example.com",
       "name": "Alex Doe",
       "password": "Password123!"
     }'
   ```

2. **Login with email and password**:
   Store the session cookie in `cookies.txt`:
   ```bash
   curl -i -X POST http://localhost:8080/api/v1/auth/login \
     -c cookies.txt \
     -H "Content-Type: application/json" \
     -d '{
       "email": "user@example.com",
       "password": "Password123!"
     }'
   ```
   *The server responds with an HTTP-only session cookie (default name: `sid`).*

3. **Get current user profile and session info**:
   ```bash
   curl -b cookies.txt http://localhost:8080/api/v1/auth/me
   ```

4. **Login with Google OAuth**:
   Send the Google ID token:
   ```bash
   curl -i -X POST http://localhost:8080/api/v1/auth/google \
     -c cookies.txt \
     -H "Content-Type: application/json" \
     -d '{
       "token": "<google_id_token>"
     }'
   ```

### Protected & Admin Endpoints

- **Get user profile by ID** (Any authenticated user):
  ```bash
  curl -b cookies.txt http://localhost:8080/api/v1/users/<user_uuid>
  ```

- **List users** (Admin only):
  ```bash
  curl -b cookies.txt "http://localhost:8080/api/v1/users?page=1&per_page=20"
  ```

- **Update user** (Admin only):
  ```bash
  curl -i -X PATCH http://localhost:8080/api/v1/users/<user_uuid> \
    -b cookies.txt \
    -H "Content-Type: application/json" \
    -d '{
      "name": "Alex Smith"
    }'
  ```

- **Delete user** (Admin only):
  ```bash
  curl -i -X DELETE http://localhost:8080/api/v1/users/<user_uuid> \
    -b cookies.txt
  ```

### Session Management

- **List active sessions**:
  ```bash
  curl -b cookies.txt http://localhost:8080/api/v1/auth/sessions
  ```

- **Revoke a specific session**:
  ```bash
  curl -i -X DELETE http://localhost:8080/api/v1/auth/sessions/<session_uuid> \
    -b cookies.txt
  ```

- **Log out of current session**:
  ```bash
  curl -i -X POST http://localhost:8080/api/v1/auth/logout \
    -b cookies.txt
  ```

- **Log out of all devices**:
  ```bash
  curl -i -X POST http://localhost:8080/api/v1/auth/logout-all \
    -b cookies.txt
  ```

---

## How to Add a New Feature

Follow this standardized recipe to add a new domain feature (e.g., `posts`):

### Step 1: Create a Database Migration

Generate a new migration file:
```bash
make migrate-create name=create_posts_table
```

Edit the generated file in `db/migrations/<timestamp>_create_posts_table.sql`:
```sql
-- +goose Up
CREATE TABLE posts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_user_id ON posts(user_id);

-- +goose Down
DROP TABLE IF EXISTS posts;
```

Apply the migration:
```bash
make migrate-up
```

### Step 2: Create the Feature Directory

Create `internal/features/posts/`:
```bash
mkdir -p internal/features/posts
```

Create the following files:

1. **`internal/features/posts/query.sql`** (sqlc queries):
   ```sql
   -- name: CreatePost :one
   INSERT INTO posts (user_id, title, content)
   VALUES ($1, $2, $3)
   RETURNING *;

   -- name: GetPostByID :one
   SELECT * FROM posts WHERE id = $1 LIMIT 1;

   -- name: ListPosts :many
   SELECT * FROM posts
   ORDER BY created_at DESC
   LIMIT $1 OFFSET $2;

   -- name: CountPosts :one
   SELECT count(*) FROM posts;
   ```

2. **`internal/features/posts/posts.go`** (DTOs and models):
   ```go
   package posts

   import (
       "time"
       "github.com/google/uuid"
   )

   type CreatePostRequest struct {
       Title   string `json:"title" binding:"required,min=3,max=255"`
       Content string `json:"content" binding:"required"`
   }

   type PostResponse struct {
       ID        uuid.UUID `json:"id"`
       UserID    uuid.UUID `json:"user_id"`
       Title     string    `json:"title"`
       Content   string    `json:"content"`
       CreatedAt time.Time `json:"created_at"`
   }
   ```

3. **`internal/features/posts/service.go`** (Business logic):
   ```go
   package posts

   import (
       "context"
       "github.com/google/uuid"
       "gin-template-v2/internal/platform/dbgen"
   )

   type Service interface {
       Create(ctx context.Context, userID uuid.UUID, title, content string) (dbgen.Post, error)
       Get(ctx context.Context, id uuid.UUID) (dbgen.Post, error)
   }

   type service struct {
       q dbgen.Querier
   }

   func NewService(q dbgen.Querier) Service {
       return &service{q: q}
   }

   func (s *service) Create(ctx context.Context, userID uuid.UUID, title, content string) (dbgen.Post, error) {
       return s.q.CreatePost(ctx, dbgen.CreatePostParams{
           UserID:  userID,
           Title:   title,
           Content: content,
       })
   }

   func (s *service) Get(ctx context.Context, id uuid.UUID) (dbgen.Post, error) {
       return s.q.GetPostByID(ctx, id)
   }
   ```

4. **`internal/features/posts/handler.go`** (HTTP Transport):
   ```go
   package posts

   import (
       "net/http"
       "github.com/gin-gonic/gin"
       "github.com/google/uuid"
       "gin-template-v2/internal/platform/httpx"
       "gin-template-v2/internal/platform/middleware"
   )

   type Handler struct {
       svc Service
   }

   func NewHandler(svc Service) *Handler {
       return &Handler{svc: svc}
   }

   func (h *Handler) Create(c *gin.Context) {
       user, ok := middleware.CurrentUser(c)
       if !ok {
           httpx.AbortError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
           return
       }

       var req CreatePostRequest
       if err := c.ShouldBindJSON(&req); err != nil {
           httpx.BindError(c, err)
           return
       }

       post, err := h.svc.Create(c.Request.Context(), user.ID, req.Title, req.Content)
       if err != nil {
           httpx.Error(c, http.StatusInternalServerError, "internal_error", "Failed to create post")
           return
       }

       httpx.JSON(c, http.StatusCreated, post)
   }
   ```

5. **`internal/features/posts/route.go`** (Route definition):
   ```go
   package posts

   import "github.com/gin-gonic/gin"

   func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
       g := rg.Group("/posts")
       g.Use(authMiddleware)
       {
           g.POST("", h.Create)
       }
   }
   ```

### Step 3: Register in `sqlc.yaml` and Generate Code

Open `sqlc.yaml` and add the new query file under `queries:`:

```yaml
sql:
  - name: app
    engine: postgresql
    schema: db/migrations
    queries:
      - internal/features/user/query.sql
      - internal/features/auth/query.sql
      - internal/features/posts/query.sql  # <-- Add this line
    gen:
      go:
        package: dbgen
        out: internal/platform/dbgen
        ...
```

Run sqlc to compile queries:
```bash
make sqlc
```

### Step 4: Wire in `internal/server/router.go`

In `internal/server/router.go`:

1. Import the new feature package:
   ```go
   import "gin-template-v2/internal/features/posts"
   ```
2. Initialize and register the handler:
   ```go
   // Under dependency wiring:
   postsHandler := posts.NewHandler(posts.NewService(q))

   // Under API v1 route mounting:
   posts.RegisterRoutes(v1, postsHandler, authMiddleware)
   ```

---

## Adding New API Versions

To introduce a new major API version without breaking existing clients:

```go
// In internal/server/router.go:
v2 := r.Group("/api/v2")

// Mount features that changed:
posts.RegisterRoutesV2(v2, postsHandlerV2, authMiddleware)

// Unchanged features continue serving /api/v1/...
```

Keep breaking modifications confined to the feature's v2 handlers and DTOs; underlying services and database tables can be reused or extended.

---

## Authentication & Authorization

### Session-Based Cookie Auth
- Upon successful login (`/api/v1/auth/login` or `/api/v1/auth/google`), a cryptographically secure random session token is generated.
- The raw token is stored on the client as an `HttpOnly`, `SameSite` cookie.
- The SHA-256 hash of the token is stored in the `user_sessions` database table along with `user_agent`, `ip_address`, and `expires_at`.
- Active sessions are validated on every authenticated request via `middleware.Authenticate()`.

### Administrator Authorization
- The `users` table includes an `is_admin` boolean flag.
- Protected admin routes use `middleware.RequireAdmin()`, which checks if the authenticated user has `is_admin == true`.
- If unauthorized, returns HTTP 403 Forbidden with code `forbidden`.

### Background Cleanup
- The HTTP server starts a background goroutine on launch (`server.runSessionCleanup`).
- Automatically deletes expired sessions from `user_sessions` every 24 hours.

---

## Database & Migration Workflow

| Command | Description |
|---|---|
| `make migrate-create name=<name>` | Create a new SQL migration file in `db/migrations` |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Roll back the most recent migration |
| `make migrate-status` | Display status of applied and pending migrations |
| `make sqlc` | Regenerate Go code in `internal/platform/dbgen` from queries |
| `make db-up` | Start only the PostgreSQL container |
| `make db-psql` | Connect to PostgreSQL with interactive `psql` shell |

> [!TIP]
> The generated files in `internal/platform/dbgen` are committed to git. CI verifies that committed files match `sqlc generate` and checks that migrations apply and roll back cleanly.

---

## Environment Configuration

Configuration is loaded from environment variables (or `.env` during local development) and validated at startup.

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | Application mode: `development`, `production`, `test` |
| `HTTP_PORT` | `8080` | Port for the HTTP server to listen on |
| `LOG_LEVEL` | `debug` (dev) / `info` | Logging verbosity: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `text` (dev) / `json` | Logging format: `text` or `json` |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins; `*` allows any |
| `TRUSTED_PROXIES` | *empty* | Comma-separated CIDR blocks of trusted reverse proxies |
| `DATABASE_URL` | *required* | Postgres connection string with connection options |
| `DB_MAX_CONNS` | `10` | Maximum connections in the pgx pool |
| `DB_MIN_CONNS` | `2` | Minimum idle connections in the pgx pool |
| `GOOGLE_CLIENT_ID` | *optional* | Google OAuth Client ID for Google login token verification |
| `AUTH_COOKIE_NAME` | `sid` | Cookie name for the session token |
| `AUTH_COOKIE_DOMAIN`| *empty* | Cookie domain attribute |
| `AUTH_COOKIE_SECURE`| `false` | Set to `true` in production (requires HTTPS) |
| `AUTH_COOKIE_SAME_SITE` | `lax` | Cookie SameSite setting: `lax`, `strict`, `none` |
| `AUTH_SESSION_DURATION` | `168h` (7d) | Session duration before expiration |
| `ADMINER_PORT` | `8088` | Port for the Adminer container UI in `compose.yaml` |

---

## Makefile Reference

Run `make help` to inspect available commands:

```bash
Develop
  run               Run the API locally (needs database: make db-up migrate-up)
  dev               Run with hot reload (Air)
  build             Compile binary to ./bin/api
  test              Run unit tests with race detector
  cover             Run unit tests with test coverage summary
  lint              Run golangci-lint
  fmt               Format Go code with gofmt
  vet               Run go vet
  tidy              Run go mod tidy

Database
  sqlc              Regenerate Go code from SQL queries
  migrate-up        Apply all pending migrations
  migrate-down      Roll back the last migration
  migrate-status    Show migration status
  migrate-create    Create a migration: make migrate-create name=...
  db-up             Start PostgreSQL container
  db-psql           Open psql inside PostgreSQL container

Containers
  up                Build and start db + migrations + api + adminer
  down              Stop all containers (preserves pgdata volume)
  logs              Tail API container logs
  image             Build production container image

Tooling
  tools             Install goose, sqlc, air, golangci-lint
```

---

## Docker & Podman Notes

The template supports both **Docker** and **Podman** seamlessly.

### Using Podman
`Makefile` defaults to `CONTAINER=podman` and `COMPOSE=podman-compose`.
```bash
# Start containers with podman
make up

# Tail logs
make logs
```

For rootless Podman:
```bash
export DOCKER_HOST="unix://${XDG_RUNTIME_DIR}/podman/podman.sock"
```

### Using Docker
Override the `CONTAINER` and `COMPOSE` variables:
```bash
make up CONTAINER=docker COMPOSE="docker compose"
make down CONTAINER=docker COMPOSE="docker compose"
```

---

## Production Notes & Best Practices

1. **Environment Mode**: Always set `APP_ENV=production`. This enables Gin release mode, turns on structured JSON logging, and disables debug endpoints.
2. **HTTPS & Cookies**:
   - Set `AUTH_COOKIE_SECURE=true` in production to enforce `Secure` flags on auth cookies.
   - Set `AUTH_COOKIE_DOMAIN` to your domain (e.g. `api.example.com` or `.example.com`).
3. **Database URL & Pooling**:
   - Use `sslmode=require` or `sslmode=verify-full` in `DATABASE_URL`.
   - Tune `DB_MAX_CONNS` and `DB_MIN_CONNS` based on server CPU/memory and traffic expectations.
4. **Reverse Proxies & Ingress**:
   - Set `TRUSTED_PROXIES` to the CIDR of your proxy / load balancer (e.g. `10.0.0.0/8` or AWS ALB IP ranges) so `c.ClientIP()` accurately identifies clients.
5. **Database Migrations in CI/CD**:
   - Run migrations as an independent pipeline step (or Kubernetes init container using the `migrate` Dockerfile target) prior to deploying new API pods.
6. **Graceful Shutdown**:
   - The server catches `SIGINT` and `SIGTERM`, drains active connections within `SHUTDOWN_TIMEOUT` (default: 10s), and closes database connection pools safely.

---

## License

This project is licensed under the [MIT License](LICENSE).
