// Package admin provides the administrative interface for MindBalancer.
package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mindbalancer/mindbalancer/internal/balancer"
	"github.com/mindbalancer/mindbalancer/internal/cache"
	"github.com/mindbalancer/mindbalancer/internal/circuit"
	"github.com/mindbalancer/mindbalancer/internal/config"
	"github.com/mindbalancer/mindbalancer/internal/health"
	"github.com/mindbalancer/mindbalancer/internal/router"
	"github.com/mindbalancer/mindbalancer/internal/storage"
)

// Admin provides administrative functions.
type Admin struct {
	config   *config.Config
	storage  *storage.Storage
	balancer *balancer.Balancer
	health   *health.Checker
	circuits *circuit.Manager
	router   *router.Router
	cache    *cache.Cache
}

// NewAdmin creates a new admin interface.
func NewAdmin(cfg *config.Config, store *storage.Storage, bal *balancer.Balancer, hc *health.Checker, cm *circuit.Manager, rtr *router.Router) *Admin {
	return &Admin{
		config:   cfg,
		storage:  store,
		balancer: bal,
		health:   hc,
		circuits: cm,
		router:   rtr,
	}
}

// SetCache sets the cache instance for admin management.
func (a *Admin) SetCache(c *cache.Cache) {
	a.cache = c
}

// Handler returns the HTTP handler for admin interface.
func (a *Admin) Handler() http.Handler {
	mux := http.NewServeMux()

	// Server management
	mux.HandleFunc("/api/servers", a.handleServers)
	mux.HandleFunc("/api/servers/", a.handleServerByName)

	// User management
	mux.HandleFunc("/api/users", a.handleUsers)
	mux.HandleFunc("/api/users/", a.handleUserByName)

	// Routing rules
	mux.HandleFunc("/api/rules", a.handleRules)
	mux.HandleFunc("/api/routing-rules", a.handleRules)

	// Hostgroups
	mux.HandleFunc("/api/hostgroups", a.handleHostgroups)

	// Variables
	mux.HandleFunc("/api/variables", a.handleVariables)

	// Stats
	mux.HandleFunc("/api/stats", a.handleStats)
	mux.HandleFunc("/api/stats/servers", a.handleServerStats)
	mux.HandleFunc("/api/stats/requests", a.handleRequestStats)

	// Health
	mux.HandleFunc("/api/health", a.handleHealthStatus)

	// Operations
	mux.HandleFunc("/api/reload", a.handleReload)

	// Cache management
	mux.HandleFunc("/api/cache", a.handleCache)
	mux.HandleFunc("/api/cache/clear", a.handleCacheClear)

	// Web UI Dashboard
	mux.HandleFunc("/", a.handleDashboard)

	return mux
}

// handleDashboard serves the web UI dashboard.
func (a *Admin) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for API calls from dashboard
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// If it's an API path, return 404 (already handled by other handlers)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		a.writeError(w, http.StatusNotFound, "Endpoint not found")
		return
	}

	// Serve embedded dashboard HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

// Server handlers

