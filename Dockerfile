# Step 1: Modules caching
FROM --platform=$BUILDPLATFORM golang:1.25.3 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download && go mod verify

# Step 2: Builder
FROM --platform=$BUILDPLATFORM golang:1.25.3 AS builder
COPY --from=modules /go/pkg /go/pkg

COPY . /app
WORKDIR /app

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 \
  go build -o /bin/app ./cmd/main.go

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS:-linux} \
    GOARCH=${TARGETARCH:-amd64} \
    go build -a \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -trimpath \
    -o /bin/app ./cmd/main.go

# Step 3: Final  
#FROM gcr.io/distroless/base-debian12
#FROM gcr.io/distroless/static:nonroot
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

COPY --from=builder /bin/app /app/
COPY --from=builder /app/app.env /app/

USER nonroot:nonroot
EXPOSE 4343

ENTRYPOINT ["/app/app"]

