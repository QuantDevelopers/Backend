package middleware

import (
	"net/http"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

type CORSMiddleware struct {
	cors *bootstrap.CORS
}

func NewCorsMiddleware(cors *bootstrap.CORS) *CORSMiddleware {
	return &CORSMiddleware{cors: cors}
}

func (cm *CORSMiddleware) CORS() echo.MiddlewareFunc {
	allowOrigins := []string{"*"}
	if cm.cors != nil && len(cm.cors.AllowOrigins) > 0 {
		allowOrigins = cm.cors.AllowOrigins
	}

	return echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Accept-Language"},
		ExposeHeaders:    []string{echo.HeaderContentLength},
		AllowCredentials: true,
		MaxAge:           int((12 * time.Hour).Seconds()),
	})
}
