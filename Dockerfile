# --- Build stage ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache the dependency layer separately from the source.
COPY go.mod go.sum ./
RUN go mod download

# CGO_ENABLED=0 produces a statically linked binary that runs on plain alpine.
COPY . .
RUN CGO_ENABLED=0 go build -o /mslatencytracker .

# --- Runtime stage ---
FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 app

WORKDIR /app
COPY --from=builder /mslatencytracker /app/mslatencytracker

# Release mode disables Gin's per-request debug logging.
ENV GIN_MODE=release

# The binary only needs to bind :8080 and open outbound sockets — no root.
USER app

EXPOSE 8080

CMD ["/app/mslatencytracker"]
