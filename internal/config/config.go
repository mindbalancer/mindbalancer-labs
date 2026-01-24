// Package config provides configuration management for MindBalancer.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/ini.v1"
)

// Config holds all configuration for MindBalancer.
type Config struct {
	mu sync.RWMutex

	// Network
	AdminBindAddress string
	AdminPort        int
	ProxyBindAddress string
	ProxyPort        int

	// Storage
	DataDir string

	// Logging
	LogLevel  string
	LogFormat string
	LogFile   string

	// Failover
	FailoverEnabled bool
	MaxRetries      int
	RetryDelayMS    int
	RetryMaxDelayMS int
	RetryMultiplier float64

	// Circuit Breaker
	CircuitBreakerEnabled          bool
	CircuitBreakerThreshold        int
	CircuitBreakerTimeoutMS        int
	CircuitBreakerSuccessThreshold int

	// Health Check
	HealthCheckEnabled    bool
	HealthCheckIntervalMS int
	HealthCheckTimeoutMS  int

	// Connection Pool
	MaxConnectionsPerServer int
	ConnectionTimeoutMS     int
	IdleTimeoutMS           int

	// Request
	RequestTimeoutMS   int
	MaxRequestBodySize int64

	// Rate Limiting
	RateLimitEnabled         bool
	DefaultRequestsPerMinute int
	DefaultTokensPerMinute   int

	// Metrics
	PrometheusEnabled bool
	PrometheusPort    int
	PrometheusPath    string

	// Security
	AdminUsername        string
	AdminPasswordHash    string
	APIKeyEncryptionKey  string

	// TLS
	TLSEnabled  bool
	TLSCertFile string
	TLSKeyFile  string

	// Cache
	CacheEnabled            bool
	CacheMaxSize            int   // Maximum number of cached items
	CacheMaxMemoryMB        int   // Maximum memory usage in MB
	CacheTTLSeconds         int   // Default TTL in seconds
	CacheMaxItemSizeKB      int   // Maximum size of a single item in KB
	CacheCompressionEnabled bool  // Enable compression
	CacheEmbeddingsTTLHours int   // TTL for embeddings in hours

	// Cluster
	ClusterEnabled bool
	ClusterName    string
	ClusterPeers   []string

	// Model-specific timeouts (model name -> timeout in ms)
	ModelTimeouts map[string]int

	// Referee Mode
	RefereeEnabled      bool   // Enable referee mode globally
	RefereeMinResponses int    // Minimum successful responses required (default: 2)
	RefereeTimeoutMS    int    // Per-provider timeout in ms (default: 60000)
	RefereeMaxProviders int    // Maximum providers to query (default: 4)
	RefereeDefaultModel string // Default referee model if not specified in request
}