func (a *Admin) handleServers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		servers, err := a.storage.GetServers(ctx, nil)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Enrich with runtime status
		type ServerInfo struct {
			storage.Server
			Healthy     bool   `json:"healthy"`
			Connections int64  `json:"connections"`
			AvgLatency  string `json:"avg_latency"`
		}

		info := make([]ServerInfo, len(servers))
		for i, srv := range servers {
			info[i].Server = srv
			info[i].Healthy = a.health.IsHealthy(srv.Name)

			if state, err := a.balancer.GetServerState(srv.Name); err == nil {
				info[i].Connections = state.Connections
				info[i].AvgLatency = state.AvgLatency.String()
			}
		}

		a.writeJSON(w, info)

	case http.MethodPost:
		var srv storage.Server
		if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if srv.Name == "" {
			a.writeError(w, http.StatusBadRequest, "Server name is required")
			return
		}
		if srv.Endpoint == "" {
			a.writeError(w, http.StatusBadRequest, "Endpoint is required")
			return
		}
		if srv.ProviderType == "" {
			srv.ProviderType = "openai"
		}
		if srv.Weight == 0 {
			srv.Weight = 1
		}
		if srv.MaxConnections == 0 {
			srv.MaxConnections = 100
		}
		if srv.Status == "" {
			srv.Status = storage.ServerStatusOnline
		}

		if err := a.storage.InsertServer(ctx, &srv); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		a.writeJSON(w, srv)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *Admin) handleServerByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := strings.TrimPrefix(r.URL.Path, "/api/servers/")

	switch r.Method {
	case http.MethodGet:
		srv, err := a.storage.GetServerByName(ctx, name)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if srv == nil {
			a.writeError(w, http.StatusNotFound, "Server not found")
			return
		}
		a.writeJSON(w, srv)

	case http.MethodPut:
		var srv storage.Server
		if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		srv.Name = name

		if err := a.storage.UpdateServer(ctx, &srv); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.writeJSON(w, srv)

	case http.MethodDelete:
		if err := a.storage.DeleteServer(ctx, name); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// User handlers

func (a *Admin) handleUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		users, err := a.storage.GetUsers(ctx)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.writeJSON(w, users)

	case http.MethodPost:
		var user storage.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if user.Username == "" {
			a.writeError(w, http.StatusBadRequest, "Username is required")
			return
		}
		if user.MaxRequestsPerMinute == 0 {
			user.MaxRequestsPerMinute = 1000
		}
		if user.MaxTokensPerMinute == 0 {
			user.MaxTokensPerMinute = 100000
		}
		user.Active = true

		if err := a.storage.InsertUser(ctx, &user); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		a.writeJSON(w, user)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *Admin) handleUserByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := strings.TrimPrefix(r.URL.Path, "/api/users/")

	switch r.Method {
	case http.MethodGet:
		user, err := a.storage.GetUserByUsername(ctx, username)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if user == nil {
			a.writeError(w, http.StatusNotFound, "User not found")
			return
		}
		a.writeJSON(w, user)

	case http.MethodPut:
		var user storage.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		user.Username = username

		if err := a.storage.UpdateUser(ctx, &user); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.writeJSON(w, user)

	case http.MethodDelete:
		if err := a.storage.DeleteUser(ctx, username); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Rules handlers

func (a *Admin) handleRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		rules, err := a.storage.GetRoutingRules(ctx)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		a.writeJSON(w, rules)

	case http.MethodPost:
		var rule storage.RoutingRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		rule.Active = true

		if err := a.storage.InsertRoutingRule(ctx, &rule); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		a.writeJSON(w, rule)

	case http.MethodDelete:
		ruleIDStr := r.URL.Query().Get("rule_id")
		if ruleIDStr == "" {
			a.writeError(w, http.StatusBadRequest, "rule_id is required")
			return
		}
		ruleID, err := strconv.Atoi(ruleIDStr)
		if err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid rule_id")
			return
		}

		if err := a.storage.DeleteRoutingRule(ctx, ruleID); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Hostgroups handler

func (a *Admin) handleHostgroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		hostgroups, err := a.storage.GetHostgroups(ctx)
		if err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Enrich with server count
		type HostgroupInfo struct {
			storage.Hostgroup
			ServerCount int `json:"server_count"`
		}

		info := make([]HostgroupInfo, len(hostgroups))
		for i, hg := range hostgroups {
			info[i].Hostgroup = hg
			info[i].ServerCount = len(a.balancer.GetHostgroupServers(hg.GroupID))
		}

		a.writeJSON(w, info)

	case http.MethodPost:
		var hg storage.Hostgroup
		if err := json.NewDecoder(r.Body).Decode(&hg); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := a.storage.InsertHostgroup(ctx, &hg); err != nil {
			a.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.WriteHeader(http.StatusCreated)
		a.writeJSON(w, hg)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Variables handler

func (a *Admin) handleVariables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		vars := a.config.GetAllVariables()
		a.writeJSON(w, vars)

	case http.MethodPut:
		var req struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := a.config.Set(req.Name, req.Value); err != nil {
			a.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		a.writeJSON(w, map[string]string{"status": "ok"})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Stats handlers

func (a *Admin) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := a.balancer.Stats()
	a.writeJSON(w, stats)
}

func (a *Admin) handleServerStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := a.storage.GetServerStats(ctx)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.writeJSON(w, stats)
}

