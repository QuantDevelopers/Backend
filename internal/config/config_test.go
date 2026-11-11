package config

import (
	"testing"
)

func TestNormalizeDatabaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "sqlalchemy asyncpg postgresql prefix",
			in:   "postgresql+asyncpg://postgres:postgres@localhost:5432/quant_db",
			want: "postgresql://postgres:postgres@localhost:5432/quant_db",
		},
		{
			name: "sqlalchemy asyncpg postgres prefix",
			in:   "postgres+asyncpg://user:pass@db:5432/app",
			want: "postgres://user:pass@db:5432/app",
		},
		{
			name: "already standard postgresql url",
			in:   "postgresql://postgres:postgres@postgres:5432/quant_db?sslmode=disable",
			want: "postgresql://postgres:postgres@postgres:5432/quant_db?sslmode=disable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeDatabaseURL(tc.in); got != tc.want {
				t.Fatalf("NormalizeDatabaseURL(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("APP_NAME", "QuantTest")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("ROOT_PATH", "api/quant/")
	t.Setenv("DATABASE_URL", "postgresql+asyncpg://u:p@localhost:5432/testdb")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000, https://example.com ")
	t.Setenv("DB_POOL_SIZE", "5")
	t.Setenv("DB_MAX_OVERFLOW", "10")

	cfg := Load()

	if cfg.AppName != "QuantTest" {
		t.Fatalf("AppName = %q, want QuantTest", cfg.AppName)
	}
	if cfg.AppPort != 9090 || cfg.Address() != "0.0.0.0:9090" {
		t.Fatalf("Address = %q, want 0.0.0.0:9090", cfg.Address())
	}
	if cfg.RootPath != "/api/quant" {
		t.Fatalf("RootPath = %q, want /api/quant", cfg.RootPath)
	}
	if cfg.DatabaseURL != "postgresql://u:p@localhost:5432/testdb" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.MaxPoolConns() != 15 {
		t.Fatalf("MaxPoolConns = %d, want 15", cfg.MaxPoolConns())
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "http://localhost:3000" || cfg.CORSOrigins[1] != "https://example.com" {
		t.Fatalf("unexpected CORSOrigins: %#v", cfg.CORSOrigins)
	}
}