// ModelTimeout represents a model-specific timeout configuration.
type ModelTimeout struct {
	Model     string
	TimeoutMS int
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		AdminBindAddress: "0.0.0.0",
		AdminPort:        6032,
		ProxyBindAddress: "0.0.0.0",
		ProxyPort:        6033,

		DataDir: "/var/lib/mindbalancer",

		LogLevel:  "info",
		LogFormat: "json",
		LogFile:   "",

		FailoverEnabled: true,
		MaxRetries:      3,
		RetryDelayMS:    100,
		RetryMaxDelayMS: 5000,
		RetryMultiplier: 2.0,

		CircuitBreakerEnabled:          true,
		CircuitBreakerThreshold:        5,
		CircuitBreakerTimeoutMS:        30000,
		CircuitBreakerSuccessThreshold: 2,

		HealthCheckEnabled:    true,
		HealthCheckIntervalMS: 5000,
		HealthCheckTimeoutMS:  3000,

		MaxConnectionsPerServer: 100,
		ConnectionTimeoutMS:     10000,
		IdleTimeoutMS:           60000,

		RequestTimeoutMS:   120000,
		MaxRequestBodySize: 10 * 1024 * 1024,

		RateLimitEnabled:         true,
		DefaultRequestsPerMinute: 1000,
		DefaultTokensPerMinute:   100000,

		PrometheusEnabled: true,
		PrometheusPort:    9090,
		PrometheusPath:    "/metrics",

		AdminUsername:     "admin",
		AdminPasswordHash: "",

		TLSEnabled: false,

		ClusterEnabled: false,
		ClusterName:    "mindbalancer-cluster",
		ClusterPeers:   []string{},

		// Default model-specific timeouts (longer for reasoning models)
		ModelTimeouts: map[string]int{
			"o1":            300000, // 5 minutes for o1
			"o1-preview":    300000, // 5 minutes for o1-preview
			"o1-mini":       180000, // 3 minutes for o1-mini
			"gpt-4":         120000, // 2 minutes default
			"claude-3-opus": 180000, // 3 minutes for opus
		},

		// Cache defaults
		CacheEnabled:            true,
		CacheMaxSize:            10000,           // 10k items total
		CacheMaxMemoryMB:        512,             // 512MB
		CacheTTLSeconds:         300,             // 5 minutes
		CacheMaxItemSizeKB:      2048,            // 2MB per item
		CacheCompressionEnabled: true,
		CacheEmbeddingsTTLHours: 24,              // 24 hours for embeddings

		// Referee Mode defaults
		RefereeEnabled:      true,
		RefereeMinResponses: 2,       // At least 2 successful responses
		RefereeTimeoutMS:    60000,   // 60 seconds per provider
		RefereeMaxProviders: 4,       // Query up to 4 providers
		RefereeDefaultModel: "gpt-4o", // Default referee model
	}
}