func (a *Admin) handleRequestStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := a.storage.GetRecentLogs(ctx, limit)
	if err != nil {
		a.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.writeJSON(w, logs)
}

// Health handler

func (a *Admin) handleHealthStatus(w http.ResponseWriter, r *http.Request) {
	status := a.health.GetAllStatus()
	a.writeJSON(w, status)
}

// Reload handler

func (a *Admin) handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Reload servers
	if err := a.balancer.LoadServers(ctx); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to reload servers: "+err.Error())
		return
	}

	// Reload routing rules
	if err := a.router.LoadRules(ctx); err != nil {
		a.writeError(w, http.StatusInternalServerError, "Failed to reload rules: "+err.Error())
		return
	}

	a.writeJSON(w, map[string]string{"status": "reloaded"})
}

// Cache handlers

func (a *Admin) handleCache(w http.ResponseWriter, r *http.Request) {
	if a.cache == nil {
		a.writeError(w, http.StatusServiceUnavailable, "Cache not initialized")
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Get cache status and stats
		stats := a.cache.Stats()
		response := map[string]any{
			"enabled":    a.cache.IsEnabled(),
			"hits":       stats.Hits,
			"misses":     stats.Misses,
			"hit_rate":   a.cache.HitRate(),
			"evictions":  stats.Evictions,
			"size_bytes": stats.Size,
			"item_count": stats.ItemCount,
		}
		a.writeJSON(w, response)

	case http.MethodPut, http.MethodPost:
		// Enable/disable cache
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			a.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		a.cache.SetEnabled(req.Enabled)
		status := "disabled"
		if req.Enabled {
			status = "enabled"
		}
		a.writeJSON(w, map[string]string{"status": "ok", "cache": status})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *Admin) handleCacheClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if a.cache == nil {
		a.writeError(w, http.StatusServiceUnavailable, "Cache not initialized")
		return
	}

	a.cache.Clear()
	a.writeJSON(w, map[string]string{"status": "ok", "message": "Cache cleared"})
}

