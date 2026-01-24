// Package storage provides SQLite-based persistent storage for MindBalancer.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mindbalancer/mindbalancer/internal/crypto"
)

// Storage handles all database operations.
type Storage struct {
	db        *sql.DB
	mu        sync.RWMutex
	encryptor *crypto.Encryptor
}

// ServerStatus represents the status of an AI server.
type ServerStatus string

const (
	ServerStatusOnline   ServerStatus = "ONLINE"
	ServerStatusOffline  ServerStatus = "OFFLINE"
	ServerStatusShunned  ServerStatus = "SHUNNED"
)

// Server represents an AI server/provider.
type Server struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	ProviderType    string       `json:"provider_type"` // openai, anthropic, ollama, azure, groq, google, bedrock, custom
	Endpoint        string       `json:"endpoint"`
	APIKeyEncrypted string       `json:"api_key_encrypted,omitempty"`
	Hostgroup       int          `json:"hostgroup"`
	Weight          int          `json:"weight"`
	MaxConnections  int          `json:"max_connections"`
	Status          ServerStatus `json:"status"`
	Comment         string       `json:"comment,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// User represents an API user.
type User struct {
	ID                   int64     `json:"id"`
	Username             string    `json:"username"`
	PasswordHash         string    `json:"password_hash,omitempty"`
	Active               bool      `json:"active"`
	DefaultHostgroup     int       `json:"default_hostgroup"`
	MaxRequestsPerMinute int       `json:"max_requests_per_minute"`
	MaxTokensPerMinute   int       `json:"max_tokens_per_minute"`
	Comment              string    `json:"comment,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// RoutingRule represents a request routing rule.
