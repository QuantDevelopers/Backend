package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Settings holds application, database, CORS, and crawler configuration.
type Settings struct {
	AppName  string
	AppHost  string
	AppPort  int
	Debug    bool
	RootPath string

	DatabaseURL   string
	DBEcho        bool
	DBPoolSize    int
	DBMaxOverflow int

	CORSOrigins []string

	CrawlerRateLimitRPS  float64
	CrawlerMaxRetries    int
	CrawlerBackoffFactor float64
	CrawlerRawDataPath   string
}

// Address returns the host:port bind address for the HTTP server.
func (s Settings) Address() string {
	return fmt.Sprintf("%s:%d", s.AppHost, s.AppPort)
}

// MaxPoolConns computes the total connection pool limit for pgxpool.
func (s Settings) MaxPoolConns() int32 {
	total := s.DBPoolSize + s.DBMaxOverflow
	if total <= 0 {
		return 10
	}
	return int32(total)
}

// Load reads configuration from an optional .env file and environment variables.
func Load() Settings {
	// Best-effort load; missing .env is valid in containerized and CI runs.
	_ = godotenv.Load()

	rawDBURL := getEnv("DATABASE_URL", "")
	if rawDBURL == "" {
		pgUser := getEnv("POSTGRES_USER", "postgres")
		pgPass := getEnv("POSTGRES_PASSWORD", "postgres")
		pgHost := getEnv("POSTGRES_HOST", "postgres")
		pgPort := getEnv("POSTGRES_PORT", "5432")
		pgDB := getEnv("POSTGRES_DB", "quant_db")
		rawDBURL = fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable", pgUser, pgPass, pgHost, pgPort, pgDB)
	}

	return Settings{
		AppName:  getEnv("APP_NAME", "QuantBackend"),
		AppHost:  getEnv("APP_HOST", "0.0.0.0"),
		AppPort:  getEnvInt("APP_PORT", 8000),
		Debug:    getEnvBool("DEBUG", true),
		RootPath: normalizeRootPath(getEnv("ROOT_PATH", "")),

		DatabaseURL:   NormalizeDatabaseURL(rawDBURL),
		DBEcho:        getEnvBool("DB_ECHO", false),
		DBPoolSize:    getEnvInt("DB_POOL_SIZE", 10),
		DBMaxOverflow: getEnvInt("DB_MAX_OVERFLOW", 20),

		CORSOrigins: parseCommaSeparated(getEnv(
			"CORS_ORIGINS",
			"http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173",
		)),

		CrawlerRateLimitRPS:  getEnvFloat("CRAWLER_RATE_LIMIT_RPS", 3.0),
		CrawlerMaxRetries:    getEnvInt("CRAWLER_MAX_RETRIES", 5),
		CrawlerBackoffFactor: getEnvFloat("CRAWLER_BACKOFF_FACTOR", 1.5),
		CrawlerRawDataPath:   getEnv("CRAWLER_RAW_DATA_PATH", "./data/raw"),
	}
}

// NormalizeDatabaseURL converts Python SQLAlchemy asyncpg DSN prefixes into standard PostgreSQL URLs.
func NormalizeDatabaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Replace(trimmed, "postgresql+asyncpg://", "postgresql://", 1)
	trimmed = strings.Replace(trimmed, "postgres+asyncpg://", "postgres://", 1)
	return trimmed
}

func normalizeRootPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func parseCommaSeparated(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(val))
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	val, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(val) == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil {
		return fallback
	}
	return parsed
}
