# Setup Go Backend — Quick Start

## 1\. Install prerequisites on Windows

Install:

* Go 1.26.5 (Windows x64 MSI)
* Node.js 24+
* Docker Desktop, if using Docker Compose
* VS Code with the official Go extension, optional but recommended

Verify in PowerShell:

```powershell
go version
node --version
npm --version
docker --version
docker compose version
```

## 2\. Bootstrap the project

Open PowerShell in the `code` directory:

```powershell
Set-ExecutionPolicy -Scope Process Bypass .\\scripts\\setup-windows.ps1 -AdminPassword "replace-with-a-strong-password"
```

Then edit the root `.env` and set:

```env
WHOISFREAKS\_API\_KEY=your-real-api-key
```

The script already creates a random JWT secret and a bcrypt password hash.

## 3\. Run locally

From `code/`:

```powershell
.\\scripts\\dev-windows.ps1
```

Open:

* Frontend: `http://localhost:3000`
* Backend health: `http://localhost:8080/api/health`

## 4\. Run with Docker Compose

From `code/`:

```powershell
docker compose config
docker compose up --build
```

Open `http://localhost:3000`.

For Cloudflare Tunnel, configure `CLOUDFLARE\_TUNNEL\_TOKEN` and route the public hostname to `http://frontend:80`, then run:

```powershell
docker compose --profile tunnel up --build -d
```

## 5\. Backend development commands

```powershell
cd code/server

gofmt -w .
go vet ./...
go test ./...
go test -race ./...
go build -buildvcs=false -o api.exe ./cmd/api
```

Generate another password hash:

```powershell
go run ./cmd/hash-password "new-password"
```

## Important behavior

* The backend automatically discovers the root `.env` when started from the repository root, `code/`, or `code/server/`.
* Real operating-system environment variables override `.env` values.
* `ADMIN\_PASSWORD\_HASH` must be a valid bcrypt hash.
* `JWT\_SECRET` must contain at least 32 characters.
* `WHOISFREAKS\_API\_KEY` remains backend-only.
* Domain jobs are kept in memory and disappear after backend restart.