type RoutingRule struct {
	ID                   int64     `json:"id"`
	RuleID               int       `json:"rule_id"`
	Active               bool      `json:"active"`
	MatchModel           string    `json:"match_model,omitempty"`
	MatchPattern         string    `json:"match_pattern,omitempty"`
	MatchUser            string    `json:"match_user,omitempty"`
	DestinationHostgroup int       `json:"destination_hostgroup"`
	MirrorHostgroup      *int      `json:"mirror_hostgroup,omitempty"`
	Priority             int       `json:"priority"`
	Comment              string    `json:"comment,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Hostgroup represents a logical group of servers.
type Hostgroup struct {
	ID      int64
	GroupID int
	Name    string
	Comment string
}

// RequestLog represents a logged request.
type RequestLog struct {
	ID             int64
	RequestID      string
	Username       string
	ServerName     string
	Model          string
	Endpoint       string
	PromptTokens   int
	OutputTokens   int
	TotalTokens    int
	LatencyMS      int64
	FirstByteMS    int64
	StatusCode     int
	ErrorMessage   string
	Streaming      bool
	Timestamp      time.Time
}

// New creates a new Storage instance.
func New(dbPath string) (*Storage, error) {
	return NewWithEncryption(dbPath, "")
}

// NewWithEncryption creates a new Storage instance with API key encryption.
func NewWithEncryption(dbPath, encryptionKey string) (*Storage, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single connection
	db.SetMaxIdleConns(1)

	// Create encryptor
	encryptor, err := crypto.NewEncryptor(encryptionKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	s := &Storage{db: db, encryptor: encryptor}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// migrate creates the necessary tables.
func (s *Storage) migrate() error {
	schema := `
	-- AI Servers (providers)
	CREATE TABLE IF NOT EXISTS ai_servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		provider_type TEXT NOT NULL DEFAULT 'openai',
		endpoint TEXT NOT NULL,
		api_key_encrypted TEXT,
		hostgroup INTEGER NOT NULL DEFAULT 0,
		weight INTEGER NOT NULL DEFAULT 1,
		max_connections INTEGER NOT NULL DEFAULT 100,
		status TEXT NOT NULL DEFAULT 'ONLINE',
		comment TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- AI Users
	CREATE TABLE IF NOT EXISTS ai_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT,
		active INTEGER NOT NULL DEFAULT 1,
		default_hostgroup INTEGER NOT NULL DEFAULT 0,
		max_requests_per_minute INTEGER NOT NULL DEFAULT 1000,
		max_tokens_per_minute INTEGER NOT NULL DEFAULT 100000,
		comment TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Routing Rules
	CREATE TABLE IF NOT EXISTS ai_routing_rules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		rule_id INTEGER UNIQUE,
		active INTEGER NOT NULL DEFAULT 1,
		match_model TEXT,
		match_pattern TEXT,
		match_user TEXT,
		destination_hostgroup INTEGER NOT NULL,
		mirror_hostgroup INTEGER,
		priority INTEGER NOT NULL DEFAULT 100,
		comment TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Hostgroups
	CREATE TABLE IF NOT EXISTS ai_hostgroups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER UNIQUE NOT NULL,
		name TEXT,
		comment TEXT
	);

	-- Request Logs
	CREATE TABLE IF NOT EXISTS ai_request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT,
		username TEXT,
		server_name TEXT,
		model TEXT,
		endpoint TEXT,
		prompt_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		latency_ms INTEGER DEFAULT 0,
		first_byte_ms INTEGER DEFAULT 0,
		status_code INTEGER,
		error_message TEXT,
		streaming INTEGER DEFAULT 0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Global Variables (runtime config)
	CREATE TABLE IF NOT EXISTS global_variables (
		variable_name TEXT PRIMARY KEY,
		variable_value TEXT
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_servers_hostgroup ON ai_servers(hostgroup);
	CREATE INDEX IF NOT EXISTS idx_servers_status ON ai_servers(status);
	CREATE INDEX IF NOT EXISTS idx_users_username ON ai_users(username);
	CREATE INDEX IF NOT EXISTS idx_routing_rules_priority ON ai_routing_rules(priority);
	CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON ai_request_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_request_logs_username ON ai_request_logs(username);
	CREATE INDEX IF NOT EXISTS idx_request_logs_server ON ai_request_logs(server_name);

	-- Default hostgroup
	INSERT OR IGNORE INTO ai_hostgroups (group_id, name, comment) VALUES (0, 'default', 'Default hostgroup');
	`

	_, err := s.db.Exec(schema)
	return err
}

// Server operations

// GetServers returns all servers, optionally filtered by hostgroup.
func (s *Storage) GetServers(ctx context.Context, hostgroup *int) ([]Server, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT id, name, provider_type, endpoint, api_key_encrypted, hostgroup, 
		weight, max_connections, status, comment, created_at, updated_at 
		FROM ai_servers`
	args := []any{}

	if hostgroup != nil {
		query += " WHERE hostgroup = ?"
		args = append(args, *hostgroup)
	}

	query += " ORDER BY hostgroup, weight DESC, name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []Server
	for rows.Next() {
		var srv Server
		var comment sql.NullString
		err := rows.Scan(&srv.ID, &srv.Name, &srv.ProviderType, &srv.Endpoint,
			&srv.APIKeyEncrypted, &srv.Hostgroup, &srv.Weight, &srv.MaxConnections,
			&srv.Status, &comment, &srv.CreatedAt, &srv.UpdatedAt)
		if err != nil {
			return nil, err
		}
		srv.Comment = comment.String
		servers = append(servers, srv)
	}

	return servers, rows.Err()
}

// GetServerByName returns a server by name.
func (s *Storage) GetServerByName(ctx context.Context, name string) (*Server, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var srv Server
	var comment sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, provider_type, endpoint, api_key_encrypted, hostgroup, 
			weight, max_connections, status, comment, created_at, updated_at 
		FROM ai_servers WHERE name = ?`, name).Scan(
		&srv.ID, &srv.Name, &srv.ProviderType, &srv.Endpoint,
		&srv.APIKeyEncrypted, &srv.Hostgroup, &srv.Weight, &srv.MaxConnections,
		&srv.Status, &comment, &srv.CreatedAt, &srv.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	srv.Comment = comment.String
	return &srv, nil
}

// InsertServer adds a new server.
func (s *Storage) InsertServer(ctx context.Context, srv *Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encrypt API key if provided
	apiKey := srv.APIKeyEncrypted
	if apiKey != "" && s.encryptor != nil {
		encrypted, err := s.encryptor.Encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %w", err)
		}
		apiKey = encrypted
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_servers (name, provider_type, endpoint, api_key_encrypted, 
			hostgroup, weight, max_connections, status, comment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		srv.Name, srv.ProviderType, srv.Endpoint, apiKey,
		srv.Hostgroup, srv.Weight, srv.MaxConnections, srv.Status, srv.Comment)
	if err != nil {
		return err
	}

	srv.ID, _ = result.LastInsertId()
	return nil
}

// UpdateServer updates a server.
func (s *Storage) UpdateServer(ctx context.Context, srv *Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Encrypt API key if provided
	apiKey := srv.APIKeyEncrypted
	if apiKey != "" && s.encryptor != nil {
		encrypted, err := s.encryptor.Encrypt(apiKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %w", err)
		}
		apiKey = encrypted
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE ai_servers SET 
			provider_type = ?, endpoint = ?, api_key_encrypted = ?,
			hostgroup = ?, weight = ?, max_connections = ?, 
			status = ?, comment = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?`,
		srv.ProviderType, srv.Endpoint, apiKey,
		srv.Hostgroup, srv.Weight, srv.MaxConnections,
		srv.Status, srv.Comment, srv.Name)
	return err
}

