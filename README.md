# Pelagica Collector

A lightweight, privacy-respecting analytics collector for [Pelagica](https://github.com/PelagicaApp/pelagica), a custom frontend for Jellyfin. It counts how many instances are installed and how many are actively running, without collecting any personally identifiable information.

## How it works

Each Pelagica instance generates a random UUID on first run and sends a minimal ping to the collector once every 24 hours. The collector records the instance ID, version, and timestamp. No IP addresses are stored. No user data is collected.

The `/stats` endpoint is public so anyone can verify what the numbers look like.

## Data collected

| Field | Description |
|---|---|
| `instance_id` | A random UUID generated on first install |
| `version` | The Pelagica version string |
| `pinged_at` | Timestamp of the ping |

Nothing else is collected or stored.

## API

### `POST /ping`

Sent automatically by Pelagica once per 24 hours.

**Request body:**
```json
{
  "instance_id": "a4b1c2d3-e5f6-7890-abcd-ef1234567890",
  "version": "1.2.0",
  "token": "optional-token-if-needed"
}
```

**Responses:**
- `204 No Content` on success
- `400 Bad Request` if the payload is invalid
- `401 Unauthorized` if a token is required and missing/invalid
- `429 Too Many Requests` if the rate limit is exceeded

### `GET /stats`

Returns the current install and activity counts.

**Response:**
```json
{
  "total_installs": 412,
  "active_instances": 198
}
```

Active instances are those that have sent a ping in the last 48 hours.

## Running with Docker

The recommended way to run the collector is via Docker Compose.

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_USER: pelagica
      POSTGRES_PASSWORD: yourpassword
      POSTGRES_DB: pelagica
    volumes:
      - postgres_db_data:/var/lib/postgresql/data
    expose:
      - "5432"

  collector:
    image: kartoffelchipss/pelagica-collector:latest
    environment:
      DATABASE_URL: postgres://pelagica:yourpassword@postgres:5432/pelagica
      BEHIND_PROXY: "true"
    ports:
      - "4000:4000"

volumes:
  postgres_db_data:
```

## Configuration

All configuration is done via environment variables.

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | Postgres connection string, e.g. `postgres://user:pass@host:5432/db` |
| `BEHIND_PROXY` | No | Set to `true` if running behind a reverse proxy. Enables `X-Forwarded-For` trust. Default: `false` |
| `TRUSTED_PROXIES` | No | Comma-separated list of trusted proxy IPs or CIDR ranges if `BEHIND_PROXY=true`. Default: empty (trust all proxies) |
| `PORT` | No | Port to listen on. Default: `4000` |
| `PING_TOKEN` | No | If set, requires this token in the `token` field of the ping payload for authentication. Default: no token required |

## Reverse proxy

If you run the collector behind Nginx or Caddy, set `BEHIND_PROXY=true`. This tells the collector to read the real client IP from the `X-Forwarded-For` header for rate limiting purposes.

For Nginx, make sure you forward the real IP:

```nginx
proxy_set_header X-Forwarded-For $remote_addr;
```

## Building from source

Requires Go 1.25 or later.

```bash
git clone https://github.com/PelagicaApp/pelagica-collector
cd pelagica-collector
go build -o collector .
```

## Database

The collector manages its own schema and runs migrations automatically on startup. No manual setup is required beyond creating the database.

Ping history older than 90 days should be pruned periodically. You can set up a cron job or Postgres scheduled task with:

```sql
DELETE FROM pings WHERE pinged_at < now() - interval '90 days';
```

## License

MIT. See [LICENSE](LICENSE).