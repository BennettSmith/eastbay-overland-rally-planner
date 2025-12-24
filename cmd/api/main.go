package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eastbay-overland-rally-planner/internal/adapters/httpapi"
	"eastbay-overland-rally-planner/internal/platform/auth/jwtverifier"
	"eastbay-overland-rally-planner/internal/platform/config"
)

func main() {
	port := getenv("PORT", "8080")

	jwtCfg, err := config.LoadJWTConfigFromEnv()
	if err != nil {
		log.Fatalf("invalid auth config: %v", err)
	}
	verifier := jwtverifier.New(jwtCfg)
	authMW := httpapi.NewAuthMiddleware(verifier)

	// Temporary strict-server stub (we'll swap this out once app services exist),
	// but the HTTP layer is already "real" (auth + error shaping).
	handler := httpapi.NewRouterWithOptions(
		httpapi.StrictUnimplemented{},
		httpapi.RouterOptions{AuthMiddleware: authMW},
	)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
