package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"aba-pocket/internal/config"
	"aba-pocket/internal/db"
	"aba-pocket/internal/handlers"
	"aba-pocket/internal/repository"
)

func main() {
	cfg := config.Load()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	if err := db.Migrate(pool); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Database migrations applied")

	repos := repository.New(pool)

	// Seed initial admin if configured and no users exist
	seedAdmin(cfg, repos)

	h := handlers.New(cfg, repos)
	if len(cfg.TrustedProxies) > 0 {
		log.Printf("Trusted proxies: %v", cfg.TrustedProxies)
	}
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      h.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	go cleanupSessions(repos)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

func seedAdmin(cfg *config.Config, repos *repository.Repositories) {
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return
	}

	ctx := context.Background()
	count, err := repos.Users.Count(ctx)
	if err != nil || count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash admin password: %v", err)
		return
	}

	_, err = repos.Users.Create(ctx, cfg.AdminUsername, "", string(hash))
	if err != nil {
		log.Printf("Failed to create initial admin: %v", err)
		return
	}
	log.Printf("Initial admin user '%s' created", cfg.AdminUsername)
}

func cleanupSessions(repos *repository.Repositories) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		n, err := repos.Users.DeleteExpiredSessions(context.Background())
		if err != nil {
			log.Printf("Session cleanup error: %v", err)
		} else if n > 0 {
			log.Printf("Session cleanup: %d expired sessions removed", n)
		}
	}
}
