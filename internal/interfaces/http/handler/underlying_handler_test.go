package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	httpiface "github.com/QuantDevelopers/Backend/internal/interfaces/http"
	"github.com/QuantDevelopers/Backend/internal/interfaces/http/dto"
)

type mockUnderlyingUseCase struct {
	assets []entity.UnderlyingAsset
	bySym  map[string]*entity.UnderlyingAsset
	err    error
}

func (m mockUnderlyingUseCase) ListUnderlyings(_ context.Context) ([]entity.UnderlyingAsset, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.assets, nil
}

func (m mockUnderlyingUseCase) GetUnderlyingBySymbol(_ context.Context, symbolFA string) (*entity.UnderlyingAsset, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.bySym[symbolFA], nil
}

func TestListUnderlyingsEndpoint(t *testing.T) {
	tsetmc1 := "24000305886470390"
	tsetmc2 := "2400322364771558"
	uc := mockUnderlyingUseCase{
		assets: []entity.UnderlyingAsset{
			{
				ID:         1,
				SymbolFA:   "اهرم",
				NameFA:     "صندوق س. ا. کاریسما-شتاب",
				TSETMCID:   &tsetmc1,
				MarketType: "Bourse",
				IsActive:   true,
			},
			{
				ID:         2,
				SymbolFA:   "خودرو",
				NameFA:     "ایران خودرو",
				TSETMCID:   &tsetmc2,
				MarketType: "Bourse",
				IsActive:   true,
			},
		},
	}

	e := httpiface.NewRouter(config.Settings{AppName: "QuantBackend"}, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: true},
		UnderlyingUseCase: uc,
	})

	for _, path := range []string{"/api/v1/underlyings", "/api/v1/underlyings/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}

		var resp dto.UnderlyingListResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response for %s: %v", path, err)
		}
		if resp.Total != 2 || len(resp.Items) != 2 {
			t.Fatalf("unexpected total/items for %s: total=%d len=%d", path, resp.Total, len(resp.Items))
		}
		if resp.Items[0].SymbolFA != "اهرم" || resp.Items[1].SymbolFA != "خودرو" {
			t.Fatalf("unexpected items for %s: %+v", path, resp.Items)
		}
	}
}

func TestGetUnderlyingBySymbolFound(t *testing.T) {
	tsetmc := "24000305886470390"
	asset := &entity.UnderlyingAsset{
		ID:         1,
		SymbolFA:   "اهرم",
		NameFA:     "صندوق س. ا. کاریسما-شتاب",
		TSETMCID:   &tsetmc,
		MarketType: "Bourse",
		IsActive:   true,
	}

	e := httpiface.NewRouter(config.Settings{AppName: "QuantBackend"}, nil, httpiface.RouterDeps{
		HealthChecker: mockHealthChecker{healthy: true},
		UnderlyingUseCase: mockUnderlyingUseCase{
			bySym: map[string]*entity.UnderlyingAsset{"اهرم": asset},
		},
	})

	// Test both percent-encoded and raw UTF-8 Persian paths.
	paths := []string{
		"/api/v1/underlyings/" + url.PathEscape("اهرم"),
		"/api/v1/underlyings/اهرم",
	}

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200 (body: %s)", path, rec.Code, rec.Body.String())
		}

		var resp dto.UnderlyingAssetResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.SymbolFA != "اهرم" || resp.NameFA != "صندوق س. ا. کاریسما-شتاب" {
			t.Fatalf("unexpected response for %s: %+v", path, resp)
		}
	}
}

func TestGetUnderlyingBySymbolNotFoundAndErrors(t *testing.T) {
	e := httpiface.NewRouter(config.Settings{AppName: "QuantBackend"}, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: true},
		UnderlyingUseCase: mockUnderlyingUseCase{bySym: map[string]*entity.UnderlyingAsset{}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/underlyings/"+url.PathEscape("ناموجود"), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var errResp dto.HTTPErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if !strings.Contains(errResp.Detail, "ناموجود") {
		t.Fatalf("expected symbol in detail message, got %q", errResp.Detail)
	}

	eErr := httpiface.NewRouter(config.Settings{AppName: "QuantBackend"}, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: true},
		UnderlyingUseCase: mockUnderlyingUseCase{err: errors.New("db failure")},
	})
	reqErr := httptest.NewRequest(http.MethodGet, "/api/v1/underlyings", nil)
	recErr := httptest.NewRecorder()
	eErr.ServeHTTP(recErr, reqErr)
	if recErr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recErr.Code)
	}
}

func TestDocsAndOpenAPIEndpoints(t *testing.T) {
	e := httpiface.NewRouter(config.Settings{AppName: "QuantBackend", RootPath: "/quant"}, nil, httpiface.RouterDeps{
		HealthChecker:     mockHealthChecker{healthy: true},
		UnderlyingUseCase: mockUnderlyingUseCase{},
	})

	for _, path := range []string{"/quant/openapi.json", "/quant/docs", "/quant/redoc"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}
	}

	reqSpec := httptest.NewRequest(http.MethodGet, "/quant/openapi.json", nil)
	recSpec := httptest.NewRecorder()
	e.ServeHTTP(recSpec, reqSpec)
	var spec map[string]any
	if err := json.Unmarshal(recSpec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid openapi json: %v", err)
	}
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("openapi version = %v, want 3.0.3", spec["openapi"])
	}
}
