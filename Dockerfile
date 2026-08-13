# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o keyparty .

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/keyparty .
COPY --from=builder /app/web ./web

EXPOSE 8080

VOLUME ["/app/data"]

ENTRYPOINT ["./keyparty"]
CMD ["-port", "8080", "-db", "/app/data/gateway.db"]
