package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marketplace/internal/api/generated"
	"marketplace/internal/config"
	"marketplace/internal/httpapi"
	appmiddleware "marketplace/internal/middleware"
	"marketplace/internal/security"
	"marketplace/internal/service"
	"marketplace/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("read config", "error", err)
		os.Exit(1)
	}

	startupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	database, err := store.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	tokens := security.NewManager(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)

	queries := database.Queries()
	authService := service.NewAuthService(queries, tokens)
	productService := service.NewProductService(queries)
	orderService := service.NewOrderService(database, cfg.OrderRateLimit)
	promoService := service.NewPromoService(queries)
	api := httpapi.NewServer(database, authService, productService, orderService, promoService)

	spec, err := generated.GetSwagger()
	if err != nil {
		logger.Error("load OpenAPI specification", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	var handler http.Handler = generated.HandlerWithOptions(api, generated.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: httpapi.RouteErrorHandler,
	})

	// Порядок читается снизу вверх: запрос сначала получает request_id,
	// затем JWT, проверяется по OpenAPI и только после этого попадает в handler.
	handler = appmiddleware.OpenAPI(spec)(handler)
	handler = appmiddleware.Auth(tokens)(handler)
	handler = appmiddleware.Recovery(logger)(handler)
	handler = appmiddleware.Logging(logger)(handler)
	handler = http.MaxBytesHandler(handler, 1<<20)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("marketplace api started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-stopCtx.Done()
	logger.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
