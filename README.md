# MSLatencyTracker-Go

A Go/[Gin](https://gin-gonic.com/) microservice that monitors MapleStory game
server latency. It pings every configured channel IP in parallel over raw ICMP
([pro-bing](https://github.com/prometheus-community/pro-bing)) every 5 minutes,
caches the latest reading per channel in memory, and persists history to
PostgreSQL ([pgx](https://github.com/jackc/pgx)) for average and time-series
queries.

## Architecture

```
┌─────────────┐    every 5 min    ┌──────────────────┐
│ ping worker │──────────────────>│  ICMP Ping       │
│ (goroutine  │                   │  (all IPs, ||)   │
│  + Ticker)  │                   └──────────────────┘
└──────┬──────┘
       │ records latency
       v
┌──────────────┐   in-memory    ┌───────────────────┐
│    store     │───────────────>│ map[key]reading   │
│ (sync.RWMutex│                │ (latest per ch)   │
│  + SQL)      │   persist      ├───────────────────┤
│              │───────────────>│ PostgreSQL        │
│              │                │ (24h history)     │
└──────┬───────┘                └───────────────────┘
       ^
       │ queries
┌──────┴───────┐
│   Gin API    │
│  /api/v1/... │
└──────────────┘
```

## API Endpoints

All endpoints are under `/api/v1`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/worlds` | Lists all worlds and their channel counts |
| GET | `/api/v1/{world}/latency` | Latest latency (ms) for every channel in a world |
| GET | `/api/v1/{world}/{channel}/latency` | Latest latency (ms) for a specific channel |
| GET | `/api/v1/{world}/{channel}/latency/average` | Average latency over the last hour |
| GET | `/api/v1/{world}/{channel}/latency/history` | Full 24-hour history (designed for Chart.js) |
| GET | `/health` | Database connectivity check |

A `latencyMs` value of `-1` indicates the server was unreachable or timed out.

### Example Responses

**GET /api/v1/worlds**
```json
[
  { "name": "Scania", "channelCount": 0 },
  { "name": "Bera", "channelCount": 0 },
  { "name": "Kronos", "channelCount": 40 },
  { "name": "Hyperion", "channelCount": 0 }
]
```

**GET /api/v1/kronos/1/latency**
```json
{ "world": "Kronos", "channel": 1, "latencyMs": 23, "timestamp": 1709000000000 }
```

**GET /api/v1/kronos/1/latency/history**
```json
{
  "world": "Kronos",
  "channel": 1,
  "dataPoints": [
    { "timestamp": 1709000000000, "latencyMs": 23 },
    { "timestamp": 1709000300000, "latencyMs": 25 }
  ]
}
```

## Project Structure

```
MSLatencyTracker-Go/
├── main.go                   # Entry point: config, wiring, server start
├── config/
│   └── servers.go            # World type + IP configuration + lookup helpers
├── db/
│   └── db.go                 # Postgres connection, migration, cleanup
├── store/
│   └── store.go              # Data access + in-memory latest-value cache
├── pinger/
│   └── pinger.go             # Background ping worker (goroutines + Ticker)
├── handlers/
│   └── latency.go            # Gin route handlers
├── Dockerfile
├── docker-compose.yml
├── go.mod / go.sum
├── .env.example
├── .gitignore
└── README.md
```

## Environment Variables

Copy `.env.example` to `.env` and fill in the values:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PORT` | No | `8080` | HTTP server port |
| `PING_INTERVAL_MS` | No | `300000` | Ping interval in milliseconds (5 min) |
| `PING_TIMEOUT_S` | No | `5` | Per-ping timeout in seconds |
| `HISTORY_RETENTION_HOURS` | No | `24` | How long to keep history rows |
| `CLEANUP_INTERVAL_MIN` | No | `60` | How often the cleanup job runs (minutes) |

## Local Development

### Prerequisites

- Go 1.25+
- PostgreSQL (local or remote)
- On Linux, ICMP ping needs the `NET_RAW` capability or root. On Windows and
  macOS, running the binary normally is sufficient.

### Run

```bash
# Download dependencies
go mod download

# Create .env from the template and set DATABASE_URL
cp .env.example .env

# Run directly (no build step needed)
go run .

# …or build a binary and run it
go build -o mslatencytracker .
./mslatencytracker
```

The server starts on `http://localhost:8080`. The schema is created
automatically on startup (`CREATE TABLE IF NOT EXISTS`), so there is no
separate migration command.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/worlds
```

## Deployment

Build and run with Docker Compose:

```bash
cp .env.example .env   # set DATABASE_URL
docker compose up -d --build
```

The compose file grants `NET_RAW` for ICMP, restarts unless stopped, and
exposes port 8080.

The service shuts down gracefully on `SIGINT`/`SIGTERM`: the ping and cleanup
workers stop, and in-flight HTTP requests get a 10-second grace period.
