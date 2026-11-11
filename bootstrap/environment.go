package bootstrap

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Env struct {
	Server     Server
	Logger     Logger
	RateLimit  RateLimit
	PrimaryDB  Database
	Pagination Pagination
	CORS       CORS
	Crawler    Crawler
}

type Server struct {
	AppName  string
	Host     string
	Port     string
	Mode     string
	Debug    bool
	RootPath string
}

func (s Server) Address() string {
	return fmt.Sprintf("%s:%s", s.Host, s.Port)
}

type Logger struct {
	LogLevel      string
	LogFile       string
	ConsoleOutput string
}

type RateLimit struct {
	Limit int
	Burst int
}

type Database struct {
	URL         string
	Host        string
	Port        string
	User        string
	Password    string
	Name        string
	Echo        bool
	PoolSize    int
	MaxOverflow int
}

func (d Database) DSN() string {
	if d.URL != "" {
		return d.URL
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		d.Host, d.User, d.Password, d.Name, d.Port,
	)
}

type Pagination struct {
	DefaultPage     int
	DefaultPageSize int
}

type CORS struct {
	AllowOrigins []string
}

type Crawler struct {
	RateLimitRPS  float64
	MaxRetries    int
	BackoffFactor float64
	RawDataPath   string
}

func NewEnvironments() *Env {
	_ = godotenv.Load(".env")

	debug := getEnvBool("DEBUG", true)
	defaultLogLevel := "info"
	if debug {
		defaultLogLevel = "debug"
	}

	port := getEnvFirst([]string{"SERVER_PORT", "APP_PORT"}, "8000")
	host := getEnvFirst([]string{"SERVER_HOST", "APP_HOST"}, "0.0.0.0")
	mode := getEnvFirst([]string{"SERVER_MODE"}, "development")
	if !debug && mode == "development" {
		mode = "production"
	}

	dbHost := getEnvFirst([]string{"DB_HOST", "POSTGRES_HOST"}, "postgres")
	dbPort := getEnvFirst([]string{"DB_PORT", "POSTGRES_PORT"}, "5432")
	dbUser := getEnvFirst([]string{"DB_USER", "POSTGRES_USER"}, "postgres")
	dbPass := getEnvFirst([]string{"DB_PASSWORD", "POSTGRES_PASSWORD"}, "postgres")
	dbName := getEnvFirst([]string{"DB_NAME", "POSTGRES_DB"}, "quant_db")
	dbURL := NormalizeDatabaseURL(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		dbURL = fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
			url.QueryEscape(dbUser),
			url.QueryEscape(dbPass),
			dbHost,
			dbPort,
			dbName,
		)
	}

	return &Env{
		Server: Server{
			AppName:  getEnvFirst([]string{"APP_NAME"}, "QuantBackend"),
			Host:     host,
			Port:     port,
			Mode:     mode,
			Debug:    debug,
			RootPath: normalizeRootPath(os.Getenv("ROOT_PATH")),
		},
		Logger: Logger{
			LogLevel:      getEnvFirst([]string{"LOG_LEVEL"}, defaultLogLevel),
			LogFile:       os.Getenv("LOG_FILE"),
			ConsoleOutput: getEnvFirst([]string{"CONSOLE_OUTPUT"}, "true"),
		},
		RateLimit: RateLimit{
			Limit: getEnvInt("RATE_LIMIT", 60),
			Burst: getEnvIntFirst([]string{"RATE_LIMIT_BURST", "RATE_lIMIT_BURST"}, 120),
		},
		PrimaryDB: Database{
			URL:         dbURL,
			Host:        dbHost,
			Port:        dbPort,
			User:        dbUser,
			Password:    dbPass,
			Name:        dbName,
			Echo:        getEnvBool("DB_ECHO", false),
			PoolSize:    getEnvInt("DB_POOL_SIZE", 10),
			MaxOverflow: getEnvInt("DB_MAX_OVERFLOW", 20),
		},
		Pagination: Pagination{
			DefaultPage:     getEnvInt("DEFAULT_PAGE", 1),
			DefaultPageSize: getEnvInt("DEFAULT_PAGE_SIZE", 25),
		},
		CORS: CORS{
			AllowOrigins: parseCommaSeparated(getEnvFirst(
				[]string{"CORS_ORIGINS"},
				"http://localhost:3000,http://localhost:5173,http://127.0.0.1:3000,http://127.0.0.1:5173",
			)),
		},
		Crawler: Crawler{
			RateLimitRPS:  getEnvFloat("CRAWLER_RATE_LIMIT_RPS", 3.0),
			MaxRetries:    getEnvInt("CRAWLER_MAX_RETRIES", 5),
			BackoffFactor: getEnvFloat("CRAWLER_BACKOFF_FACTOR", 1.5),
			RawDataPath:   getEnvFirst([]string{"CRAWLER_RAW_DATA_PATH"}, "./data/raw"),
		},
	}
}

// NormalizeDatabaseURL converts SQLAlchemy asyncpg prefixes into standard PostgreSQL DSNs.
func NormalizeDatabaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
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

func getEnvFirst(keys []string, fallback string) string {
	for _, k := range keys {
		if val, ok := os.LookupEnv(k); ok && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	return getEnvIntFirst([]string{key}, fallback)
}

func getEnvIntFirst(keys []string, fallback int) int {
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
			if parsed, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(val)); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
			return parsed
		}
	}
	return fallback
}