// DeleteServer deletes a server by name.
func (s *Storage) DeleteServer(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM ai_servers WHERE name = ?", name)
	return err
}

// User operations

// GetUsers returns all users.
func (s *Storage) GetUsers(ctx context.Context) ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, username, password_hash, active, default_hostgroup,
			max_requests_per_minute, max_tokens_per_minute, comment, 
			created_at, updated_at
		FROM ai_users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var comment, pwHash sql.NullString
		err := rows.Scan(&u.ID, &u.Username, &pwHash, &u.Active,
			&u.DefaultHostgroup, &u.MaxRequestsPerMinute, &u.MaxTokensPerMinute,
			&comment, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = pwHash.String
		u.Comment = comment.String
		users = append(users, u)
	}

	return users, rows.Err()
}

// GetUserByUsername returns a user by username.
func (s *Storage) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var u User
	var comment, pwHash sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, active, default_hostgroup,
			max_requests_per_minute, max_tokens_per_minute, comment, 
			created_at, updated_at
		FROM ai_users WHERE username = ?`, username).Scan(
		&u.ID, &u.Username, &pwHash, &u.Active,
		&u.DefaultHostgroup, &u.MaxRequestsPerMinute, &u.MaxTokensPerMinute,
		&comment, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	u.PasswordHash = pwHash.String
	u.Comment = comment.String
	return &u, nil
}

// InsertUser adds a new user.
func (s *Storage) InsertUser(ctx context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_users (username, password_hash, active, default_hostgroup,
			max_requests_per_minute, max_tokens_per_minute, comment)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.Username, u.PasswordHash, u.Active, u.DefaultHostgroup,
		u.MaxRequestsPerMinute, u.MaxTokensPerMinute, u.Comment)
	if err != nil {
		return err
	}

	u.ID, _ = result.LastInsertId()
	return nil
}

// UpdateUser updates a user.
func (s *Storage) UpdateUser(ctx context.Context, u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		UPDATE ai_users SET 
			password_hash = ?, active = ?, default_hostgroup = ?,
			max_requests_per_minute = ?, max_tokens_per_minute = ?,
			comment = ?, updated_at = CURRENT_TIMESTAMP
		WHERE username = ?`,
		u.PasswordHash, u.Active, u.DefaultHostgroup,
		u.MaxRequestsPerMinute, u.MaxTokensPerMinute,
		u.Comment, u.Username)
	return err
}

// DeleteUser deletes a user.
func (s *Storage) DeleteUser(ctx context.Context, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM ai_users WHERE username = ?", username)
	return err
}

// Routing Rule operations

