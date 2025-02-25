# Step 1: Modules caching
FROM golang:1.24.0 AS modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

# Step 2: Builder
FROM golang:1.24.0 AS builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 \
  go build -o /bin/app ./cmd/main.go

# Step 3: Final  
#FROM gcr.io/distroless/base-debian12
FROM gcr.io/distroless/static:nonroot

EXPOSE 4343

WORKDIR /

COPY --from=builder /bin/app /
COPY --from=builder /app/app.env /
CMD ["/app"]

