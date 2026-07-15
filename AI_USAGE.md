# Laporan Ringkasan Pengembangan Bulk Domain Availability Checker

## 1. Ringkasan Eksekutif

Proyek ini bertujuan membangun aplikasi internal untuk memeriksa ketersediaan domain secara massal. Pengguna dapat memasukkan antara 1 sampai 100 domain dalam satu kali proses, memantau progres pengecekan, memfilter hasil, dan mengekspor hasil ke CSV.

Aplikasi dirancang dengan alur penggunaan yang sederhana dan fokus, terinspirasi dari pola UX iLovePDF: pengguna login, memasukkan daftar domain, menjalankan pengecekan, melihat progres, lalu meninjau dan mengekspor hasil.

Keputusan teknologi akhir yang dipilih adalah:

- Frontend: React, TypeScript, Vite, dan Tailwind CSS.
- Backend: Go.
- Authentication: JWT dan bcrypt.
- Domain availability API: WhoisFreaks.
- Runtime state: in-memory job manager.
- Concurrency: bounded worker pool berbasis goroutine.
- Reverse proxy dan static hosting: Nginx.
- Containerization: Docker.
- Local orchestration dan deployment: Docker Compose.
- Optional public access: Cloudflare Tunnel melalui `cloudflared`.
- API testing: Postman Collection dan Postman Environment.

---

## 2. Latar Belakang Kebutuhan

Kebutuhan utama muncul karena proses pengecekan domain satu per satu melalui website pihak ketiga dianggap lambat dan tidak efisien untuk kebutuhan internal.

Aplikasi yang dibutuhkan harus memungkinkan pengguna internal untuk:

1. Login menggunakan akun administrator.
2. Memasukkan 1 sampai 100 domain sekaligus.
3. Menjalankan pengecekan ketersediaan domain secara batch.
4. Melihat progres pengecekan.
5. Menampilkan hasil sebagai `AVAILABLE`, `TAKEN`, atau `ERROR`.
6. Memfilter hasil berdasarkan status.
7. Mencari domain tertentu dari hasil.
8. Mengekspor hasil ke CSV.
9. Menggunakan aplikasi tanpa penyimpanan database permanen.

---

## 3. Requirement Awal

Requirement awal yang disepakati adalah sebagai berikut:

### Authentication

- Menggunakan JWT.
- Password administrator disimpan dalam bentuk bcrypt hash.
- Hanya diperlukan satu akun administrator.
- Tidak diperlukan registration.
- Tidak diperlukan user management.
- Tidak diperlukan forgot password.

### Domain Checking

- Pengguna dapat memasukkan 1 sampai 100 domain.
- Domain dicek menggunakan WhoisFreaks Domain Availability API.
- API key hanya boleh tersedia di backend.
- Hasil tidak perlu disimpan secara permanen.
- Hasil cukup ditampilkan selama sesi aktif.

### Progress

- Aplikasi harus menyediakan progress bar.
- Progres harus mewakili pekerjaan batch yang benar-benar telah selesai.
- UI tidak boleh freeze selama proses berjalan.

### Result Dashboard

- Filter:
  - All
  - Available
  - Taken
  - Error
- Search domain.
- Summary cards:
  - Total
  - Available
  - Taken
  - Error
- Export CSV.

### UX

UX diarahkan mengikuti pola interaksi iLovePDF:

- Satu pekerjaan utama per layar.
- Area input besar di tengah.
- Primary action yang jelas.
- Processing state yang sederhana.
- Result state yang langsung dapat digunakan.
- Tidak meniru branding atau aset visual iLovePDF.

---

## 4. API Eksternal

API yang digunakan adalah WhoisFreaks Domain Availability API.

Contoh request:

```bash
curl "https://api.whoisfreaks.com/v1.0/domain/availability?domain=example.com&apiKey=YOUR_API_KEY"
```

Catatan penting yang ditemukan selama proses:

- Query parameter kedua harus menggunakan `&apiKey=`, bukan `?apiKey=`.
- API key tidak boleh dikirim dari frontend.
- Backend harus menjadi satu-satunya service yang berkomunikasi dengan WhoisFreaks.
- Field response utama yang digunakan adalah `availability`.
- Backend juga disiapkan defensif untuk menerima kemungkinan fallback field lama seperti `domain_availability`.

Mapping status internal:

| WhoisFreaks | Internal |
|---|---|
| `AVAILABLE` | `AVAILABLE` |
| `UNAVAILABLE` | `TAKEN` |
| Unknown atau missing | `ERROR` |
| Request gagal | `ERROR` |

---

## 5. Evolusi Keputusan Arsitektur

## 5.1 Rencana Awal

Stack awal yang diusulkan:

- React + Vite.
- Node.js + Express.
- PostgreSQL.
- Prisma.
- Tailwind CSS.
- JWT.
- bcrypt.

Setelah requirement diperjelas bahwa hasil tidak perlu disimpan, database dan Prisma tidak lagi diperlukan.

## 5.2 Penghapusan Database

Keputusan akhir:

- Tidak menggunakan PostgreSQL.
- Tidak menyimpan hasil permanen.
- Hasil disimpan sementara di memory backend dan state frontend.
- Hasil akan hilang ketika backend restart, container restart, atau job expired.

Alasan:

- Scope aplikasi internal relatif kecil.
- Tidak diperlukan history.
- Deployment menjadi lebih sederhana.
- Prototyping menjadi lebih cepat.
- Operational overhead lebih rendah.

## 5.3 Perubahan Backend ke Go

Backend kemudian diubah dari Node.js menjadi Go.

Alasan pemilihan Go:

- Cocok untuk concurrent I/O.
- Goroutine ringan.
- Worker pool mudah dibuat.
- Deployment sederhana sebagai compiled binary.
- Memory footprint rendah.
- Cocok untuk container.
- Lebih mudah mengontrol timeout, context, dan bounded concurrency.
- Tepat untuk banyak request eksternal paralel.

## 5.4 Docker Compose

Docker Compose dipilih sebagai orchestration layer untuk:

- Prototyping.
- Local development.
- Deployment sederhana.
- Menjalankan frontend, backend, dan cloudflared secara konsisten.
- Menghindari kebutuhan Kubernetes pada tahap awal.

Service utama:

```text
frontend
backend
cloudflared
```

## 5.5 Cloudflare Tunnel

Cloudflare Tunnel dipilih sebagai fitur opsional.

Flow:

```text
Internet
   ↓
Cloudflare Tunnel
   ↓
Frontend / Nginx
   ↓ /api
Go Backend
   ↓
WhoisFreaks API
```

Prinsipnya:

- `cloudflared` tidak langsung mengekspos backend.
- Traffic publik masuk ke frontend atau Nginx.
- Nginx meneruskan request `/api` ke Go backend.
- Backend tetap berada di internal Docker network.
- JWT tetap digunakan meskipun Cloudflare Tunnel aktif.
- Cloudflare Access dapat ditambahkan di masa depan sebagai defense in depth.

---

## 6. Desain Alur Sistem

## 6.1 Authentication Flow

```text
User
  ↓ username + password
POST /api/auth/login
  ↓
Go Backend
  ↓ bcrypt comparison
JWT generated
  ↓
Frontend stores JWT in sessionStorage
  ↓
Authorization: Bearer <token>
```

JWT direncanakan memiliki masa berlaku default delapan jam.

## 6.2 Domain Batch Flow

```text
User inputs domains
  ↓
Frontend parses and previews input
  ↓
POST /api/domain-checks
  ↓
Backend validates and normalizes
  ↓
Backend creates in-memory job
  ↓
Backend returns jobId immediately
  ↓
Worker pool checks WhoisFreaks
  ↓
Frontend polls job status
  ↓
Progress and partial results returned
  ↓
Final result displayed
```

## 6.3 Polling Flow

Endpoint:

```http
GET /api/domain-checks/:jobId
```

Polling interval yang direkomendasikan:

```text
500–1000 ms
```

Polling berhenti ketika:

- Job `COMPLETED`.
- Job `FAILED`.
- Pengguna logout.
- Component di-unmount.
- Request dibatalkan.
- Terjadi terminal error.

---

## 7. Contract API Internal

## 7.1 Health Check

```http
GET /api/health
```

## 7.2 Login

```http
POST /api/auth/login
```

Request:

```json
{
  "username": "admin",
  "password": "secret"
}
```

Expected response:

```json
{
  "token": "<jwt-token>"
}
```

## 7.3 Create Domain Check Job

```http
POST /api/domain-checks
Authorization: Bearer <token>
```

Request:

```json
{
  "domains": [
    "example.com",
    "example.id"
  ]
}
```

Response:

```json
{
  "jobId": "generated-uuid",
  "status": "QUEUED",
  "total": 2
}
```

Recommended HTTP status:

```text
202 Accepted
```

## 7.4 Get Job Status

```http
GET /api/domain-checks/:jobId
Authorization: Bearer <token>
```

Response:

```json
{
  "jobId": "generated-uuid",
  "status": "PROCESSING",
  "progress": {
    "total": 2,
    "completed": 1,
    "percentage": 50
  },
  "summary": {
    "available": 1,
    "taken": 0,
    "error": 0
  },
  "results": [
    {
      "domain": "example.id",
      "status": "AVAILABLE",
      "checkedAt": "2026-07-15T03:30:00Z",
      "error": null
    }
  ]
}
```

Job status:

```text
QUEUED
PROCESSING
COMPLETED
FAILED
```

Domain status:

```text
AVAILABLE
TAKEN
ERROR
```

---

## 8. Domain Parsing dan Validation

Input mendukung separator:

- Newline.
- Comma.
- Semicolon.
- Whitespace.

Contoh:

```text
example.com
example.id
example.net
```

atau:

```text
example.com, example.id; example.net
```

Normalization:

- Trim whitespace.
- Lowercase.
- Remove `http://`.
- Remove `https://`.
- Remove path.
- Remove query string.
- Remove fragment.
- Remove trailing slash.
- Remove trailing dot secara aman.
- Remove duplicates.
- Validate domain syntax.

Validation rules:

- Maksimal 100 unique valid domains.
- Maksimal total panjang domain 253 karakter.
- Maksimal label 63 karakter.
- Tidak ada empty label.
- Label tidak boleh diawali atau diakhiri hyphen.
- Domain harus memiliki top-level domain.
- Invalid domain tidak boleh dianggap `TAKEN`.
- Backend validation menjadi authoritative source.

Edge cases yang dibahas:

- Empty input.
- Whitespace only.
- Duplicate.
- Duplicate setelah normalization.
- Uppercase.
- URL lengkap.
- URL dengan path.
- URL dengan query.
- URL dengan fragment.
- Consecutive dots.
- Missing TLD.
- Domain terlalu panjang.
- Label terlalu panjang.
- Unsupported scheme.
- Input memiliki port.
- Email address tidak sengaja dipaste.
- IDN atau Unicode domain.
- Unicode whitespace.

---

## 9. Strategi Concurrency

Go backend menggunakan bounded concurrency.

Rekomendasi default:

```text
5 concurrent WhoisFreaks requests
```

Concurrency dapat dikonfigurasi menggunakan environment variable:

```env
DOMAIN_CHECK_CONCURRENCY=5
```

Pattern yang dapat digunakan:

- Fixed worker pool.
- Buffered-channel semaphore.
- `errgroup` dengan limit.

Pendekatan yang dipilih harus:

- Tidak menjalankan 100 request secara uncontrolled.
- Tidak membuat 100 goroutine tanpa batas.
- Menggunakan shared reusable HTTP client.
- Menggunakan `context.Context`.
- Memiliki timeout per domain.
- Memiliki retry terbatas.
- Memiliki backoff.
- Tidak me-retry deterministic failure.
- Memastikan satu domain gagal tidak menggagalkan batch.
- Menjaga shared state menggunakan `sync.RWMutex`.
- Menghindari data race dan concurrent map access.

Testing concurrency:

```bash
go test -race ./...
```

---

## 10. In-Memory Job Manager

Job manager menyimpan:

- Job ID.
- Status.
- Total domain.
- Completed count.
- Result list.
- Summary.
- Created time.
- Updated time.
- Expiration time.
- Cancellation context bila digunakan.

Karakteristik:

- Tidak persisten.
- Single backend instance.
- Job hilang saat backend restart.
- Job hilang saat container restart.
- Expired job dibersihkan secara berkala.
- Cleanup interval configurable.
- Active job tidak boleh dibersihkan sebelum selesai.
- Job manager harus thread-safe.

Environment variables:

```env
JOB_RETENTION_MINUTES=30
JOB_CLEANUP_INTERVAL_MINUTES=5
```

---

## 11. Frontend UX

## 11.1 Login Page

Elemen:

- Application name.
- Username.
- Password.
- Sign in button.
- Loading state.
- Generic invalid credentials message.

Error:

```text
Invalid username or password.
```

## 11.2 Main Checker

Elemen utama:

- Compact header.
- Logout.
- Title.
- Description.
- Large textarea.
- Valid domain counter.
- Invalid domain summary.
- Duplicate summary.
- Check Availability button.

## 11.3 Processing State

Contoh:

```text
Checking domain availability...

45 of 100 domains checked

[██████████████░░░░░░░░░░░░░] 45%
```

Progress harus berdasarkan actual completed checks.

## 11.4 Result State

Summary cards:

- Total.
- Available.
- Taken.
- Error.

Filter:

- All.
- Available.
- Taken.
- Error.

Action:

- Search.
- Export CSV.
- Check New Domains.

---

## 12. CSV Export

CSV dibuat di frontend dari result yang telah tersedia.

Columns:

```csv
domain,status,checked_at,error_message
```

Fitur:

- Export all.
- Export filtered.
- Correct escaping untuk comma.
- Correct escaping untuk quotation marks.
- Correct escaping untuk line break.
- UTF-8 output.
- Optional UTF-8 BOM untuk spreadsheet compatibility.

Contoh filename:

```text
domain-availability-2026-07-15T10-30-00.csv
```

---

## 13. Environment Configuration

Environment variables utama:

```env
APP_ENV=development

BACKEND_PORT=8080
FRONTEND_PORT=3000

JWT_SECRET=replace-with-a-long-random-secret
JWT_EXPIRES_IN=8h

ADMIN_USERNAME=admin
ADMIN_PASSWORD_HASH=replace-with-a-bcrypt-hash

WHOISFREAKS_API_KEY=replace-with-the-api-key
WHOISFREAKS_BASE_URL=https://api.whoisfreaks.com/v1.0

DOMAIN_CHECK_CONCURRENCY=5
DOMAIN_CHECK_TIMEOUT_SECONDS=10
DOMAIN_CHECK_MAX_RETRIES=2

JOB_RETENTION_MINUTES=30
JOB_CLEANUP_INTERVAL_MINUTES=5

CORS_ALLOWED_ORIGINS=http://localhost:3000

CLOUDFLARE_TUNNEL_TOKEN=
PUBLIC_APP_URL=
```

Security principles:

- `.env` tidak boleh committed.
- `.env.example` harus tersedia.
- Tidak boleh ada real API key dalam source.
- Tidak boleh ada plaintext admin password.
- Tidak boleh ada Cloudflare token di repository.
- Frontend build args tidak boleh berisi secret.

---

## 14. Struktur Repository yang Direncanakan

```text
live-coding-rumahweb/
├── .env.example
├── AI_USAGE.md
├── ARCHITECTURE.md
├── PLANNING.md
├── GO_SETUP.md
├── PROMPT.md
├── postman/
│   ├── Bulk_Domain_Availability_Checker.postman_collection.json
│   ├── Bulk_Domain_Checker_Local.postman_environment.json
│   ├── Bulk_Domain_Checker_Cloudflare.postman_environment.json
│   └── README.md
└── code/
    ├── README.md
    ├── Makefile
    ├── docker-compose.yml
    ├── client/
    │   ├── Dockerfile
    │   ├── nginx.conf
    │   ├── package.json
    │   └── src/
    └── server/
        ├── Dockerfile
        ├── go.mod
        ├── go.sum
        ├── cmd/
        │   ├── api/
        │   └── hash-password/
        └── internal/
            ├── auth/
            ├── config/
            ├── domain/
            ├── httpapi/
            ├── jobs/
            ├── middleware/
            └── whoisfreaks/
```

---

## 15. Dokumentasi yang Direncanakan

## 15.1 `PLANNING.md`

Harus memuat:

- Scope summary.
- Task breakdown.
- Subtask.
- Dependency.
- Estimasi.
- Completion criteria.
- Minimal lima pertanyaan ke PM.
- Alasan setiap pertanyaan.
- Edge cases.
- Assumptions.

## 15.2 `ARCHITECTURE.md`

Harus memuat:

- Frontend-backend flow.
- Mermaid atau ASCII diagram.
- Technology choices.
- Folder structure.
- Concurrency strategy.
- Docker networking.
- Cloudflare routing.
- API contract.
- Security consideration.
- Trade-offs.

## 15.3 `AI_USAGE.md`

Diputuskan untuk tidak dimodifikasi oleh agent karena akan diisi sendiri oleh project owner.

## 15.4 `code/README.md`

Harus memuat:

- Setup.
- Local run.
- Docker run.
- Cloudflare run.
- Testing.
- Build.
- API.
- Password hash generation.
- Architecture decisions.
- Known limitations.
- Troubleshooting.

---

## 16. Agentic AI Prompt

Sebuah prompt implementasi lengkap telah disusun untuk agentic AI.

Prompt tersebut menginstruksikan agent agar:

- Memeriksa repository.
- Tidak membuat project di luar struktur.
- Tidak mengubah `AI_USAGE.md`.
- Mengisi `PLANNING.md`.
- Mengisi `ARCHITECTURE.md`.
- Mengimplementasikan frontend dan backend.
- Menggunakan Go.
- Menggunakan Docker Compose.
- Menggunakan cloudflared sebagai optional profile.
- Membuat test.
- Menjalankan build.
- Menjalankan race detector.
- Memvalidasi Docker Compose.
- Tidak meninggalkan TODO.
- Tidak menggunakan fake response.
- Tidak mengklaim test sukses jika tidak dijalankan.

Prompt awal menggunakan Node.js backend, lalu direvisi menjadi Go backend dan ditambahkan Docker Compose serta Cloudflare Tunnel.

---

## 17. Postman Collection

Postman Collection dan environments telah dibuat.

File yang dihasilkan:

```text
Bulk_Domain_Availability_Checker.postman_collection.json
Bulk_Domain_Checker_Local.postman_environment.json
Bulk_Domain_Checker_Cloudflare.postman_environment.json
README.md
```

Collection mencakup 13 requests.

Happy path:

1. Health check.
2. Login.
3. Save JWT.
4. Create batch.
5. Save job ID.
6. Poll status.
7. Stop ketika terminal state.

Negative tests:

- Wrong password.
- Missing login fields.
- Missing JWT.
- Invalid JWT.
- Empty domain list.
- Invalid domains.
- More than 100 domains.
- Job ID not found.

Collection variables:

- `baseUrl`
- `adminUsername`
- `adminPassword`
- `jwtToken`
- `jobId`
- `pollCount`
- `maxPollAttempts`

Local base URL:

```text
http://localhost:3000
```

Cloudflare base URL:

```text
https://your-domain.example.com
```

Collection Runner digunakan untuk workflow polling otomatis.

---

## 18. Setup Go

Setup Go kemudian dirapikan secara menyeluruh.

Keputusan:

- Go version dipin agar konsisten.
- `.go-version` ditambahkan.
- Docker builder dipin.
- Local development mendukung Windows.
- Environment root dapat dibaca meskipun API dijalankan dari subfolder.
- OS environment variable tetap memiliki prioritas di atas `.env`.

Masalah penting yang ditemukan:

```text
.env di root tidak otomatis terbaca jika menjalankan:
cd code/server
go run ./cmd/api
```

Solusi yang diterapkan:

- Config loader mencari `.env` dari beberapa working directory.
- Script setup Windows dibuat.
- Script local development dibuat.
- Environment validation diperketat.

Environment yang divalidasi:

- Port.
- JWT secret.
- Bcrypt hash.
- WhoisFreaks API key.
- WhoisFreaks URL.
- Concurrency.
- Timeout.
- Retry.
- Job retention.
- Cleanup interval.

---

## 19. Perbaikan Backend yang Dibahas

Perbaikan teknis yang disebutkan selama setup:

- Membaca response field `availability`.
- Fallback ke `domain_availability`.
- Menggunakan cleanup interval dari environment.
- Menambahkan cancellation pada job manager.
- Menambahkan graceful shutdown.
- Memperbaiki login rate limiter agar berdasarkan IP tanpa random client port.
- CORS mendukung comma-separated origins.
- Sanitized logging.
- Tidak me-log URL upstream yang berisi API key.
- Reusable HTTP client.
- Context propagation.
- Request timeout.
- Panic recovery.
- Body size limit.
- HTTP server timeouts.

---

## 20. Docker dan Nginx

## 20.1 Backend Dockerfile

Arah implementasi:

- Multi-stage build.
- Go builder image.
- Download module dependencies.
- Compile binary.
- Minimal runtime image.
- Non-root user jika praktis.
- Health support.
- Tidak membawa source dan build cache ke runtime.

## 20.2 Frontend Dockerfile

Arah implementasi:

- Node builder.
- npm install.
- Vite build.
- Nginx runtime.
- SPA fallback.
- Proxy `/api`.
- Cache static assets.
- Jangan terlalu agresif cache `index.html`.