// GetRoutingRules returns all routing rules.
func (s *Storage) GetRoutingRules(ctx context.Context) ([]RoutingRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_id, active, match_model, match_pattern, match_user,
			destination_hostgroup, mirror_hostgroup, priority, comment,
			created_at, updated_at
		FROM ai_routing_rules ORDER BY priority, rule_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RoutingRule
	for rows.Next() {
		var r RoutingRule
		var ruleID, mirrorHG sql.NullInt64
		var matchModel, matchPattern, matchUser, comment sql.NullString

		err := rows.Scan(&r.ID, &ruleID, &r.Active, &matchModel, &matchPattern,
			&matchUser, &r.DestinationHostgroup, &mirrorHG, &r.Priority,
			&comment, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if ruleID.Valid {
			r.RuleID = int(ruleID.Int64)
		}
		r.MatchModel = matchModel.String
		r.MatchPattern = matchPattern.String
		r.MatchUser = matchUser.String
		r.Comment = comment.String
		if mirrorHG.Valid {
			v := int(mirrorHG.Int64)
			r.MirrorHostgroup = &v
		}

		rules = append(rules, r)
	}

	return rules, rows.Err()
}

// InsertRoutingRule adds a new routing rule.
func (s *Storage) InsertRoutingRule(ctx context.Context, r *RoutingRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_routing_rules (rule_id, active, match_model, match_pattern,
			match_user, destination_hostgroup, mirror_hostgroup, priority, comment)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.RuleID, r.Active, r.MatchModel, r.MatchPattern, r.MatchUser,
		r.DestinationHostgroup, r.MirrorHostgroup, r.Priority, r.Comment)
	if err != nil {
		return err
	}

	r.ID, _ = result.LastInsertId()
	return nil
}

// DeleteRoutingRule deletes a routing rule.
func (s *Storage) DeleteRoutingRule(ctx context.Context, ruleID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, "DELETE FROM ai_routing_rules WHERE rule_id = ?", ruleID)
	return err
}

// Request Log operations

// InsertRequestLog logs a request.
func (s *Storage) InsertRequestLog(ctx context.Context, log *RequestLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_request_logs (request_id, username, server_name, model,
			endpoint, prompt_tokens, output_tokens, total_tokens, latency_ms,
			first_byte_ms, status_code, error_message, streaming, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.RequestID, log.Username, log.ServerName, log.Model,
		log.Endpoint, log.PromptTokens, log.OutputTokens, log.TotalTokens,
		log.LatencyMS, log.FirstByteMS, log.StatusCode, log.ErrorMessage,
		log.Streaming, log.Timestamp)
	return err
}

// GetRecentLogs returns recent request logs.
func (s *Storage) GetRecentLogs(ctx context.Context, limit int) ([]RequestLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, request_id, username, server_name, model, endpoint,
			prompt_tokens, output_tokens, total_tokens, latency_ms, first_byte_ms,
			status_code, error_message, streaming, timestamp
		FROM ai_request_logs ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var l RequestLog
		var reqID, username, serverName, model, endpoint, errorMsg sql.NullString

		err := rows.Scan(&l.ID, &reqID, &username, &serverName, &model, &endpoint,
			&l.PromptTokens, &l.OutputTokens, &l.TotalTokens, &l.LatencyMS,
			&l.FirstByteMS, &l.StatusCode, &errorMsg, &l.Streaming, &l.Timestamp)
		if err != nil {
			return nil, err
		}

		l.RequestID = reqID.String
		l.Username = username.String
		l.ServerName = serverName.String
		l.Model = model.String
		l.Endpoint = endpoint.String
		l.ErrorMessage = errorMsg.String

		logs = append(logs, l)
	}

	return logs, rows.Err()
}

// GetServerStats returns statistics for servers.
func (s *Storage) GetServerStats(ctx context.Context) ([]map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			server_name,
			COUNT(*) as total_requests,
			SUM(CASE WHEN status_code >= 200 AND status_code < 300 THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count,
			AVG(latency_ms) as avg_latency_ms,
			SUM(total_tokens) as total_tokens,
			MAX(timestamp) as last_request
		FROM ai_request_logs
		WHERE timestamp > datetime('now', '-1 hour')
		GROUP BY server_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]any
	for rows.Next() {
		var serverName string
		var totalReqs, successCount, errorCount, totalTokens int64
		var avgLatency float64
		var lastRequest time.Time

		err := rows.Scan(&serverName, &totalReqs, &successCount, &errorCount,
			&avgLatency, &totalTokens, &lastRequest)
		if err != nil {
			return nil, err
		}

		stats = append(stats, map[string]any{
			"server_name":    serverName,
			"total_requests": totalReqs,
			"success_count":  successCount,
			"error_count":    errorCount,
			"avg_latency_ms": avgLatency,
			"total_tokens":   totalTokens,
			"last_request":   lastRequest,
		})
	}

	return stats, rows.Err()
}