// Execute executes an admin command (for mindsql compatibility).
func (a *Admin) Execute(ctx context.Context, command string) (string, error) {
	command = strings.TrimSpace(command)
	upper := strings.ToUpper(command)

	switch {
	case strings.HasPrefix(upper, "SELECT * FROM AI_SERVERS"):
		return a.executeSelectServers(ctx)
	case strings.HasPrefix(upper, "SELECT * FROM AI_USERS"):
		return a.executeSelectUsers(ctx)
	case strings.HasPrefix(upper, "SELECT * FROM AI_ROUTING_RULES"):
		return a.executeSelectRules(ctx)
	case strings.HasPrefix(upper, "SELECT * FROM GLOBAL_VARIABLES"):
		return a.executeSelectVariables()
	case strings.HasPrefix(upper, "SELECT * FROM STATS_AI_SERVERS"):
		return a.executeSelectServerStats(ctx)
	case strings.HasPrefix(upper, "SELECT * FROM STATS_AI_REQUESTS"):
		return a.executeSelectRequestStats(ctx)
	case strings.HasPrefix(upper, "SHOW PROCESSLIST"):
		return a.executeShowProcesslist()
	case strings.HasPrefix(upper, "SHOW STATS"):
		return a.executeShowStats()
	case strings.HasPrefix(upper, "SHOW HOSTGROUPS"):
		return a.executeShowHostgroups(ctx)
	case strings.HasPrefix(upper, "SHOW API KEYS"):
		return a.executeShowAPIKeys(ctx)
	case strings.HasPrefix(upper, "SHOW HEALTH STATUS"):
		return a.executeShowHealthStatus()
	case strings.HasPrefix(upper, "SHOW CACHE STATUS"):
		return a.executeShowCacheStatus()
	case strings.HasPrefix(upper, "CACHE ENABLE"):
		return a.executeCacheEnable(true)
	case strings.HasPrefix(upper, "CACHE DISABLE"):
		return a.executeCacheEnable(false)
	case strings.HasPrefix(upper, "CACHE CLEAR"):
		return a.executeCacheClear()
	case strings.HasPrefix(upper, "LOAD AI SERVERS TO RUNTIME"):
		return a.executeLoadServers(ctx)
	case strings.HasPrefix(upper, "LOAD AI ROUTING RULES TO RUNTIME"):
		return a.executeLoadRules(ctx)
	case strings.HasPrefix(upper, "INSERT INTO AI_SERVERS"):
		return a.executeInsertServer(ctx, command)
	case strings.HasPrefix(upper, "DELETE FROM AI_SERVERS"):
		return a.executeDeleteServer(ctx, command)
	case strings.HasPrefix(upper, "SET "):
		return a.executeSet(command)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

func (a *Admin) executeSelectServers(ctx context.Context) (string, error) {
	servers, err := a.storage.GetServers(ctx, nil)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+-----------------+-------------+----------------------------------+----------+--------+--------+\n")
	sb.WriteString("| name            | provider    | endpoint                         | hostgroup| weight | status |\n")
	sb.WriteString("+-----------------+-------------+----------------------------------+----------+--------+--------+\n")

	for _, srv := range servers {
		sb.WriteString(fmt.Sprintf("| %-15s | %-11s | %-32s | %8d | %6d | %-6s |\n",
			truncate(srv.Name, 15),
			truncate(srv.ProviderType, 11),
			truncate(srv.Endpoint, 32),
			srv.Hostgroup,
			srv.Weight,
			srv.Status))
	}
	sb.WriteString("+-----------------+-------------+----------------------------------+----------+--------+--------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(servers)))

	return sb.String(), nil
}

func (a *Admin) executeSelectUsers(ctx context.Context) (string, error) {
	users, err := a.storage.GetUsers(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+------------------+--------+------------+---------------+\n")
	sb.WriteString("| username         | active | req/min    | tokens/min    |\n")
	sb.WriteString("+------------------+--------+------------+---------------+\n")

	for _, u := range users {
		active := "Yes"
		if !u.Active {
			active = "No"
		}
		sb.WriteString(fmt.Sprintf("| %-16s | %-6s | %10d | %13d |\n",
			truncate(u.Username, 16),
			active,
			u.MaxRequestsPerMinute,
			u.MaxTokensPerMinute))
	}
	sb.WriteString("+------------------+--------+------------+---------------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(users)))

	return sb.String(), nil
}

func (a *Admin) executeSelectRules(ctx context.Context) (string, error) {
	rules, err := a.storage.GetRoutingRules(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+---------+--------+---------------+---------------+----------+---------+\n")
	sb.WriteString("| rule_id | active | match_model   | match_pattern | dest_hg  | priority|\n")
	sb.WriteString("+---------+--------+---------------+---------------+----------+---------+\n")

	for _, r := range rules {
		active := "Yes"
		if !r.Active {
			active = "No"
		}
		sb.WriteString(fmt.Sprintf("| %7d | %-6s | %-13s | %-13s | %8d | %7d |\n",
			r.RuleID,
			active,
			truncate(r.MatchModel, 13),
			truncate(r.MatchPattern, 13),
			r.DestinationHostgroup,
			r.Priority))
	}
	sb.WriteString("+---------+--------+---------------+---------------+----------+---------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(rules)))

	return sb.String(), nil
}

func (a *Admin) executeSelectVariables() (string, error) {
	vars := a.config.GetAllVariables()

	var sb strings.Builder
	sb.WriteString("+---------------------------+------------------+\n")
	sb.WriteString("| variable_name             | variable_value   |\n")
	sb.WriteString("+---------------------------+------------------+\n")

	for name, value := range vars {
		sb.WriteString(fmt.Sprintf("| %-25s | %-16s |\n",
			truncate(name, 25),
			truncate(value, 16)))
	}
	sb.WriteString("+---------------------------+------------------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(vars)))

	return sb.String(), nil
}

func (a *Admin) executeSelectServerStats(ctx context.Context) (string, error) {
	stats, err := a.storage.GetServerStats(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+-----------------+-----------+---------+--------+-------------+\n")
	sb.WriteString("| server          | requests  | success | errors | avg_latency |\n")
	sb.WriteString("+-----------------+-----------+---------+--------+-------------+\n")

	for _, s := range stats {
		sb.WriteString(fmt.Sprintf("| %-15s | %9d | %7d | %6d | %9.2fms |\n",
			truncate(s["server_name"].(string), 15),
			s["total_requests"],
			s["success_count"],
			s["error_count"],
			s["avg_latency_ms"]))
	}
	sb.WriteString("+-----------------+-----------+---------+--------+-------------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(stats)))

	return sb.String(), nil
}

func (a *Admin) executeSelectRequestStats(ctx context.Context) (string, error) {
	logs, err := a.storage.GetRecentLogs(ctx, 20)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+------------------+-----------------+-------------+--------+----------+\n")
	sb.WriteString("| timestamp        | server          | model       | status | latency  |\n")
	sb.WriteString("+------------------+-----------------+-------------+--------+----------+\n")

	for _, l := range logs {
		sb.WriteString(fmt.Sprintf("| %-16s | %-15s | %-11s | %6d | %6dms |\n",
			l.Timestamp.Format("15:04:05"),
			truncate(l.ServerName, 15),
			truncate(l.Model, 11),
			l.StatusCode,
			l.LatencyMS))
	}
	sb.WriteString("+------------------+-----------------+-------------+--------+----------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(logs)))

	return sb.String(), nil
}

func (a *Admin) executeShowProcesslist() (string, error) {
	states := a.balancer.GetAllServerStates()

	var sb strings.Builder
	sb.WriteString("+-----------------+-------------+------------+\n")
	sb.WriteString("| server          | connections | total_reqs |\n")
	sb.WriteString("+-----------------+-------------+------------+\n")

	for _, s := range states {
		sb.WriteString(fmt.Sprintf("| %-15s | %11d | %10d |\n",
			truncate(s.Server.Name, 15),
			s.Connections,
			s.TotalReqs))
	}
	sb.WriteString("+-----------------+-------------+------------+\n")

	return sb.String(), nil
}

func (a *Admin) executeShowStats() (string, error) {
	stats := a.balancer.Stats()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Algorithm: %s\n", stats.Algorithm))
	sb.WriteString(fmt.Sprintf("Total Servers: %d\n", stats.TotalServers))
	sb.WriteString(fmt.Sprintf("Healthy Servers: %d\n", stats.HealthyServers))
	sb.WriteString(fmt.Sprintf("Hostgroups: %d\n", len(stats.HostgroupSizes)))

	return sb.String(), nil
}

func (a *Admin) executeShowHostgroups(ctx context.Context) (string, error) {
	hostgroups, err := a.storage.GetHostgroups(ctx)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+----------+---------------+-------------+\n")
	sb.WriteString("| group_id | name          | servers     |\n")
	sb.WriteString("+----------+---------------+-------------+\n")

	for _, hg := range hostgroups {
		count := len(a.balancer.GetHostgroupServers(hg.GroupID))
		sb.WriteString(fmt.Sprintf("| %8d | %-13s | %11d |\n",
			hg.GroupID,
			truncate(hg.Name, 13),
			count))
	}
	sb.WriteString("+----------+---------------+-------------+\n")

	return sb.String(), nil
}

func (a *Admin) executeShowAPIKeys(ctx context.Context) (string, error) {
	servers, err := a.storage.GetServers(ctx, nil)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("+-----------------+-------------+------------------------------------------+\n")
	sb.WriteString("| name            | provider    | api_key (masked)                         |\n")
	sb.WriteString("+-----------------+-------------+------------------------------------------+\n")

	for _, srv := range servers {
		maskedKey := maskAPIKey(srv.APIKeyEncrypted)
		sb.WriteString(fmt.Sprintf("| %-15s | %-11s | %-40s |\n",
			truncate(srv.Name, 15),
			truncate(srv.ProviderType, 11),
			truncate(maskedKey, 40)))
	}
	sb.WriteString("+-----------------+-------------+------------------------------------------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(servers)))

	return sb.String(), nil
}

func (a *Admin) executeShowHealthStatus() (string, error) {
	status := a.health.GetAllStatus()

	var sb strings.Builder
	sb.WriteString("+-----------------+----------+------------+\n")
	sb.WriteString("| server          | healthy  | latency_ms |\n")
	sb.WriteString("+-----------------+----------+------------+\n")

	for name, s := range status {
		healthy := "Yes"
		if !s.Healthy {
			healthy = "No"
		}
		latencyMs := s.Latency.Milliseconds()
		sb.WriteString(fmt.Sprintf("| %-15s | %-8s | %10d |\n",
			truncate(name, 15),
			healthy,
			latencyMs))
	}
	sb.WriteString("+-----------------+----------+------------+\n")
	sb.WriteString(fmt.Sprintf("%d rows in set\n", len(status)))

	return sb.String(), nil
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (a *Admin) executeInsertServer(ctx context.Context, command string) (string, error) {
	// Parse: INSERT INTO ai_servers (cols) VALUES (vals)
	// Simple parser for: INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, hostgroup, weight) VALUES ('name', 'type', 'url', 'key', 0, 1)
	
	upper := strings.ToUpper(command)
	valuesIdx := strings.Index(upper, "VALUES")
	if valuesIdx == -1 {
		return "", fmt.Errorf("invalid INSERT syntax: missing VALUES")
	}

	valuesPart := command[valuesIdx+6:]
	valuesPart = strings.TrimSpace(valuesPart)
	
	// Remove parentheses
	valuesPart = strings.TrimPrefix(valuesPart, "(")
	valuesPart = strings.TrimSuffix(valuesPart, ")")
	valuesPart = strings.TrimSuffix(valuesPart, ";")
	valuesPart = strings.TrimSuffix(valuesPart, ")")

	// Parse values
	values := parseCSVValues(valuesPart)
	if len(values) < 4 {
		return "", fmt.Errorf("invalid INSERT: need at least name, provider_type, endpoint, api_key_encrypted")
	}

	srv := &storage.Server{
		Name:            unquote(values[0]),
		ProviderType:    unquote(values[1]),
		Endpoint:        unquote(values[2]),
		APIKeyEncrypted: unquote(values[3]),
		Hostgroup:       0,
		Weight:          1,
		MaxConnections:  100,
		Status:          storage.ServerStatusOnline,
	}

	if len(values) > 4 {
		fmt.Sscanf(values[4], "%d", &srv.Hostgroup)
	}
	if len(values) > 5 {
		fmt.Sscanf(values[5], "%d", &srv.Weight)
	}

	if err := a.storage.InsertServer(ctx, srv); err != nil {
		return "", err
	}

	return fmt.Sprintf("Query OK, 1 row affected (server '%s' added)\n", srv.Name), nil
}

func (a *Admin) executeDeleteServer(ctx context.Context, command string) (string, error) {
	// Parse: DELETE FROM ai_servers WHERE name = 'xxx'
	upper := strings.ToUpper(command)
	whereIdx := strings.Index(upper, "WHERE")
	if whereIdx == -1 {
		return "", fmt.Errorf("invalid DELETE syntax: missing WHERE clause (required for safety)")
	}

	wherePart := command[whereIdx+5:]
	wherePart = strings.TrimSpace(wherePart)
	wherePart = strings.TrimSuffix(wherePart, ";")

	// Parse: name = 'xxx'
	if !strings.Contains(strings.ToUpper(wherePart), "NAME") {
		return "", fmt.Errorf("DELETE only supports WHERE name = 'value'")
	}

	parts := strings.SplitN(wherePart, "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid WHERE clause")
	}

	name := strings.TrimSpace(parts[1])
	name = unquote(name)

	if err := a.storage.DeleteServer(ctx, name); err != nil {
		return "", err
	}

	return fmt.Sprintf("Query OK, 1 row affected (server '%s' deleted)\n", name), nil
}

func parseCSVValues(s string) []string {
	var values []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		
		if !inQuote && (c == '\'' || c == '"') {
			inQuote = true
			quoteChar = c
			current.WriteByte(c)
		} else if inQuote && c == quoteChar {
			inQuote = false
			current.WriteByte(c)
		} else if !inQuote && c == ',' {
			values = append(values, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(c)
		}
	}
	
	if current.Len() > 0 {
		values = append(values, strings.TrimSpace(current.String()))
	}

	return values
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func (a *Admin) executeLoadServers(ctx context.Context) (string, error) {
	if err := a.balancer.LoadServers(ctx); err != nil {
		return "", err
	}
	return "Query OK, servers loaded to runtime\n", nil
}

func (a *Admin) executeLoadRules(ctx context.Context) (string, error) {
	if err := a.router.LoadRules(ctx); err != nil {
		return "", err
	}
	return "Query OK, routing rules loaded to runtime\n", nil
}

func (a *Admin) executeSet(command string) (string, error) {
	// Parse: SET variable-name = value
	parts := strings.SplitN(command[4:], "=", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid SET syntax")
	}

	name := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if err := a.config.Set(name, value); err != nil {
		return "", err
	}

	return fmt.Sprintf("Query OK, %s set to %s\n", name, value), nil
}

func (a *Admin) executeShowCacheStatus() (string, error) {
	if a.cache == nil {
		return "Cache not initialized\n", nil
	}

	stats := a.cache.Stats()
	enabled := "disabled"
	if a.cache.IsEnabled() {
		enabled = "enabled"
	}

	var sb strings.Builder
	sb.WriteString("+------------------+------------------+\n")
	sb.WriteString("| Variable         | Value            |\n")
	sb.WriteString("+------------------+------------------+\n")
	sb.WriteString(fmt.Sprintf("| %-16s | %-16s |\n", "status", enabled))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16d |\n", "hits", stats.Hits))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16d |\n", "misses", stats.Misses))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16.2f |\n", "hit_rate", a.cache.HitRate()))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16d |\n", "evictions", stats.Evictions))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16d |\n", "size_bytes", stats.Size))
	sb.WriteString(fmt.Sprintf("| %-16s | %-16d |\n", "item_count", stats.ItemCount))
	sb.WriteString("+------------------+------------------+\n")

	return sb.String(), nil
}

func (a *Admin) executeCacheEnable(enable bool) (string, error) {
	if a.cache == nil {
		return "", fmt.Errorf("cache not initialized")
	}

	a.cache.SetEnabled(enable)
	status := "disabled"
	if enable {
		status = "enabled"
	}
	return fmt.Sprintf("Query OK, cache %s\n", status), nil
}

func (a *Admin) executeCacheClear() (string, error) {
	if a.cache == nil {
		return "", fmt.Errorf("cache not initialized")
	}

	a.cache.Clear()
	return "Query OK, cache cleared\n", nil
}

func (a *Admin) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (a *Admin) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}
