# MSLatencyTracker-Go

A Go/[Gin](https://gin-gonic.com/) microservice that monitors MapleStory game
server latency. Every 5 minutes it probes every configured channel IP in
parallel by timing a TCP handshake against the game port (8585) — the channel
servers drop ICMP, and the handshake is exactly one round trip over the same
path the game client uses. The latest reading per channel is cached in memory,
and history is persisted to PostgreSQL ([pgx](https://github.com/jackc/pgx))
for average and time-series queries.

## Architecture

```
┌─────────────┐    every 5 min    ┌──────────────────┐
│ ping worker │──────────────────>│ TCP probe :8585  │
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
These readings appear in `/history` (so charts can show outage gaps) but are
excluded from `/average`.

### Example Responses

**GET /api/v1/worlds**
```json
[
  { "name": "Scania", "channelCount": 30 },
  { "name": "Bera", "channelCount": 30 },
  { "name": "Kronos", "channelCount": 40 },
  { "name": "Hyperion", "channelCount": 30 }
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
| `POSTGRES_PASSWORD` | Compose only | — | Password for the bundled postgres container |
| `POSTGRES_USER` | No | `mslatency` | User for the bundled postgres container |
| `POSTGRES_DB` | No | `mslatency` | Database name in the bundled postgres container |
| `TUNNEL_TOKEN` | Compose only | — | Cloudflare Tunnel token for the bundled cloudflared container |
| `DATABASE_URL` | Local dev only | — | PostgreSQL connection string; Docker Compose derives its own from the `POSTGRES_*` values |
| `PORT` | No | `8080` | HTTP server port |
| `PING_INTERVAL_MS` | No | `300000` | Ping interval in milliseconds (5 min) |
| `PING_TIMEOUT_S` | No | `5` | Per-ping timeout in seconds |
| `HISTORY_RETENTION_HOURS` | No | `24` | How long to keep history rows |
| `CLEANUP_INTERVAL_MIN` | No | `60` | How often the cleanup job runs (minutes) |

## Local Development

### Prerequisites

- Go 1.26+
- PostgreSQL (local or remote)

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

Docker Compose runs three containers — the tracker, a bundled PostgreSQL, and
a [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/)
(`cloudflared`) that publishes the API without exposing the host:

```
internet ──https──▶ Cloudflare edge ──tunnel──▶ cloudflared ──http──▶ tracker ──▶ postgres
                                                 (container)          (:8080)     (internal)
```

Neither the tracker nor postgres publishes any host port. `cloudflared` makes
an **outbound-only** connection to Cloudflare, so there is no router
port-forwarding and your origin IP never appears in DNS. Postgres additionally
sits on an `internal: true` network reachable only by the tracker.

### 1. Create the tunnel

Use a remotely-managed (token) tunnel — ingress is configured in the
Cloudflare dashboard, and the container only needs the token:

1. Cloudflare dashboard → **Zero Trust** → **Networks** → **Tunnels** →
   **Create a tunnel** → **Cloudflared**.
2. Name it (e.g. `mslatencytracker`) → **Save**, then copy the tunnel token
   (the long string after `--token` in the install command shown — you do
   **not** run that command; Compose runs `cloudflared` for you).
3. **Public Hostname** tab → **Add a public hostname**:
   - **Subdomain/Domain:** your choice (e.g. `latency.example.com`)
   - **Type:** `HTTP`
   - **URL:** `tracker:8080`  ← the Compose service name and internal port

Cloudflare auto-creates the DNS record pointing at the tunnel.

### 2. Configure and start

```bash
cp .env.example .env   # set POSTGRES_PASSWORD and TUNNEL_TOKEN
docker compose up -d --build
docker compose logs -f cloudflared   # look for "Registered tunnel connection"
```

Verify from any machine:

```bash
curl https://latency.example.com/health
curl https://latency.example.com/api/v1/worlds
dig +short latency.example.com   # Cloudflare anycast IPs — never your home IP
```

> **Local testing without a tunnel:** temporarily add a
> `ports: ["8080:8080"]` mapping to the `tracker` service, or run the app
> directly with `go run .` (see Local Development).

Data persists across restarts in the `pgdata` named volume. The service shuts
down gracefully on `SIGINT`/`SIGTERM`: the ping and cleanup workers stop, and
in-flight HTTP requests get a 10-second grace period.

Because the app sits behind Cloudflare, it reads the real client IP from the
`CF-Connecting-IP` header (Gin's `TrustedPlatform`) and otherwise trusts no
proxy headers.
