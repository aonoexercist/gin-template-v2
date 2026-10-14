# syntax=docker/dockerfile:1
# Works with Docker and Podman (fully-qualified image names avoid Podman's registry prompt).
#
#   docker build -t myapp .                      # runtime image (default, last stage)
#   docker build --target migrate -t myapp-migrate .   # goose + migrations

ARG GO_VERSION=1.24

# ---------- build ----------
FROM docker.io/library/golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w \
        -X gin-template-v2/internal/version.Version=${VERSION} \
        -X gin-template-v2/internal/version.Commit=${COMMIT} \
        -X gin-template-v2/internal/version.BuildDate=${BUILD_DATE}" \
      -o /out/api ./cmd/api

RUN --mount=type=cache,target=/go/pkg/mod \
    GOBIN=/out CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.22.1

# ---------- migrate (one-shot job: applies db/migrations) ----------
FROM docker.io/library/alpine:3.20 AS migrate
COPY --from=build /out/goose /usr/local/bin/goose
COPY db/migrations /migrations
# Expects DATABASE_URL in the environment.
CMD ["sh", "-c", "goose -dir /migrations postgres \"$DATABASE_URL\" up"]

# ---------- runtime (default) ----------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/api /app/api
ENV HTTP_PORT=8080 APP_ENV=production
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/api", "healthcheck"]
ENTRYPOINT ["/app/api"]
