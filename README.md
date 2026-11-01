# Signwell

Signwell is a Go HTTP API for a sign-painting studio. Track commissions by substrate and medium, advance each board from quote through lettering, and keep a shop-floor note trail as work moves.

## Stack

- Go 1.22+
- [chi](https://github.com/go-chi/chi) router
- In-memory store with mutex safety
- Standard library `net/http` server

## Features

- Sign job CRUD with unique job codes
- Status workflow: `quoted` → `sketched` → `lettered` → `finishing` → `ready` → `delivered`
- Dedicated advance action with conflict checks
- Shop notes that can also move a job to a named status
- Filter by status, substrate, or free-text query
- JSON error responses with status + detail
- Seeded demo jobs on startup

## Architecture

```
cmd/signwell
    │
    ▼
httpapi (chi routes)
    │
    ▼
store.MemoryStore
    └── jobs + notes
```

## Project layout

```
signwell/
├── cmd/signwell/         # process entrypoint
├── internal/
│   ├── httpapi/          # HTTP handlers
│   ├── models/           # request/response types
│   └── store/            # in-memory domain store
├── go.mod
├── LICENSE
├── README.md
└── requests.http
```

## Setup

```bash
git clone https://github.com/<your-username>/signwell.git
cd signwell
go mod tidy
```

## Run

```bash
go run ./cmd/signwell
```

Optional address override:

```bash
# Windows PowerShell
$env:SIGNWELL_ADDR=":8080"; go run ./cmd/signwell

# macOS / Linux
SIGNWELL_ADDR=:8080 go run ./cmd/signwell
```

- API root: http://localhost:8080/
- Health: http://localhost:8080/health
- Jobs: http://localhost:8080/api/v1/jobs

## API overview

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness check |
| `GET` | `/api/v1/jobs` | List jobs (`status`, `substrate`, `q`, `limit`, `offset`) |
| `POST` | `/api/v1/jobs` | Create a job |
| `GET` | `/api/v1/jobs/{id}` | Get one job |
| `PATCH` | `/api/v1/jobs/{id}` | Update a job |
| `DELETE` | `/api/v1/jobs/{id}` | Delete a job |
| `POST` | `/api/v1/jobs/{id}/advance` | Move to next status |
| `POST` | `/api/v1/jobs/{id}/notes` | Append a shop note (optional `status`) |

### Example

```bash
# Create a quoted job
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d "{\"code\":\"SW-210\",\"client\":\"Harbor Inn\",\"title\":\"Porch hanging sign\",\"substrate\":\"wood\",\"medium\":\"enamel\",\"dueOn\":\"2026-09-01\",\"tags\":[\"inn\"]}"

# Advance quoted → sketched
curl -X POST http://localhost:8080/api/v1/jobs/<job-id>/advance

# Log lettering
curl -X POST http://localhost:8080/api/v1/jobs/<job-id>/notes \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"Layout approved on tracing paper.\",\"status\":\"lettered\"}"
```

## Tests

```bash
go test ./...
```

## Build

```bash
go build -o signwell ./cmd/signwell
./signwell
```

## License

MIT
