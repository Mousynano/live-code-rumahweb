package main

import (
	"bulk-domain-checker/server/internal/auth"
	"bulk-domain-checker/server/internal/domain"
	"bulk-domain-checker/server/internal/jobs"
	"bulk-domain-checker/server/internal/whois"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type config struct {
	port, username, hash, secret, base, key, cors string
	ttl, timeOut, retention                       time.Duration
	retries, concurrency                          int
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envInt(k string, d int) int {
	v, e := strconv.Atoi(env(k, strconv.Itoa(d)))
	if e != nil || v < 1 {
		return d
	}
	return v
}
func load() (config, error) {
	ttl, e := time.ParseDuration(env("JWT_EXPIRES_IN", "8h"))
	if e != nil {
		return config{}, e
	}
	c := config{env("BACKEND_PORT", "8080"), env("ADMIN_USERNAME", "admin"), os.Getenv("ADMIN_PASSWORD_HASH"), os.Getenv("JWT_SECRET"), env("WHOISFREAKS_BASE_URL", "https://api.whoisfreaks.com/v1.0"), os.Getenv("WHOISFREAKS_API_KEY"), env("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ttl, time.Duration(envInt("DOMAIN_CHECK_TIMEOUT_SECONDS", 10)) * time.Second, time.Duration(envInt("JOB_RETENTION_MINUTES", 30)) * time.Minute, envInt("DOMAIN_CHECK_MAX_RETRIES", 2), envInt("DOMAIN_CHECK_CONCURRENCY", 5)}
	if len(c.secret) < 32 || c.hash == "" || c.key == "" {
		return config{}, fmt.Errorf("JWT_SECRET (32+ chars), ADMIN_PASSWORD_HASH, and WHOISFREAKS_API_KEY are required")
	}
	return c, nil
}

type limiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func main() {
	c, err := load()
	if err != nil {
		log.Fatal(err)
	}
	a := auth.New(c.username, c.hash, c.secret, c.ttl)
	m := jobs.New(whois.New(c.base, c.key, c.timeOut, c.retries), c.concurrency, c.retention, time.Duration(envInt("JOB_CLEANUP_INTERVAL_MINUTES", 5))*time.Minute)
	defer m.Close()
	l := &limiter{attempts: map[string][]time.Time{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { jsonOut(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		l.mu.Lock()
		now := time.Now()
		recentAttempts := make([]time.Time, 0, 10)
		for _, t := range l.attempts[ip] {
			if now.Sub(t) < time.Minute {
				recentAttempts = append(recentAttempts, t)
			}
		}
		recentAttempts = append(recentAttempts, now)
		l.attempts[ip] = recentAttempts
		recent := len(recentAttempts)
		l.mu.Unlock()
		if recent > 10 {
			errOut(w, 429, "RATE_LIMITED", "Too many login attempts.", nil)
			return
		}
		var body struct{ Username, Password string }
		if !decode(w, r, &body) {
			return
		}
		token, exp, e := a.Login(body.Username, body.Password)
		if e != nil {
			errOut(w, 401, "INVALID_CREDENTIALS", "Invalid username or password.", nil)
			return
		}
		jsonOut(w, 200, map[string]any{"token": token, "expiresAt": exp})
	})
	mux.HandleFunc("POST /api/domain-checks", func(w http.ResponseWriter, r *http.Request) {
		if !protect(w, r, a) {
			return
		}
		var body struct {
			Domains []string `json:"domains"`
		}
		if !decode(w, r, &body) {
			return
		}
		p := domain.Parse(body.Domains)
		if p.Detected == 0 || len(p.Invalid) > 0 || len(p.Domains) == 0 || len(p.Domains) > domain.MaxBatch {
			msg := "Provide between 1 and 100 valid domains."
			errOut(w, 422, "VALIDATION_ERROR", msg, p)
			return
		}
		s := m.Create(p.Domains)
		jsonOut(w, 202, map[string]any{"jobId": s.JobID, "status": s.Status, "total": s.Progress.Total})
	})
	mux.HandleFunc("GET /api/domain-checks/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !protect(w, r, a) {
			return
		}
		s, ok := m.Get(r.PathValue("id"))
		if !ok {
			errOut(w, 404, "JOB_NOT_FOUND", "Job was not found or has expired.", nil)
			return
		}
		jsonOut(w, 200, s)
	})
	h := middleware(mux, c.cors)
	srv := &http.Server{Addr: ":" + c.port, Handler: h, ReadTimeout: 10 * time.Second, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		log.Printf("API listening on :%s", c.port)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatal(e)
		}
	}()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
func protect(w http.ResponseWriter, r *http.Request, a *auth.Service) bool {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") || a.Verify(strings.TrimPrefix(h, "Bearer ")) != nil {
		errOut(w, 401, "UNAUTHORIZED", "Authentication is required.", nil)
		return false
	}
	return true
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if json.NewDecoder(r.Body).Decode(v) != nil {
		errOut(w, 400, "INVALID_JSON", "Invalid request body.", nil)
		return false
	}
	return true
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func errOut(w http.ResponseWriter, status int, code, msg string, details any) {
	jsonOut(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg, "details": details}})
}
func middleware(next http.Handler, cors string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Access-Control-Allow-Origin", cors)
		w.Header().Set("Vary", "Origin")
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.WriteHeader(204)
			return
		}
		defer func() {
			if recover() != nil {
				errOut(w, 500, "INTERNAL_ERROR", "Unexpected server error.", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
