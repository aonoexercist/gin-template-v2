# Gin API Template

Production-minded Go REST API starter: **Gin** + **PostgreSQL** (pgx) + **sqlc** (type-safe queries) + **goose** (migrations), with a **feature-based** layout, `/api/v1` versioning, structured logging, graceful shutdown and Docker/Podman support.

## What's included

- **Feature-based structure** – each feature owns its handler, service, repository, DTOs and SQL
- **sqlc** per feature, reading the **goose** migrations as its schema
- **API versioning** – `/api/v1`, add `/api/v2` side by side without touching v1
- **Structured logging** (`log/slog`): text in dev, JSON in prod, request-scoped logger with `request_id`
- Middleware: request ID, access log, panic recovery, security headers, CORS
- Consistent JSON envelope for data and errors, validation errors keyed by JSON field name
- `/healthz` (liveness) and `/readyz` (readiness, pings the DB)
- Env-based config with validation, graceful shutdown, pgx connection pool
- Multi-stage Dockerfile (distroless, non-root) + `compose.yaml` (db → migrate → api), works with Docker **and** Podman
- Unit tests, golangci-lint, GitHub Actions CI (test, lint, sqlc drift check, migration up/down, image build)

## Project layout

```
cmd/api/                    entrypoint (main.go, `healthcheck` subcommand)
db/migrations/              goose SQL migrations (schema source of truth)
internal/
  config/                   env config
  server/                   http.Server lifecycle + router (middleware, versions, wiring)
  platform/                 shared infrastructure, no business logic
    database/  logger/  httpx/  middleware/
  features/
    health/                 /healthz, /readyz
    users/                  example feature
      handler.go            HTTP <-> service
      service.go            business rules
      repository.go         persistence boundary (wraps sqlc)
      queries.sql           sqlc input
      usersdb/              sqlc output (generated, committed)
sqlc.yaml
```

Dependency direction: `handler → service → repository → usersdb`. Features never import each other's internals; shared things go in `platform/`.

## Getting started

```bash
# 1. Use the template (GitHub: "Use this template"), then rename the module
./scripts/rename.sh github.com/<you>/<project>
go mod tidy                      # generates go.sum

# 2. Configure
cp .env.example .env

# 3a. Everything in containers
make up                          # docker;   or: make up CONTAINER=podman
curl localhost:8080/readyz

# 3b. ...or develop on your host with only Postgres in a container
make tools                       # installs goose, sqlc, air, golangci-lint
make db-up
make migrate-up
make dev                         # hot reload (or: make run)
```

Try it:

```bash
curl -X POST localhost:8080/api/v1/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"jane@example.com","name":"Jane"}'
curl 'localhost:8080/api/v1/users?page=1&per_page=20'
```

### Podman

`make up CONTAINER=podman` runs `podman compose`. For rootless Podman set `export DOCKER_HOST=unix://$XDG_RUNTIME_DIR/podman/podman.sock` if your compose provider needs it. Image names are fully qualified, so no registry prompts. `podman-compose` also works: `make up COMPOSE=podman-compose`.

## API conventions

Success: `{"data": ..., "meta": {"page":1,"per_page":20,"total":42}}`

Error:
```json
{"error": {"code": "validation_error", "message": "Request validation failed",
           "details": {"email": "failed 'email' validation"}, "request_id": "…"}}
```

| Status | `code`                         |
|--------|--------------------------------|
| 400    | `invalid_request`, `invalid_id` |
| 404    | `not_found`                    |
| 409    | `email_taken`                  |
| 422    | `validation_error`             |
| 500    | `internal_error` (details only in logs) |

Clients may send `X-Request-ID`; it is echoed back and attached to every log line.

## Adding a feature

1. `cp -r internal/features/users internal/features/posts` and rename the package/types.
2. `make migrate-create name=create_posts`, write the table in the `Up`/`Down` sections.
3. Edit `internal/features/posts/queries.sql`, add an entry to `sqlc.yaml` (new `package`/`out`), run `make sqlc`.
4. Wire it in `internal/server/router.go` (one line for deps, one for routes).

## Adding API v2

```go
v2 := r.Group("/api/v2")
posts.RegisterRoutesV2(v2, postsHandler) // only what changed; v1 stays as is
```
Keep breaking changes in the new version's handlers/DTOs; services and repositories are shared.

## Database workflow

| Task                  | Command                                |
|-----------------------|----------------------------------------|
| New migration         | `make migrate-create name=add_x`       |
| Apply / roll back one | `make migrate-up` / `make migrate-down`|
| Status                | `make migrate-status`                  |
| Regenerate sqlc code  | `make sqlc`                            |

CI fails if committed sqlc output differs from `sqlc generate`, and checks that migrations go up → down → up.

> The files in `usersdb/` are committed generated code. Run `make sqlc` once after cloning if you change queries or want to confirm they match your installed sqlc version.

## Configuration

See [`.env.example`](.env.example). Key variables: `APP_ENV`, `HTTP_PORT`, `DATABASE_URL` (required), `LOG_LEVEL`, `LOG_FORMAT`, `CORS_ALLOWED_ORIGINS`, `TRUSTED_PROXIES`, `DB_MAX_CONNS`.

## Production notes

- Set `APP_ENV=production` (Gin release mode, JSON logs) and use a real `DATABASE_URL` with `sslmode=require` or stronger.
- Set `TRUSTED_PROXIES` to your load balancer's CIDR, otherwise `client_ip` in logs is the proxy's address.
- Run migrations as a separate step (the `migrate` image/target) before rolling out new app versions; keep migrations backward compatible with the previous app version.
- Authentication/authorization is intentionally not included: add it as a `platform/auth` middleware plus a feature.

## License

MIT
