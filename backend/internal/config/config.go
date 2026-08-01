package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process configuration loaded from the environment.
type Config struct {
	// Addr is the HTTP listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a Postgres connection string. Empty means no DB.
	DatabaseURL string
	// AuthJWTSecret signs access tokens. Empty disables auth routes (503).
	AuthJWTSecret string
	// AuthTokenTTL is access token lifetime.
	AuthTokenTTL time.Duration
	// CORSAllowedOrigins is the Origin allowlist (comma-separated env).
	CORSAllowedOrigins []string
	// CookieSecure marks the session cookie Secure (HTTPS-only).
	CookieSecure bool
	// CookieDomain optionally scopes the session cookie to a domain.
	CookieDomain string
	// KafkaBrokers is the Kafka broker list (comma-separated env). Empty
	// disables the event bus (api keeps working; pushes are not produced).
	KafkaBrokers []string
	// VAPIDPublicKey is the Web Push VAPID public key (base64url).
	VAPIDPublicKey string
	// VAPIDPrivateKey is the Web Push VAPID private key (base64url).
	VAPIDPrivateKey string
	// PushSubject is the VAPID subject (mailto: or URL) in push requests.
	PushSubject string
	// PushAggregateWindow is the offline-push aggregation window.
	PushAggregateWindow time.Duration
	// MetricsAddr is the Prometheus metrics listen address. Empty disables the
	// metrics server (process behavior is unchanged).
	MetricsAddr string
}

// Load reads configuration from environment variables.
func Load() Config {
	port := 8080
	if raw := os.Getenv("PORT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n < 65536 {
			port = n
		}
	}

	ttl := 24 * time.Hour // 1 day
	if raw := strings.TrimSpace(os.Getenv("AUTH_TOKEN_TTL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			ttl = d
		}
	}

	aggWindow := 2 * time.Second
	if raw := strings.TrimSpace(os.Getenv("PUSH_AGGREGATE_WINDOW")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			aggWindow = d
		}
	}

	var kafkaBrokers []string
	for _, b := range strings.Split(os.Getenv("KAFKA_BROKERS"), ",") {
		b = strings.TrimSpace(b)
		if b != "" {
			kafkaBrokers = append(kafkaBrokers, b)
		}
	}
	// Dev default matches docker-compose host mapping.
	if len(kafkaBrokers) == 0 {
		kafkaBrokers = []string{"localhost:19092"}
	}

	secret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	// Dev convenience: allow insecure default only when explicitly opted in.
	if secret == "" && os.Getenv("AUTH_DEV_INSECURE") == "1" {
		secret = "easyim-dev-secret-change-me"
	}

	var allowed []string
	for _, o := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed = append(allowed, o)
		}
	}
	if len(allowed) == 0 {
		allowed = []string{"http://localhost:5173"} // dev frontend
	}

	return Config{
		Addr:                ":" + strconv.Itoa(port),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AuthJWTSecret:       secret,
		AuthTokenTTL:        ttl,
		CORSAllowedOrigins:  allowed,
		CookieSecure:        os.Getenv("COOKIE_SECURE") == "1",
		CookieDomain:        strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		KafkaBrokers:        kafkaBrokers,
		VAPIDPublicKey:      strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY")),
		VAPIDPrivateKey:     strings.TrimSpace(os.Getenv("VAPID_PRIVATE_KEY")),
		PushSubject:         strings.TrimSpace(os.Getenv("PUSH_SUBJECT")),
		PushAggregateWindow: aggWindow,
		MetricsAddr:         strings.TrimSpace(os.Getenv("METRICS_ADDR")),
	}
}
