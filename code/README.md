# Bulk Domain Availability Checker

Internal single-administrator application for checking 1–100 domains per batch. The frontend uses React, TypeScript, Vite, and Tailwind. The backend is a small Go API that performs bounded concurrent WhoisFreaks requests, keeps job state temporarily in memory, and exposes progress through polling.

## Runtime architecture

```text
Browser
  -> Nginx / React frontend
  -> /api reverse proxy
  -> Go API
  -> bounded worker pool
  -> WhoisFreaks API
```

The initial batch request returns a job ID immediately. The Go API processes domains asynchronously with a configurable worker pool, while the frontend polls progress. Results are intentionally transient and disappear after the retention window or an API/container restart.

## Required tools

Recommended development versions:

- Go **1.26.5** or another supported Go 1.26 patch release.
- Node.js 24 or newer.
- npm.
- Docker Desktop with Docker Compose for container-based execution.

Verify the tools:

```powershell
go version
node --version
npm --version
docker --version
docker compose version
```

## Windows setup

From the `code` directory in PowerShell:

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\setup-windows.ps1 -AdminPassword "replace-with-a-strong-password"
```

The script:

1. Checks Go, Node.js, and npm.
2. Creates the repository-root `.env` from `.env.example` when missing.
3. Generates a random `JWT_SECRET`.
4. Generates `ADMIN_PASSWORD_HASH` when `-AdminPassword` is supplied.
5. Downloads Go and npm dependencies.
6. Runs backend and frontend tests unless `-SkipTests` is supplied.

Afterward, edit the repository-root `.env` and set a real `WHOISFREAKS_API_KEY`.

Start local development:

```powershell
.\scripts\dev-windows.ps1
```

The script opens separate terminals for:

- Go API: `http://localhost:8080`
- React frontend: `http://localhost:3000`

The Go API automatically discovers `.env` when started from the repository root, `code/`, or `code/server/`. Real environment variables still take precedence over file values.

## Manual setup

Create `.env` at the repository root:

```powershell
Copy-Item .env.example .env
```

Generate the password hash:

```powershell
cd code/server
go run ./cmd/hash-password "replace-with-a-strong-password"
```

Put the output in `ADMIN_PASSWORD_HASH`, generate a 32+ character `JWT_SECRET`, and set `WHOISFREAKS_API_KEY`.

Install dependencies:

```powershell
cd code/server
go mod download

cd ../client
npm ci
```

Run in two terminals:

```powershell
cd code/server
go run ./cmd/api
```

```powershell
cd code/client
npm run dev
```

Health check:

```powershell
Invoke-RestMethod http://localhost:8080/api/health
```

Expected response:

```json
{"status":"ok"}
```

## Environment variables

The application reads the root `.env` automatically for local development. Docker Compose injects the same file into the backend container.

Important settings:

- `BACKEND_PORT`: Go API port, default `8080`.
- `JWT_SECRET`: at least 32 characters.
- `JWT_EXPIRES_IN`: Go duration, such as `8h`.
- `ADMIN_USERNAME`: single administrator username.
- `ADMIN_PASSWORD_HASH`: bcrypt hash, never plaintext.
- `WHOISFREAKS_API_KEY`: server-only upstream credential.
- `DOMAIN_CHECK_CONCURRENCY`: bounded number of simultaneous checks.
- `DOMAIN_CHECK_TIMEOUT_SECONDS`: timeout for one upstream request.
- `DOMAIN_CHECK_MAX_RETRIES`: transient retry count.
- `JOB_RETENTION_MINUTES`: completed-job retention.
- `JOB_CLEANUP_INTERVAL_MINUTES`: cleanup scan interval.
- `CORS_ALLOWED_ORIGINS`: comma-separated local origins.

## Go project layout

```text
server/
├── cmd/
│   ├── api/              # API entrypoint
│   └── hash-password/    # bcrypt helper
└── internal/
    ├── auth/             # JWT and bcrypt authentication
    ├── config/           # .env discovery, parsing, and validation
    ├── domain/           # normalization and validation
    ├── jobs/             # in-memory jobs and bounded workers
    └── whois/            # WhoisFreaks HTTP client
```

The API uses Go modules. `go.mod` is the dependency contract, while `go.sum` records dependency checksums. Run `go mod tidy` only after deliberately adding or removing imports.

## Common commands

From `code/`:

```bash
make setup
make -j2 dev
make fmt
make vet
make test
make test-race
make build
```

On Windows without `make`, run the underlying Go/npm commands or use the PowerShell scripts.

## Docker Compose

From `code/`:

```powershell
docker compose config
docker compose up --build
```

Open `http://localhost:3000`. Nginx proxies `/api` to the private backend service; the backend port is not published to the host in Compose mode.

Stop the stack:

```powershell
docker compose down
```

View logs:

```powershell
docker compose logs -f
```

## Cloudflare Tunnel

Set `CLOUDFLARE_TUNNEL_TOKEN` in the root `.env`, configure the tunnel's public hostname to route to:

```text
http://frontend:80
```

Then start the optional profile:

```powershell
docker compose --profile tunnel up --build -d
```

The tunnel publishes the frontend/Nginx entry point. The Go backend remains private on the Compose network, and `/api` is proxied internally.

## Validation

```powershell
cd code/server
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build -buildvcs=false -o api.exe ./cmd/api

cd ../client
npm run lint
npm test
npm run build

cd ..
docker compose config
docker compose build
```

## API endpoints

- `GET /api/health`
- `POST /api/auth/login`
- `POST /api/domain-checks`
- `GET /api/domain-checks/{jobId}`

The WhoisFreaks response field is normalized from the documented `availability` value into `AVAILABLE`, `TAKEN`, or `ERROR`. Individual failures do not abort the full batch.

## Known limitations

- Jobs live only in one Go process; horizontal scaling requires shared state such as Redis and a queue.
- Refreshing the browser or restarting the backend can lose active result context.
- The in-memory login limiter is instance-local.
- Upstream plan limits still matter. Tune concurrency according to the actual WhoisFreaks plan rather than assuming that a higher value is always faster.
