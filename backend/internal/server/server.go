package server

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"wasooli/config"
	"wasooli/internal/handlers"
	"wasooli/internal/httperr"
)

// New builds the Echo server: middleware, CORS for the SPA, a JSON error handler, routes.
func New(cfg *config.Config, log *zap.Logger, h *handlers.Handler) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = errorHandler(log)

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(requestLogger(log))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowCredentials: true, // the session is an httpOnly cookie
	}))

	e.GET("/healthz", h.Health)
	h.RegisterRoutes(e.Group("/api/v1"))

	return e
}

// errorHandler renders *httperr.Error and echo.HTTPError as JSON, logging 5xx causes.
func errorHandler(log *zap.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		var appErr *httperr.Error
		if errors.As(err, &appErr) {
			if appErr.Status >= 500 {
				log.Error("request failed", zap.Error(appErr.Err))
			}
			_ = c.JSON(appErr.Status, map[string]string{"error": appErr.Message})
			return
		}
		var he *echo.HTTPError
		if errors.As(err, &he) {
			_ = c.JSON(he.Code, map[string]any{"error": he.Message})
			return
		}
		log.Error("unhandled error", zap.Error(err))
		_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
}

func requestLogger(log *zap.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true, LogMethod: true, LogURI: true, LogLatency: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info("request",
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			)
			return nil
		},
	})
}
