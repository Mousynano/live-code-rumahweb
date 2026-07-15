package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Config contains every runtime setting used by the API. Environment variables
// always take precedence over values loaded from a local .env file.
type Config struct {
	AppEnv            string
	Port              string
	AdminUsername     string
	AdminPasswordHash string
	JWTSecret         string
	JWTExpiresIn      time.Duration

	WhoisFreaksBaseURL string
	WhoisFreaksAPIKey  string

	DomainCheckConcurrency int
	DomainCheckTimeout     time.Duration
	DomainCheckMaxRetries  int

	JobRetention       time.Duration
	JobCleanupInterval time.Duration

	CORSAllowedOrigins []string
}

// Load reads an optional repository-local .env file and then validates all
// runtime configuration. It supports starting the API from the repository root,
// code/, or code/server/ without requiring the user to export variables first.
func Load() (Config, error) {
	if _, err := loadLocalDotEnv(); err != nil {
		return Config{}, fmt.Errorf("load local .env: %w", err)
	}

	jwtTTL, err := durationEnv("JWT_EXPIRES_IN", "8h")
	if err != nil {
		return Config{}, err
	}
	checkTimeout, err := positiveSecondsEnv("DOMAIN_CHECK_TIMEOUT_SECONDS", 10)
	if err != nil {
		return Config{}, err
	}
	retention, err := positiveMinutesEnv("JOB_RETENTION_MINUTES", 30)
	if err != nil {
		return Config{}, err
	}
	cleanupInterval, err := positiveMinutesEnv("JOB_CLEANUP_INTERVAL_MINUTES", 5)
	if err != nil {
		return Config{}, err
	}
	concurrency, err := intEnv("DOMAIN_CHECK_CONCURRENCY", 5, 1, 100)
	if err != nil {
		return Config{}, err
	}
	retries, err := intEnv("DOMAIN_CHECK_MAX_RETRIES", 2, 0, 10)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:                 env("APP_ENV", "development"),
		Port:                   env("BACKEND_PORT", "8080"),
		AdminUsername:          env("ADMIN_USERNAME", "admin"),
		AdminPasswordHash:      strings.TrimSpace(os.Getenv("ADMIN_PASSWORD_HASH")),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		JWTExpiresIn:           jwtTTL,
		WhoisFreaksBaseURL:     strings.TrimRight(env("WHOISFREAKS_BASE_URL", "https://api.whoisfreaks.com/v1.0"), "/"),
		WhoisFreaksAPIKey:      strings.TrimSpace(os.Getenv("WHOISFREAKS_API_KEY")),
		DomainCheckConcurrency: concurrency,
		DomainCheckTimeout:     checkTimeout,
		DomainCheckMaxRetries:  retries,
		JobRetention:           retention,
		JobCleanupInterval:     cleanupInterval,
		CORSAllowedOrigins:     csvEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("BACKEND_PORT must be an integer between 1 and 65535")
	}
	if strings.TrimSpace(c.AdminUsername) == "" {
		return errors.New("ADMIN_USERNAME is required")
	}
	if _, err := bcrypt.Cost([]byte(c.AdminPasswordHash)); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD_HASH must be a valid bcrypt hash: %w", err)
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must contain at least 32 characters")
	}
	if c.WhoisFreaksAPIKey == "" {
		return errors.New("WHOISFREAKS_API_KEY is required")
	}
	baseURL, err := url.Parse(c.WhoisFreaksBaseURL)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return errors.New("WHOISFREAKS_BASE_URL must be a valid HTTP or HTTPS URL")
	}
	if len(c.CORSAllowedOrigins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	return nil
}

func loadLocalDotEnv() (string, error) {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			continue
		}
		if err := loadDotEnvFile(candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}
	return "", nil
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\uFEFF"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected KEY=VALUE", path, lineNumber)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fmt.Errorf("%s:%d: empty environment key", path, lineNumber)
		}
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			if value[0] == '"' {
				decoded, err := strconv.Unquote(value)
				if err != nil {
					return fmt.Errorf("%s:%d: invalid quoted value: %w", path, lineNumber, err)
				}
				value = decoded
			} else {
				value = value[1 : len(value)-1]
			}
		}
		if _, alreadySet := os.LookupEnv(key); !alreadySet {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s: %w", key, err)
			}
		}
	}
	return scanner.Err()
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func durationEnv(key, fallback string) (time.Duration, error) {
	value := env(key, fallback)
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration such as 8h or 30m", key)
	}
	return duration, nil
}

func positiveSecondsEnv(key string, fallback int) (time.Duration, error) {
	value, err := intEnv(key, fallback, 1, 3600)
	if err != nil {
		return 0, err
	}
	return time.Duration(value) * time.Second, nil
}

func positiveMinutesEnv(key string, fallback int) (time.Duration, error) {
	value, err := intEnv(key, fallback, 1, 24*60)
	if err != nil {
		return 0, err
	}
	return time.Duration(value) * time.Minute, nil
}

func intEnv(key string, fallback, minimum, maximum int) (int, error) {
	raw := env(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", key, minimum, maximum)
	}
	return value, nil
}

func csvEnv(key, fallback string) []string {
	values := strings.Split(env(key, fallback), ",")
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
