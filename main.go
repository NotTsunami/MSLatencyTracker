// MSLatencyTracker pings MapleStory channel servers on an interval and serves
// the latest, average, and historical latency over a REST API.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"mslatencytracker/db"
	"mslatencytracker/handlers"
	"mslatencytracker/pinger"
	"mslatencytracker/store"
)

func main() {
	// A missing .env file is fine; in production the variables are set by
	// the environment.
	_ = godotenv.Load()

	port := getenv("PORT", "8080")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	pingInterval := time.Duration(getenvInt("PING_INTERVAL_MS", 300000)) * time.Millisecond
	pingTimeout := time.Duration(getenvInt("PING_TIMEOUT_S", 5)) * time.Second
	retention := time.Duration(getenvInt("HISTORY_RETENTION_HOURS", 24)) * time.Hour
	cleanupInterval := time.Duration(getenvInt("CLEANUP_INTERVAL_MIN", 60)) * time.Minute

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	s := store.New(database)

	router := gin.Default()

	// In production the only path to this service is the cloudflared tunnel,
	// so don't trust X-Forwarded-For from arbitrary peers; take the client IP
	// from Cloudflare's CF-Connecting-IP header instead (absent in local dev,
	// where Gin falls back to the socket address).
	_ = router.SetTrustedProxies(nil)
	router.TrustedPlatform = gin.PlatformCloudflare

	router.Use(corsMiddleware())

	// Health check: confirms the database is reachable. Tied to the request
	// context so a wedged database fails the check instead of hanging it.
	router.GET("/health", func(c *gin.Context) {
		if err := database.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	handlers.New(s).Register(api)

	// ctx is cancelled on SIGINT/SIGTERM; the background workers stop with it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pinger.Start(ctx, s, pingInterval, pingTimeout)
	startCleanup(ctx, database, retention, cleanupInterval)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
		// The API is public; cap how long a connection can stall so slow
		// clients can't pin connections open indefinitely.
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// Surfacing the listen error through a channel (rather than log.Fatal in
	// the goroutine) lets the deferred database.Close still run.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("MSLatencyTracker listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Shutting down...")
	case err := <-serverErr:
		log.Printf("Server failed: %v", err)
	}

	// Give in-flight requests a grace period to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// startCleanup runs the expired-row deletion job on a timer until ctx is
// cancelled.
func startCleanup(ctx context.Context, database *sql.DB, retention, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := db.CleanupOldRows(database, retention)
				if err != nil {
					log.Printf("Cleanup failed: %v", err)
					continue
				}
				if n > 0 {
					log.Printf("Cleanup: deleted %d expired rows.", n)
				}
			}
		}
	}()
}

// corsMiddleware allows requests from any origin and short-circuits the
// browser's OPTIONS preflight with a 204.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// getenv returns the value of an env var, or a fallback if it is unset/empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getenvInt is like getenv but parses the value as a positive integer,
// falling back on a missing, invalid, or non-positive value. Every caller
// builds a ticker or timeout duration, where zero or negative would panic.
func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
		log.Printf("Ignoring %s=%q: not a positive integer; using default %d", key, v, fallback)
	}
	return fallback
}
