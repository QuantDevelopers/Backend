package docs

import (
	"fmt"
	"net/http"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/labstack/echo/v4"
)

type GeneralDocsController struct {
	server *bootstrap.Server
}

func NewGeneralDocsController(server *bootstrap.Server) *GeneralDocsController {
	return &GeneralDocsController{server: server}
}

func (dc *GeneralDocsController) OpenAPISpec(ctx echo.Context) error {
	spec := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Quant Option Data Platform API",
			"version": "1.0.0",
			"description": "Tehran Stock Exchange (TSETMC) Options Market Tick & Intraday Data Platform. " +
				"Provides endpoints for underlying assets, option contracts, raw and clean tick streams, " +
				"as-of joined series, and data quality indicators.",
		},
		"servers": []map[string]any{
			{"url": dc.serverURL()},
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Health"},
					"summary":     "Service Health Check",
					"description": "Returns the operational status of the backend API and PostgreSQL database connection.",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Service and database are healthy",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/HealthEnvelope"},
								},
							},
						},
						"503": map[string]any{
							"description": "Database is unreachable",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/HealthEnvelope"},
								},
							},
						},
					},
				},
			},
			"/api/v1/underlyings": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Underlying Assets"},
					"summary":     "List All Underlying Assets",
					"description": "Returns the 25 key underlying assets for the Tehran Stock Exchange options market.",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "List of underlying assets",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/UnderlyingListEnvelope"},
								},
							},
						},
					},
				},
			},
			"/api/v1/underlyings/{symbol_fa}": map[string]any{
				"get": map[string]any{
					"tags":        []string{"Underlying Assets"},
					"summary":     "Get Underlying Asset by Symbol",
					"description": "Returns details of a specific underlying asset by its Persian symbol (e.g., خودرو, اهرم).",
					"parameters": []map[string]any{
						{
							"name":        "symbol_fa",
							"in":          "path",
							"required":    true,
							"description": "Persian symbol of the underlying asset",
							"schema":      map[string]any{"type": "string"},
						},
					},
					"responses": map[string]any{
						"200": map[string]any{
							"description": "Underlying asset details",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/UnderlyingEnvelope"},
								},
							},
						},
						"404": map[string]any{
							"description": "Underlying asset not found",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{"$ref": "#/components/schemas/ErrorEnvelope"},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"HealthResponse": map[string]any{
					"type":     "object",
					"required": []string{"status", "app", "database", "timestamp"},
					"properties": map[string]any{
						"status":    map[string]any{"type": "string", "example": "healthy"},
						"app":       map[string]any{"type": "string", "example": "QuantBackend"},
						"database":  map[string]any{"type": "string", "example": "connected"},
						"timestamp": map[string]any{"type": "string", "format": "date-time"},
					},
				},
				"HealthEnvelope": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"statusCode": map[string]any{"type": "integer", "example": 200},
						"message":    map[string]any{"type": "string", "example": "OK"},
						"data":       map[string]any{"$ref": "#/components/schemas/HealthResponse"},
					},
				},
				"UnderlyingResponse": map[string]any{
					"type":     "object",
					"required": []string{"id", "symbol_fa", "name_fa", "market_type", "is_active"},
					"properties": map[string]any{
						"id":          map[string]any{"type": "integer", "example": 1},
						"symbol_fa":   map[string]any{"type": "string", "example": "اهرم"},
						"name_fa":     map[string]any{"type": "string", "example": "صندوق س. ا. کاریسما-شتاب"},
						"tsetmc_id":   map[string]any{"type": "string", "nullable": true},
						"market_type": map[string]any{"type": "string", "example": "Bourse"},
						"is_active":   map[string]any{"type": "boolean", "example": true},
						"created_at":  map[string]any{"type": "string", "format": "date-time"},
					},
				},
				"UnderlyingListResponse": map[string]any{
					"type":     "object",
					"required": []string{"total", "items"},
					"properties": map[string]any{
						"total": map[string]any{"type": "integer", "example": 25},
						"items": map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/UnderlyingResponse"},
						},
					},
				},
				"UnderlyingEnvelope": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"statusCode": map[string]any{"type": "integer", "example": 200},
						"message":    map[string]any{"type": "string", "example": "OK"},
						"data":       map[string]any{"$ref": "#/components/schemas/UnderlyingResponse"},
					},
				},
				"UnderlyingListEnvelope": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"statusCode": map[string]any{"type": "integer", "example": 200},
						"message":    map[string]any{"type": "string", "example": "OK"},
						"data":       map[string]any{"$ref": "#/components/schemas/UnderlyingListResponse"},
					},
				},
				"ErrorEnvelope": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"statusCode": map[string]any{"type": "integer", "example": 404},
						"message":    map[string]any{"type": "string"},
						"data":       map[string]any{"nullable": true},
					},
				},
			},
		},
	}

	return ctx.JSON(http.StatusOK, spec)
}

func (dc *GeneralDocsController) SwaggerUI(ctx echo.Context) error {
	specURL := dc.specURL()
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Quant Option Data Platform API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      SwaggerUIBundle({
        url: %q,
        dom_id: '#swagger-ui',
        presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`, specURL)
	return ctx.HTML(http.StatusOK, html)
}

func (dc *GeneralDocsController) Redoc(ctx echo.Context) error {
	specURL := dc.specURL()
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Quant Option Data Platform API - ReDoc</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
</head>
<body>
  <redoc spec-url=%q></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`, specURL)
	return ctx.HTML(http.StatusOK, html)
}

func (dc *GeneralDocsController) serverURL() string {
	if dc.server == nil || dc.server.RootPath == "" {
		return "/"
	}
	return dc.server.RootPath
}

func (dc *GeneralDocsController) specURL() string {
	if dc.server == nil || dc.server.RootPath == "" {
		return "/openapi.json"
	}
	return dc.server.RootPath + "/openapi.json"
}