// Load loads configuration from a file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		// Try default locations
		defaultPaths := []string{
			"mindbalancer.cnf",
			"/etc/mindbalancer/mindbalancer.cnf",
			filepath.Join(os.Getenv("HOME"), ".mindbalancer/mindbalancer.cnf"),
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return cfg, nil // Use defaults
	}

	iniFile, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	section := iniFile.Section("mindbalancer")

	// Network
	cfg.AdminBindAddress = section.Key("admin_bind_address").MustString(cfg.AdminBindAddress)
	cfg.AdminPort = section.Key("admin_port").MustInt(cfg.AdminPort)
	cfg.ProxyBindAddress = section.Key("proxy_bind_address").MustString(cfg.ProxyBindAddress)
	cfg.ProxyPort = section.Key("proxy_port").MustInt(cfg.ProxyPort)

	// Storage
	cfg.DataDir = section.Key("data_dir").MustString(cfg.DataDir)

	// Logging
	cfg.LogLevel = section.Key("log_level").MustString(cfg.LogLevel)
	cfg.LogFormat = section.Key("log_format").MustString(cfg.LogFormat)
	cfg.LogFile = section.Key("log_file").MustString(cfg.LogFile)

	// Failover
	cfg.FailoverEnabled = section.Key("failover_enabled").MustBool(cfg.FailoverEnabled)
	cfg.MaxRetries = section.Key("max_retries").MustInt(cfg.MaxRetries)
	cfg.RetryDelayMS = section.Key("retry_delay_ms").MustInt(cfg.RetryDelayMS)
	cfg.RetryMaxDelayMS = section.Key("retry_max_delay_ms").MustInt(cfg.RetryMaxDelayMS)
	cfg.RetryMultiplier = section.Key("retry_multiplier").MustFloat64(cfg.RetryMultiplier)

	// Circuit Breaker
	cfg.CircuitBreakerEnabled = section.Key("circuit_breaker_enabled").MustBool(cfg.CircuitBreakerEnabled)
	cfg.CircuitBreakerThreshold = section.Key("circuit_breaker_threshold").MustInt(cfg.CircuitBreakerThreshold)
	cfg.CircuitBreakerTimeoutMS = section.Key("circuit_breaker_timeout_ms").MustInt(cfg.CircuitBreakerTimeoutMS)
	cfg.CircuitBreakerSuccessThreshold = section.Key("circuit_breaker_success_threshold").MustInt(cfg.CircuitBreakerSuccessThreshold)

	// Health Check
	cfg.HealthCheckEnabled = section.Key("health_check_enabled").MustBool(cfg.HealthCheckEnabled)
	cfg.HealthCheckIntervalMS = section.Key("health_check_interval_ms").MustInt(cfg.HealthCheckIntervalMS)
	cfg.HealthCheckTimeoutMS = section.Key("health_check_timeout_ms").MustInt(cfg.HealthCheckTimeoutMS)

	// Connection Pool
	cfg.MaxConnectionsPerServer = section.Key("max_connections_per_server").MustInt(cfg.MaxConnectionsPerServer)
	cfg.ConnectionTimeoutMS = section.Key("connection_timeout_ms").MustInt(cfg.ConnectionTimeoutMS)
	cfg.IdleTimeoutMS = section.Key("idle_timeout_ms").MustInt(cfg.IdleTimeoutMS)

	// Request
	cfg.RequestTimeoutMS = section.Key("request_timeout_ms").MustInt(cfg.RequestTimeoutMS)
	cfg.MaxRequestBodySize = section.Key("max_request_body_size").MustInt64(cfg.MaxRequestBodySize)

	// Rate Limiting
	cfg.RateLimitEnabled = section.Key("rate_limit_enabled").MustBool(cfg.RateLimitEnabled)
	cfg.DefaultRequestsPerMinute = section.Key("default_requests_per_minute").MustInt(cfg.DefaultRequestsPerMinute)
	cfg.DefaultTokensPerMinute = section.Key("default_tokens_per_minute").MustInt(cfg.DefaultTokensPerMinute)

	// Metrics
	cfg.PrometheusEnabled = section.Key("prometheus_enabled").MustBool(cfg.PrometheusEnabled)
	cfg.PrometheusPort = section.Key("prometheus_port").MustInt(cfg.PrometheusPort)
	cfg.PrometheusPath = section.Key("prometheus_path").MustString(cfg.PrometheusPath)

	// Security
	cfg.AdminUsername = section.Key("admin_username").MustString(cfg.AdminUsername)
	cfg.AdminPasswordHash = section.Key("admin_password_hash").MustString(cfg.AdminPasswordHash)
	cfg.APIKeyEncryptionKey = section.Key("api_key_encryption_key").MustString(cfg.APIKeyEncryptionKey)

	// TLS
	cfg.TLSEnabled = section.Key("tls_enabled").MustBool(cfg.TLSEnabled)
	cfg.TLSCertFile = section.Key("tls_cert_file").MustString(cfg.TLSCertFile)
	cfg.TLSKeyFile = section.Key("tls_key_file").MustString(cfg.TLSKeyFile)

	// Cluster
	cfg.ClusterEnabled = section.Key("cluster_enabled").MustBool(cfg.ClusterEnabled)
	cfg.ClusterName = section.Key("cluster_name").MustString(cfg.ClusterName)
	peersStr := section.Key("cluster_peers").MustString("")
	if peersStr != "" {
		cfg.ClusterPeers = strings.Split(peersStr, ",")
	}

	// Cache
	cfg.CacheEnabled = section.Key("cache_enabled").MustBool(cfg.CacheEnabled)
	cfg.CacheMaxSize = section.Key("cache_max_size").MustInt(cfg.CacheMaxSize)
	cfg.CacheMaxMemoryMB = section.Key("cache_max_memory_mb").MustInt(cfg.CacheMaxMemoryMB)
	cfg.CacheTTLSeconds = section.Key("cache_ttl_seconds").MustInt(cfg.CacheTTLSeconds)
	cfg.CacheMaxItemSizeKB = section.Key("cache_max_item_size_kb").MustInt(cfg.CacheMaxItemSizeKB)
	cfg.CacheCompressionEnabled = section.Key("cache_compression_enabled").MustBool(cfg.CacheCompressionEnabled)
	cfg.CacheEmbeddingsTTLHours = section.Key("cache_embeddings_ttl_hours").MustInt(cfg.CacheEmbeddingsTTLHours)

	// Referee Mode
	cfg.RefereeEnabled = section.Key("referee_enabled").MustBool(cfg.RefereeEnabled)
	cfg.RefereeMinResponses = section.Key("referee_min_responses").MustInt(cfg.RefereeMinResponses)
	cfg.RefereeTimeoutMS = section.Key("referee_timeout_ms").MustInt(cfg.RefereeTimeoutMS)
	cfg.RefereeMaxProviders = section.Key("referee_max_providers").MustInt(cfg.RefereeMaxProviders)
	cfg.RefereeDefaultModel = section.Key("referee_default_model").MustString(cfg.RefereeDefaultModel)

	return cfg, nil
}

