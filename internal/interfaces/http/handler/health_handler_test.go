package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	httpiface "github.com/QuantDevelopers/Backend/internal/interfaces/http"
	"github.com/QuantDevelopers/Backend/internal/interfaces/http/dto"
)

type mockHealthChecker struct {
	healthy bool
}

func (m mockHealthChecker) CheckDB(_ context.Context) bool {
	return m.healthy
}

type noopUnderlyingUseCase struct{}

func (noopUnderlyingUseCase) ListUnderlyings(_ context.Context) ([]entity.UnderlyingAsset, error) {
	return nil, nil
}

func (noopUnderlyingUseCase) GetUnderlyingBySymbol(_ context.Context, _ string) (*entity.UnderlyingAsset, error) {
	return nil, nil
}

func TestHealthEndpointHealthy(t *testing.T) {
	cfg := config.Settings{
		AppName:     "QuantBackend",
		CORSOrigins: []string{"http://localhost:3000"},
	}
	e := httpiface.NewRouter(cfg, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: true},
		UnderlyingUseCase: noopUnderlyingUseCase{},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp dto.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "healthy" || resp.App != "QuantBackend" || resp.Database != "connected" {
		t.Fatalf("unexpected health response: %+v", resp)
	}
	if resp.Timestamp.IsZero() {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestHealthEndpointDegraded(t *testing.T) {
	cfg := config.Settings{
		AppName:     "QuantBackend",
		CORSOrigins: []string{"http://localhost:3000"},
	}
	e := httpiface.NewRouter(cfg, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: false},
		UnderlyingUseCase: noopUnderlyingUseCase{},
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}

	var resp dto.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Status != "degraded" || resp.Database != "disconnected" {
		t.Fatalf("unexpected degraded health response: %+v", resp)
	}
}