// Hostgroup operations

// GetHostgroups returns all hostgroups.
func (s *Storage) GetHostgroups(ctx context.Context) ([]Hostgroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, group_id, name, comment FROM ai_hostgroups ORDER BY group_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hostgroups []Hostgroup
	for rows.Next() {
		var hg Hostgroup
		var name, comment sql.NullString
		err := rows.Scan(&hg.ID, &hg.GroupID, &name, &comment)
		if err != nil {
			return nil, err
		}
		hg.Name = name.String
		hg.Comment = comment.String
		hostgroups = append(hostgroups, hg)
	}

	return hostgroups, rows.Err()
}

// InsertHostgroup adds a new hostgroup.
func (s *Storage) InsertHostgroup(ctx context.Context, hg *Hostgroup) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO ai_hostgroups (group_id, name, comment)
		VALUES (?, ?, ?)`, hg.GroupID, hg.Name, hg.Comment)
	if err != nil {
		return err
	}

	hg.ID, _ = result.LastInsertId()
	return nil
}

// Global Variables

// GetVariable returns a global variable value.
func (s *Storage) GetVariable(ctx context.Context, name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var value string
	err := s.db.QueryRowContext(ctx,
		"SELECT variable_value FROM global_variables WHERE variable_name = ?", name).
		Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetVariable sets a global variable.
func (s *Storage) SetVariable(ctx context.Context, name, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO global_variables (variable_name, variable_value)
		VALUES (?, ?)
		ON CONFLICT(variable_name) DO UPDATE SET variable_value = excluded.variable_value`,
		name, value)
	return err
}

// GetAllVariables returns all global variables.
func (s *Storage) GetAllVariables(ctx context.Context) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, "SELECT variable_name, variable_value FROM global_variables")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vars := make(map[string]string)
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		vars[name] = value
	}

	return vars, rows.Err()
}

// DB returns the underlying database connection for advanced queries.
func (s *Storage) DB() *sql.DB {
	return s.db
}

// DecryptAPIKey decrypts an API key using the storage's encryptor.
func (s *Storage) DecryptAPIKey(encryptedKey string) (string, error) {
	if encryptedKey == "" {
		return "", nil
	}
	if s.encryptor == nil {
		return encryptedKey, nil
	}
	return s.encryptor.Decrypt(encryptedKey)
}

// EncryptAPIKey encrypts an API key using the storage's encryptor.
func (s *Storage) EncryptAPIKey(plainKey string) (string, error) {
	if plainKey == "" {
		return "", nil
	}
	if s.encryptor == nil {
		return plainKey, nil
	}
	return s.encryptor.Encrypt(plainKey)
}

// GetServerWithDecryptedKey returns a server with the API key decrypted.
func (s *Storage) GetServerWithDecryptedKey(ctx context.Context, name string) (*Server, error) {
	srv, err := s.GetServerByName(ctx, name)
	if err != nil || srv == nil {
		return srv, err
	}

	if srv.APIKeyEncrypted != "" && s.encryptor != nil {
		decrypted, err := s.encryptor.Decrypt(srv.APIKeyEncrypted)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt API key: %w", err)
		}
		srv.APIKeyEncrypted = decrypted
	}
	return srv, nil
}

// GetServersWithDecryptedKeys returns all servers with decrypted API keys.
func (s *Storage) GetServersWithDecryptedKeys(ctx context.Context, hostgroup *int) ([]Server, error) {
	servers, err := s.GetServers(ctx, hostgroup)
	if err != nil {
		return nil, err
	}

	if s.encryptor == nil {
		return servers, nil
	}

	for i := range servers {
		if servers[i].APIKeyEncrypted != "" {
			decrypted, err := s.encryptor.Decrypt(servers[i].APIKeyEncrypted)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt API key for %s: %w", servers[i].Name, err)
			}
			servers[i].APIKeyEncrypted = decrypted
		}
	}
	return servers, nil
}

// MaskAPIKey returns a masked version of an API key.
func (s *Storage) MaskAPIKey(key string) string {
	return crypto.MaskAPIKey(key)
}

// IsEncryptionEnabled returns whether encryption is enabled.
func (s *Storage) IsEncryptionEnabled() bool {
	return s.encryptor != nil && s.encryptor.IsEnabled()
}
