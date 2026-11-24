# syntax=docker/dockerfile:1

# ---- Build stage: compile a static binary ----
FROM golang:1.27-alpine AS build
WORKDIR /src

# Download dependencies first so this layer is cached between code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Runtime stage: no shell, no package manager, non-root user ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
