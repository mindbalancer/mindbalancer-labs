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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mindbalancer/mindbalancer/internal/admin"
	"github.com/mindbalancer/mindbalancer/internal/balancer"
	"github.com/mindbalancer/mindbalancer/internal/circuit"
	"github.com/mindbalancer/mindbalancer/internal/config"
	"github.com/mindbalancer/mindbalancer/internal/health"
	"github.com/mindbalancer/mindbalancer/internal/metrics"
	"github.com/mindbalancer/mindbalancer/internal/pool"
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

	// Initialize connection pool
	poolCfg := pool.Config{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: cfg.MaxConnectionsPerServer,
		MaxConnsPerHost:     cfg.MaxConnectionsPerServer,
		IdleConnTimeout:     time.Duration(cfg.IdleTimeoutMS) * time.Millisecond,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseTimeout:     time.Duration(cfg.RequestTimeoutMS) * time.Millisecond,
		KeepAlive:           30 * time.Second,
	}
	pool.InitGlobalPool(poolCfg)
	log.Printf("Connection pool initialized: max_conns_per_host=%d", cfg.MaxConnectionsPerServer)

	// Initialize storage with encryption
	store, err := storage.NewWithEncryption(cfg.DBPath(), cfg.APIKeyEncryptionKey)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	if store.IsEncryptionEnabled() {
		log.Println("API key encryption enabled")
	} else {
		log.Println("Warning: API key encryption disabled - set api_key_encryption_key in config")
	}

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

	// Share cache with admin for management
	adminHandler.SetCache(proxyHandler.GetCache())

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

	// Track active requests for graceful shutdown
	var activeRequests int64

	// Wait for shutdown or reload signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case sig := <-sigCh:
			if sig == syscall.SIGHUP {
				// Reload configuration
				log.Println("Received SIGHUP - reloading configuration...")
				newCfg, err := config.Load(*configPath)
				if err != nil {
					log.Printf("Failed to reload config: %v", err)
					continue
				}
				// Update runtime-changeable settings
				cfg.LogLevel = newCfg.LogLevel
				cfg.MaxRetries = newCfg.MaxRetries
				cfg.CircuitBreakerThreshold = newCfg.CircuitBreakerThreshold
				cfg.DefaultRequestsPerMinute = newCfg.DefaultRequestsPerMinute
				cfg.DefaultTokensPerMinute = newCfg.DefaultTokensPerMinute
				log.Println("Configuration reloaded successfully")
				continue
			}
			log.Printf("Received shutdown signal: %v", sig)
		case err := <-errCh:
			log.Printf("Server error: %v", err)
		}
		break
	}

	// Graceful shutdown with request draining
	log.Println("Initiating graceful shutdown...")

	// Stop accepting new connections first
	healthChecker.Stop()
	
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP servers (this will stop accepting new requests)
	shutdownDone := make(chan struct{})
	go func() {
		if metricsServer != nil {
			metricsServer.Shutdown(shutdownCtx)
		}
		adminServer.Shutdown(shutdownCtx)
		proxyServer.Shutdown(shutdownCtx)
		close(shutdownDone)
	}()

	// Wait for active requests to complete
	drainTicker := time.NewTicker(100 * time.Millisecond)
	drainTimeout := time.After(25 * time.Second)
	
drainLoop:
	for {
		select {
		case <-drainTimeout:
			log.Printf("Drain timeout - forcing shutdown with %d active requests", atomic.LoadInt64(&activeRequests))
			break drainLoop
		case <-drainTicker.C:
			active := atomic.LoadInt64(&activeRequests)
			if active == 0 {
				log.Println("All requests drained")
				break drainLoop
			}
			log.Printf("Waiting for %d active requests to complete...", active)
		case <-shutdownDone:
			break drainLoop
		}
	}
	drainTicker.Stop()

	// Stop MySQL server
	mysqlServer.Stop()

	// Close connection pool
	pool.GlobalPool().CloseIdleConnections()

	log.Println("Shutdown complete")
}
