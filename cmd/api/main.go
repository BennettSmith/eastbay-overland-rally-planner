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
	memidempotency "eastbay-overland-rally-planner/internal/adapters/memory/idempotency"
	memmemberrepo "eastbay-overland-rally-planner/internal/adapters/memory/memberrepo"
	memrsvprepo "eastbay-overland-rally-planner/internal/adapters/memory/rsvprepo"
	memtriprepo "eastbay-overland-rally-planner/internal/adapters/memory/triprepo"
	"eastbay-overland-rally-planner/internal/app/members"
	"eastbay-overland-rally-planner/internal/app/trips"
	"eastbay-overland-rally-planner/internal/platform/auth/jwtverifier"
	platformclock "eastbay-overland-rally-planner/internal/platform/clock"
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

	// In-memory dependencies (Milestone 3). We'll swap these to Postgres adapters later.
	clk := platformclock.NewSystemClock()
	memberRepo := memmemberrepo.NewRepo()
	tripRepo := memtriprepo.NewRepo()
	rsvpRepo := memrsvprepo.NewRepo()
	idemStore := memidempotency.NewStore()
	memberSvc := members.NewService(memberRepo, clk)
	tripSvc := trips.NewService(tripRepo, memberRepo, rsvpRepo)

	// Real server implementation for Members; other endpoints remain strict-unimplemented.
	api := httpapi.NewServer(memberSvc, tripSvc, idemStore)

	handler := httpapi.NewRouterWithOptions(
		api,
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
