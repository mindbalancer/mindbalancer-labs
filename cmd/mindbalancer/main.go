// MindBalancer - The ProxySQL for AI
// High-performance load balancer for LLM APIs
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mindbalancer/mindbalancer/internal/admin"
	"github.com/mindbalancer/mindbalancer/internal/balancer"
	"github.com/mindbalancer/mindbalancer/internal/circuit"
	"github.com/mindbalancer/mindbalancer/internal/config"
	"github.com/mindbalancer/mindbalancer/internal/health"
	"github.com/mindbalancer/mindbalancer/internal/metrics"
	"github.com/mindbalancer/mindbalancer/internal/proxy"
	"github.com/mindbalancer/mindbalancer/internal/ratelimit"
	"github.com/mindbalancer/mindbalancer/internal/router"
	"github.com/mindbalancer/mindbalancer/internal/storage"
	"github.com/mindbalancer/mindbalancer/pkg/protocol"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("MindBalancer %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if cfg.LogLevel == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	}

	log.Printf("Starting MindBalancer version=%s admin_port=%d proxy_port=%d",
		Version, cfg.AdminPort, cfg.ProxyPort)

	// Initialize storage
	store, err := storage.New(cfg.DBPath())
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize circuit breaker manager
	circuitMgr := circuit.NewManager(
		cfg.CircuitBreakerThreshold,
		cfg.CircuitBreakerSuccessThreshold,
		cfg.CircuitBreakerTimeout(),
	)

	// Initialize health checker
	healthChecker := health.NewChecker(store, cfg.HealthCheckInterval(), cfg.HealthCheckTimeout())

	// Initialize load balancer
	bal := balancer.NewBalancer(store, healthChecker, circuitMgr)

	// Initialize router
	rtr := router.NewRouter(store)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector()

	// Initialize rate limiter
	rateLimiter := ratelimit.NewLimiter(
		store,
		cfg.RateLimitEnabled,
		cfg.DefaultRequestsPerMinute,
		cfg.DefaultTokensPerMinute,
	)

	// Initialize admin interface
	adminHandler := admin.NewAdmin(cfg, store, bal, healthChecker, circuitMgr, rtr)

	// Initialize proxy
	proxyHandler := proxy.NewProxy(cfg, store, bal, rtr, metricsCollector, rateLimiter)

	// Load initial data
	ctx := context.Background()
	if err := bal.LoadServers(ctx); err != nil {
		log.Printf("Warning: Failed to load servers: %v", err)
	}
	if err := rtr.LoadRules(ctx); err != nil {
		log.Printf("Warning: Failed to load routing rules: %v", err)
	}

	// Start health checker
	healthChecker.Start(ctx)

	// Start rate limiter cleanup
	rateLimiter.StartCleanup(ctx)

	// Create servers
	proxyServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.ProxyBindAddress, cfg.ProxyPort),
		Handler:      proxyHandler.Handler(),
		ReadTimeout:  time.Duration(cfg.RequestTimeoutMS) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.RequestTimeoutMS) * time.Millisecond,
	}

	adminServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.AdminBindAddress, cfg.AdminPort+1),
		Handler:      adminHandler.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// MySQL protocol server for mindsql
	mysqlServer := protocol.NewMySQLServer(adminHandler)

	// Metrics server
	var metricsServer *http.Server
	if cfg.PrometheusEnabled {
		metricsMux := http.NewServeMux()
		metricsMux.Handle(cfg.PrometheusPath, metricsCollector.Handler())
		metricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.PrometheusPort),
			Handler: metricsMux,
		}
	}

	// Start servers
	errCh := make(chan error, 4)

	go func() {
		log.Printf("Starting proxy server on %s", proxyServer.Addr)
		if err := proxyServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("proxy server error: %w", err)
		}
	}()

	go func() {
		log.Printf("Starting admin HTTP server on %s", adminServer.Addr)
		if err := adminServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- fmt.Errorf("admin server error: %w", err)
		}
	}()

	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.AdminBindAddress, cfg.AdminPort)
		log.Printf("Starting admin MySQL server on %s", addr)
		if err := mysqlServer.Start(addr); err != nil {
			errCh <- fmt.Errorf("mysql server error: %w", err)
		}
	}()

	if metricsServer != nil {
		go func() {
			log.Printf("Starting metrics server on %s", metricsServer.Addr)
			if err := metricsServer.ListenAndServe(); err != http.ErrServerClosed {
				errCh <- fmt.Errorf("metrics server error: %w", err)
			}
		}()
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("Received shutdown signal: %v", sig)
	case err := <-errCh:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	healthChecker.Stop()
	mysqlServer.Stop()

	if metricsServer != nil {
		metricsServer.Shutdown(shutdownCtx)
	}
	adminServer.Shutdown(shutdownCtx)
	proxyServer.Shutdown(shutdownCtx)

	log.Println("Shutdown complete")
}