// Get returns a config value by name.
func (c *Config) Get(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch strings.ToLower(strings.ReplaceAll(name, "-", "_")) {
	case "admin_bind_address":
		return c.AdminBindAddress, true
	case "admin_port":
		return strconv.Itoa(c.AdminPort), true
	case "proxy_bind_address":
		return c.ProxyBindAddress, true
	case "proxy_port":
		return strconv.Itoa(c.ProxyPort), true
	case "data_dir":
		return c.DataDir, true
	case "log_level":
		return c.LogLevel, true
	case "log_format":
		return c.LogFormat, true
	case "log_file":
		return c.LogFile, true
	case "failover_enabled":
		return strconv.FormatBool(c.FailoverEnabled), true
	case "max_retries", "ai_max_retries":
		return strconv.Itoa(c.MaxRetries), true
	case "circuit_breaker_enabled":
		return strconv.FormatBool(c.CircuitBreakerEnabled), true
	case "circuit_breaker_threshold":
		return strconv.Itoa(c.CircuitBreakerThreshold), true
	case "health_check_enabled":
		return strconv.FormatBool(c.HealthCheckEnabled), true
	case "health_check_interval", "ai_health_check_interval":
		return strconv.Itoa(c.HealthCheckIntervalMS), true
	case "prometheus_enabled":
		return strconv.FormatBool(c.PrometheusEnabled), true
	case "prometheus_port":
		return strconv.Itoa(c.PrometheusPort), true
	default:
		return "", false
	}
}

// Set sets a config value by name.
func (c *Config) Set(name, value string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch strings.ToLower(strings.ReplaceAll(name, "-", "_")) {
	case "log_level":
		c.LogLevel = value
	case "max_retries", "ai_max_retries":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for max_retries: %w", err)
		}
		c.MaxRetries = v
	case "circuit_breaker_threshold":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for circuit_breaker_threshold: %w", err)
		}
		c.CircuitBreakerThreshold = v
	case "health_check_interval", "ai_health_check_interval":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for health_check_interval: %w", err)
		}
		c.HealthCheckIntervalMS = v
	default:
		return fmt.Errorf("unknown or read-only variable: %s", name)
	}

	return nil
}

// GetAllVariables returns all configuration variables.
func (c *Config) GetAllVariables() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]string{
		"admin_bind_address":          c.AdminBindAddress,
		"admin_port":                  strconv.Itoa(c.AdminPort),
		"proxy_bind_address":          c.ProxyBindAddress,
		"proxy_port":                  strconv.Itoa(c.ProxyPort),
		"data_dir":                    c.DataDir,
		"log_level":                   c.LogLevel,
		"log_format":                  c.LogFormat,
		"failover_enabled":            strconv.FormatBool(c.FailoverEnabled),
		"max_retries":                 strconv.Itoa(c.MaxRetries),
		"circuit_breaker_enabled":     strconv.FormatBool(c.CircuitBreakerEnabled),
		"circuit_breaker_threshold":   strconv.Itoa(c.CircuitBreakerThreshold),
		"health_check_enabled":        strconv.FormatBool(c.HealthCheckEnabled),
		"health_check_interval_ms":    strconv.Itoa(c.HealthCheckIntervalMS),
		"prometheus_enabled":          strconv.FormatBool(c.PrometheusEnabled),
		"prometheus_port":             strconv.Itoa(c.PrometheusPort),
		"cache_enabled":               strconv.FormatBool(c.CacheEnabled),
		"cache_max_size":              strconv.Itoa(c.CacheMaxSize),
		"cache_max_memory_mb":         strconv.Itoa(c.CacheMaxMemoryMB),
		"cache_ttl_seconds":           strconv.Itoa(c.CacheTTLSeconds),
		"cache_max_item_size_kb":      strconv.Itoa(c.CacheMaxItemSizeKB),
		"cache_compression_enabled":   strconv.FormatBool(c.CacheCompressionEnabled),
		"cache_embeddings_ttl_hours":  strconv.Itoa(c.CacheEmbeddingsTTLHours),
		"referee_enabled":             strconv.FormatBool(c.RefereeEnabled),
		"referee_min_responses":       strconv.Itoa(c.RefereeMinResponses),
		"referee_timeout_ms":          strconv.Itoa(c.RefereeTimeoutMS),
		"referee_max_providers":       strconv.Itoa(c.RefereeMaxProviders),
		"referee_default_model":       c.RefereeDefaultModel,
	}
}

