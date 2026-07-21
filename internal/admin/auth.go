// Package admin provides authentication and session management.
package admin

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents an authenticated session.
type Session struct {
	ID        string
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SessionManager manages user sessions.
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	sessionTTL   time.Duration
	cookieName   string
	cookieSecure bool
}

// NewSessionManager creates a new session manager. When secure is true the
// session cookie is marked Secure (send only over HTTPS); pass cfg.TLSEnabled.
func NewSessionManager(secure bool) *SessionManager {
	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		sessionTTL:   24 * time.Hour,
		cookieName:   "mindbalancer_session",
		cookieSecure: secure,
	}

	// Start cleanup goroutine
	go sm.cleanupExpired()

	return sm
}

// cleanupExpired removes expired sessions periodically.
func (sm *SessionManager) cleanupExpired() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
			}
		}
		sm.mu.Unlock()
	}
}

// CreateSession creates a new session for the given username.
func (sm *SessionManager) CreateSession(username string) (*Session, error) {
	// Generate random session ID
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	sessionID := base64.URLEncoding.EncodeToString(b)

	now := time.Now()
	session := &Session{
		ID:        sessionID,
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.sessionTTL),
	}

	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID.
func (sm *SessionManager) GetSession(sessionID string) *Session {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		sm.DeleteSession(sessionID)
		return nil
	}

	return session
}

// DeleteSession removes a session.
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mu.Lock()
	delete(sm.sessions, sessionID)
	sm.mu.Unlock()
}

// SetSessionCookie sets the session cookie on the response.
func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     sm.cookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   sm.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie.
func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sm.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sm.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetSessionFromRequest extracts the session from the request cookie.
func (sm *SessionManager) GetSessionFromRequest(r *http.Request) *Session {
	cookie, err := r.Cookie(sm.cookieName)
	if err != nil {
		return nil
	}
	return sm.GetSession(cookie.Value)
}

// AuthMiddleware creates HTTP middleware that requires authentication.
func (sm *SessionManager) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sm.GetSessionFromRequest(r)
		if session == nil {
			// Redirect to login page
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

// AuthAPIMiddleware creates HTTP middleware for API endpoints that requires authentication.
// Returns 401 instead of redirecting.
func (sm *SessionManager) AuthAPIMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := sm.GetSessionFromRequest(r)
		if session == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Unauthorized. Please login at /admin/login"}`))
			return
		}

		// Continue to the next handler
		next.ServeHTTP(w, r)
	})
}

// Authenticator handles user authentication.
type Authenticator struct {
	adminUsername     string
	adminPasswordHash string
	generatedPassword string // non-empty only when we auto-generated a password
}

// NewAuthenticator creates a new authenticator with the given credentials.
//
// Security: authentication is fail-closed. If no password hash is configured we
// do NOT allow "any password" (the old dev-mode behavior). Instead we generate a
// random one-time password, hash it, and expose it via GeneratedPassword() so the
// server can log it on startup. Operators should set admin_password_hash for a
// stable credential (see `mindbalancer -hash-password`).
func NewAuthenticator(username, passwordHash string) *Authenticator {
	a := &Authenticator{
		adminUsername:     username,
		adminPasswordHash: passwordHash,
	}

	if passwordHash == "" {
		pw, err := generateRandomPassword()
		if err != nil {
			// Extremely unlikely; degrade to a disabled (fail-closed) authenticator.
			log.Printf("admin auth: failed to generate bootstrap password: %v", err)
			return a
		}
		hash, err := HashPassword(pw)
		if err != nil {
			log.Printf("admin auth: failed to hash bootstrap password: %v", err)
			return a
		}
		a.adminPasswordHash = hash
		a.generatedPassword = pw
	}

	return a
}

// GeneratedPassword returns the auto-generated bootstrap password, or "" if a
// password hash was configured. Log this once at startup.
func (a *Authenticator) GeneratedPassword() string {
	return a.generatedPassword
}

// Authenticate verifies the username and password.
func (a *Authenticator) Authenticate(username, password string) bool {
	// Check username
	if username != a.adminUsername {
		return false
	}

	// Fail closed if, for some reason, no hash is available.
	if a.adminPasswordHash == "" {
		return false
	}

	// Verify password using bcrypt (constant-time comparison internally).
	err := bcrypt.CompareHashAndPassword([]byte(a.adminPasswordHash), []byte(password))
	return err == nil
}

// generateRandomPassword returns a URL-safe random password.
func generateRandomPassword() (string, error) {
	b := make([]byte, 18) // 144 bits of entropy
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashPassword generates a bcrypt hash for the given password.
// This can be used to generate password hashes for configuration.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// IsAuthConfigured checks if authentication is configured.
func (a *Authenticator) IsAuthConfigured() bool {
	return a.adminPasswordHash != ""
}

// loginHTML contains the login page HTML.
const loginHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MindBalancer - Login</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f5f5f5;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            padding: 40px;
            width: 100%;
            max-width: 400px;
        }
        h1 {
            text-align: center;
            margin-bottom: 8px;
            color: #333;
        }
        p {
            text-align: center;
            color: #666;
            margin-bottom: 24px;
        }
        .form-group {
            margin-bottom: 16px;
        }
        label {
            display: block;
            margin-bottom: 6px;
            font-weight: 500;
            color: #333;
        }
        input[type="text"], input[type="password"] {
            width: 100%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 16px;
        }
        input:focus {
            outline: none;
            border-color: #1a73e8;
        }
        button {
            width: 100%;
            padding: 12px;
            background: #1a73e8;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 16px;
            cursor: pointer;
            margin-top: 8px;
        }
        button:hover {
            background: #1557b0;
        }
        .hint {
            font-size: 12px;
            color: #888;
            margin-top: 4px;
        }
        .error {
            background: #ffebee;
            color: #c62828;
            padding: 12px;
            border-radius: 4px;
            margin-bottom: 16px;
            display: none;
        }
        .links {
            text-align: center;
            margin-top: 20px;
        }
        .links a {
            color: #1a73e8;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <div class="card">
        <div style="text-align:center;margin-bottom:24px;">
            <img src="/static/logo-mindbalancer.png" alt="MindBalancer" style="width:64px;height:64px;border-radius:12px;margin-bottom:12px;">
            <h1>MindBalancer</h1>
        </div>
        <p>Admin Panel Login</p>
        
        <form method="POST" action="/admin/login">
            <div class="form-group">
                <label>Username</label>
                <input type="text" name="username" value="admin" required>
            </div>
            <div class="form-group">
                <label>Password</label>
                <input type="password" name="password" placeholder="Password" required>
            </div>
            <button type="submit">Sign In</button>
        </form>
        
        <div class="links">
            <a href="/monitoring">Go to Monitoring</a>
        </div>
    </div>
</body>
</html>`
