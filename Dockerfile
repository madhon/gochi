ARG GOVERSION=1.26.4

# Step 1: Builder
FROM --platform=$BUILDPLATFORM golang:${GOVERSION} AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify
COPY . .
ARG TARGETOS TARGETARCH VERSION=dev BUILD_TIME
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -trimpath \
    -o /bin/app ./cmd/main.go

# Step 2: Tzdata source
FROM --platform=$BUILDPLATFORM debian:bookworm-slim AS tzdata
RUN apt-get update && apt-get install -y --no-install-recommends tzdata \
    && rm -rf /var/lib/apt/lists/*

# Step 3: Final
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=tzdata /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
WORKDIR /app
COPY --from=builder /bin/app ./
COPY --from=builder /app/app.env ./
USER nonroot:nonroot
EXPOSE 4343
ENTRYPOINT ["/app/app"]