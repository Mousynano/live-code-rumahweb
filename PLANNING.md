# Implementation plan

## Scope and assumptions

This is a single-instance internal tool for one administrator. Jobs, results, and rate-limit counters intentionally live only in backend memory; restarting the process discards them. Input is limited to ASCII hostnames with a two-or-more-character alphabetic TLD. Internationalized domains are rejected explicitly rather than silently misprocessed.

The production frontend is served by Nginx and calls `/api`; Nginx proxies that path to the private Go container. Local development uses Vite on port 3000 and Go on 8080. WhoisFreaks is integrated through the backend only, with the documented availability endpoint and defensive response mapping.

## Delivery plan

1. Implement configuration, authentication, parsing, upstream client, and asynchronous bounded job processing in Go.
2. Add protected HTTP endpoints, secure defaults, health check, graceful shutdown, and unit/integration tests.
3. Build the React/TypeScript interface with local validation, progress polling, filters, search, and CSV export.
4. Add Docker, Nginx, Compose, Cloudflare Tunnel profile, developer commands, and complete architecture/setup documentation.
5. Format, test, build, and validate configurations available in this environment.

## Design decisions

* A fixed worker pool (default five) limits upstream load and guarantees progress is updated only after a check ends.
* Initial batch submission returns a UUID immediately; the browser polls every 750ms and cancels polling when terminal, logged out, or unmounted.
* Bcrypt hash comparison and signed, expiring JWTs are used. The browser keeps the token in `sessionStorage` to limit persistence, accepting the normal XSS exposure trade-off documented in architecture.
* Transient upstream failures retry with short exponential backoff; malformed, most 4xx, and response-schema failures do not retry.