## 20.3 Docker Compose

Services:

```yaml
services:
  backend:
  frontend:
  cloudflared:
```

Cloudflared dibuat sebagai optional profile:

```bash
docker compose --profile tunnel up --build
```

Normal run:

```bash
docker compose up --build
```

Validation:

```bash
docker compose config
docker compose build
```

---

## 21. Go Development Commands

Setup Windows:

```powershell
Set-ExecutionPolicy -Scope Process Bypass

.\scripts\setup-windows.ps1 `
  -AdminPassword "ganti-dengan-password-kuat"
```

Menjalankan development:

```powershell
.\scripts\dev-windows.ps1
```

Backend:

```bash
cd code/server
go mod download
go run ./cmd/api
```

Testing:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Build:

```bash
go build ./cmd/api
```

Frontend:

```bash
cd code/client
npm install
npm run test
npm run build
```

---

## 22. Reported Validation Result

Berdasarkan proses sebelumnya, reported validation mencakup:

```text
Go unit tests          passed
Go race detector       passed
go vet                 passed
Go production build    passed
API health smoke test  passed
JWT login smoke test   passed
Invalid-login test     passed
Frontend tests         passed
Frontend build         passed
```

Docker runtime tidak dapat dijalankan di environment saat itu karena Docker Engine tidak tersedia.

Namun:

- Compose YAML diparse.
- Service frontend tersedia.
- Service backend tersedia.
- Service cloudflared tersedia.

Catatan:

Hasil ini perlu diverifikasi ulang di mesin target yang memiliki Docker Engine dan real WhoisFreaks API key.

---

## 23. Security Checklist

- JWT secret minimum length.
- bcrypt password hash.
- Generic login errors.
- Login rate limiting.
- Protected batch endpoints.
- No frontend API key.
- No plaintext password.
- No secret in source.
- No Cloudflare token in Compose file.
- `.env` ignored.
- API key not present in error.
- API key not present in logs.
- Request body size limit.
- CORS restriction.
- Security headers.
- Panic recovery.
- Non-root container where practical.
- Backend not directly public.
- Cloudflare Tunnel points to frontend.
- Cloudflare Access optional.
- TLS handled by Cloudflare when tunnel is active.

---

## 24. Testing Checklist

### Backend

- Successful login.
- Failed login.
- Missing JWT.
- Invalid JWT.
- Expired JWT.
- Empty batch.
- More than 100 domains.
- URL normalization.
- Duplicate normalization.
- Uppercase normalization.
- Invalid domain.
- AVAILABLE mapping.
- UNAVAILABLE mapping.
- Unknown status.
- Timeout.
- Retry transient failure.
- No retry deterministic failure.
- Partial failure.
- Progress calculation.
- Concurrent safety.
- Job not found.
- Expired job cleanup.
- API key redaction.
- Malformed upstream JSON.
- Health endpoint.

### Frontend

- Domain parser.
- Duplicate removal.
- Filter.
- Search.
- CSV escaping.
- Progress display.
- Login errors.
- Polling completion.
- Polling cancellation.
- Empty filter result.

### Deployment

- `docker compose config`.
- `docker compose build`.
- Health checks.
- Frontend-to-backend proxy.
- Cloudflared profile.
- Backend internal exposure only.

---

## 25. Known Limitations

- Job state hanya berada di memory.
- Job hilang ketika backend restart.
- Tidak mendukung horizontal scaling tanpa shared state.
- Browser refresh dapat kehilangan local UI state.
- Polling menghasilkan request periodik.
- Cloudflare Tunnel bergantung pada external service.
- Tidak ada audit history.
- Tidak ada multi-user management.
- Tidak ada persistent result history.
- Tidak ada automatic resumption setelah restart.
- WhoisFreaks limits bergantung pada plan yang digunakan.

Future improvements:

- Redis untuk shared job state.
- Queue system.
- Multi-instance backend.
- Persistent history.
- Audit log.
- Cloudflare Access.
- Role-based user management.
- SSE atau WebSocket.
- Bulk API jika vendor menyediakan progress yang sesuai.
- Metrics dan observability.

---

## 26. Definition of Done

Aplikasi dianggap selesai ketika:

1. Admin dapat login.
2. Password diverifikasi dengan bcrypt.
3. JWT diterbitkan dan diverifikasi.
4. Endpoint protected menolak unauthorized request.
5. API key tidak tampil di browser.
6. 1–100 domain dapat diproses.
7. Input dinormalisasi.
8. Duplicate dihapus.
9. Invalid domain ditampilkan.
10. Concurrency dibatasi.
11. UI tidak freeze.
12. Create-job request tidak menunggu semua domain.
13. Progress aktual tersedia.
14. Partial failure tidak menggagalkan batch.
15. Status dinormalisasi.
16. Hasil dapat difilter.
17. Hasil dapat dicari.
18. CSV all dapat diekspor.
19. CSV filtered dapat diekspor.
20. Tidak ada database.
21. Expired job dibersihkan.
22. Graceful shutdown tersedia.
23. Frontend disajikan oleh Nginx.
24. `/api` diproxy ke backend.
25. Docker Compose berjalan.
26. Cloudflared optional profile tersedia.
27. Dokumentasi lengkap.
28. Tests pass.
29. Race detector pass.
30. Builds pass.
31. No real secrets.
32. No unfinished TODO.
33. Bisa dijalankan berdasarkan README.
34. Bisa dipublish menggunakan Cloudflare Tunnel.

---

## 27. Artifact yang Dihasilkan

Selama interaksi, artifact yang dibahas atau dibuat mencakup:

- Prompt implementasi agentic AI.
- Prompt revisi Go + Docker Compose + Cloudflare Tunnel.
- Postman Collection.
- Local Postman Environment.
- Cloudflare Postman Environment.
- Postman README.
- Repository ZIP dengan folder Postman.
- Go-ready repository ZIP.
- `GO_SETUP.md`.
- `PLANNING.md`.
- `ARCHITECTURE.md`.
- `PROMPT.md`.
- Docker Compose specification.
- Cloudflare Tunnel profile specification.

---

## 28. Status Terakhir

Status terakhir proyek:

- Requirement telah dikunci.
- Stack telah dikunci.
- Arsitektur telah dikunci.
- API contract telah didefinisikan.
- Postman contract test telah disiapkan.
- Go setup telah disiapkan.
- Docker Compose telah dimasukkan ke rancangan.
- Cloudflare Tunnel telah dimasukkan sebagai optional deployment path.
- Dokumentasi utama telah dirancang.
- Backend dan frontend dilaporkan telah melewati test dan build.
- Real end-to-end test masih perlu dilakukan di mesin target menggunakan:
  - Docker Engine.
  - WhoisFreaks API key asli.
  - Cloudflare Tunnel token jika publikasi eksternal diperlukan.

---

## 29. Langkah Berikutnya yang Direkomendasikan

1. Extract repository Go-ready.
2. Install Go, Node.js, npm, dan Docker Desktop.
3. Copy `.env.example` menjadi `.env`.
4. Generate JWT secret.
5. Generate bcrypt hash.
6. Isi WhoisFreaks API key.
7. Jalankan backend dan frontend secara lokal.
8. Import Postman Collection.
9. Jalankan Happy Path folder.
10. Periksa progress dan hasil.
11. Jalankan negative tests.
12. Jalankan race detector.
13. Jalankan Docker Compose.
14. Verifikasi health check.
15. Verifikasi Nginx proxy.
16. Aktifkan Cloudflare Tunnel.
17. Verifikasi public URL.
18. Pastikan backend tidak langsung public.
19. Review `PLANNING.md`.
20. Review `ARCHITECTURE.md`.
21. Lengkapi `AI_USAGE.md`.
22. Dokumentasikan final result untuk submission.

---

## 30. Kesimpulan

Proyek berkembang dari ide sederhana berupa bulk domain checker menjadi aplikasi internal yang memiliki desain implementasi cukup matang.

Keputusan penting yang berhasil dikunci:

- Go dipilih untuk backend concurrent processing.
- React digunakan untuk UX interaktif.
- Database tidak digunakan karena result bersifat transient.
- Job diproses secara asynchronous.
- Progress dikirim melalui polling.
- Bounded goroutine worker pool digunakan untuk keamanan resource.
- Docker Compose digunakan untuk prototyping dan deployment.
- Cloudflare Tunnel digunakan untuk optional publishing.
- Postman digunakan sebagai executable API contract.
- Security dan testability dipertimbangkan sejak awal.

Arsitektur ini cukup sederhana untuk live-coding atau prototyping, tetapi masih memiliki fondasi yang sehat untuk dikembangkan ke deployment internal yang lebih serius.
