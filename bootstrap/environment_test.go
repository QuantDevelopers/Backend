package bootstrap

import "testing"

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

func TestRunLoadsEnvAndConstants(t *testing.T) {
	t.Setenv("APP_NAME", "QuantTest")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("ROOT_PATH", "api/quant/")
	t.Setenv("DATABASE_URL", "postgresql+asyncpg://u:p@localhost:5432/testdb")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000, https://example.com ")

	cfg := Run()

	if cfg.Constants == nil || cfg.Env == nil {
		t.Fatal("expected non-nil Constants and Env")
	}
	if cfg.Env.Server.AppName != "QuantTest" {
		t.Fatalf("AppName = %q, want QuantTest", cfg.Env.Server.AppName)
	}
	if cfg.Env.Server.Address() != "0.0.0.0:9090" {
		t.Fatalf("Address = %q, want 0.0.0.0:9090", cfg.Env.Server.Address())
	}
	if cfg.Env.Server.RootPath != "/api/quant" {
		t.Fatalf("RootPath = %q, want /api/quant", cfg.Env.Server.RootPath)
	}
	if cfg.Env.PrimaryDB.DSN() != "postgresql://u:p@localhost:5432/testdb" {
		t.Fatalf("DSN = %q", cfg.Env.PrimaryDB.DSN())
	}
	if len(cfg.Env.CORS.AllowOrigins) != 2 {
		t.Fatalf("unexpected CORS origins: %#v", cfg.Env.CORS.AllowOrigins)
	}
}