// RequestTimeout returns the request timeout as a Duration.
func (c *Config) RequestTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}

// HealthCheckInterval returns the health check interval as a Duration.
func (c *Config) HealthCheckInterval() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.HealthCheckIntervalMS) * time.Millisecond
}

// HealthCheckTimeout returns the health check timeout as a Duration.
func (c *Config) HealthCheckTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.HealthCheckTimeoutMS) * time.Millisecond
}

// CircuitBreakerTimeout returns the circuit breaker timeout as a Duration.
func (c *Config) CircuitBreakerTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.CircuitBreakerTimeoutMS) * time.Millisecond
}

// RetryDelay returns the initial retry delay as a Duration.
func (c *Config) RetryDelay() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.RetryDelayMS) * time.Millisecond
}

// RetryMaxDelay returns the maximum retry delay as a Duration.
func (c *Config) RetryMaxDelay() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.RetryMaxDelayMS) * time.Millisecond
}

// DBPath returns the full path to the SQLite database.
func (c *Config) DBPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return filepath.Join(c.DataDir, "mindbalancer.db")
}

// RefereeTimeout returns the referee per-provider timeout as a Duration.
func (c *Config) RefereeTimeout() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return time.Duration(c.RefereeTimeoutMS) * time.Millisecond
}

// GetModelTimeout returns the timeout for a specific model.
// Falls back to default RequestTimeout if not configured.
func (c *Config) GetModelTimeout(model string) time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.ModelTimeouts != nil {
		// Try exact match
		if timeout, ok := c.ModelTimeouts[model]; ok {
			return time.Duration(timeout) * time.Millisecond
		}

		// Try prefix match (e.g., "gpt-4o-2024-08-06" matches "gpt-4o")
		for name, timeout := range c.ModelTimeouts {
			if len(model) >= len(name) && model[:len(name)] == name {
				return time.Duration(timeout) * time.Millisecond
			}
		}
	}

	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}

// SetModelTimeout sets the timeout for a specific model.
func (c *Config) SetModelTimeout(model string, timeoutMS int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ModelTimeouts == nil {
		c.ModelTimeouts = make(map[string]int)
	}
	c.ModelTimeouts[model] = timeoutMS
}

// GetAllModelTimeouts returns all configured model timeouts.
func (c *Config) GetAllModelTimeouts() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]int, len(c.ModelTimeouts))
	for k, v := range c.ModelTimeouts {
		result[k] = v
	}
	return result
}

// CacheConfig holds cache-specific configuration for the cache package.
type CacheConfig struct {
	Enabled            bool
	MaxSize            int
	MaxMemoryMB        int
	TTL                time.Duration
	MaxItemSize        int64
	CompressionEnabled bool
	EmbeddingsTTL      time.Duration
}

// GetCacheConfig returns cache configuration suitable for the cache package.
func (c *Config) GetCacheConfig() CacheConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheConfig{
		Enabled:            c.CacheEnabled,
		MaxSize:            c.CacheMaxSize,
		MaxMemoryMB:        c.CacheMaxMemoryMB,
		TTL:                time.Duration(c.CacheTTLSeconds) * time.Second,
		MaxItemSize:        int64(c.CacheMaxItemSizeKB) * 1024,
		CompressionEnabled: c.CacheCompressionEnabled,
		EmbeddingsTTL:      time.Duration(c.CacheEmbeddingsTTLHours) * time.Hour,
	}
}
