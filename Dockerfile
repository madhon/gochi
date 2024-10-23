FROM golang:1.23.2 AS builder

WORKDIR /app

ENV GO111MODULE=on

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v2 go build -o chier cmd/main.go

#FROM gcr.io/distroless/base-debian12
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /app/chier /
COPY --from=builder /app/app.env /
EXPOSE 4343
CMD ["/chier"]

