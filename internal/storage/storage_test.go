package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func newTestStorage(t *testing.T, encryptionKey string) *Storage {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := NewWithEncryption(dbPath, encryptionKey)
	if err != nil {
		t.Fatalf("NewWithEncryption: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func sampleServer(name string) *Server {
	return &Server{
		Name:            name,
		ProviderType:    "openai",
		Endpoint:        "https://api.openai.com",
		APIKeyEncrypted: "sk-plaintext-secret-value",
		Hostgroup:       0,
		Weight:          1,
		MaxConnections:  100,
		Status:          ServerStatusOnline,
	}
}

// TestServerKeyEncryptionRoundTrip is the regression test for the hot-path key
// bug: with encryption enabled, GetServers must return ciphertext (safe for the
// admin/display path) while GetServersWithDecryptedKeys must return the original
// plaintext (what providers use as the upstream credential).
func TestServerKeyEncryptionRoundTrip(t *testing.T) {
	ctx := context.Background()
	st := newTestStorage(t, "unit-test-encryption-key-0123456789")

	if !st.IsEncryptionEnabled() {
		t.Fatal("expected encryption to be enabled")
	}

	if err := st.InsertServer(ctx, sampleServer("openai-main")); err != nil {
		t.Fatalf("InsertServer: %v", err)
	}

	// Admin/display path: encrypted.
	enc, err := st.GetServers(ctx, nil)
	if err != nil {
		t.Fatalf("GetServers: %v", err)
	}
	if len(enc) != 1 {
		t.Fatalf("expected 1 server, got %d", len(enc))
	}
	if !strings.HasPrefix(enc[0].APIKeyEncrypted, "enc:") {
		t.Fatalf("GetServers should return ciphertext, got %q", enc[0].APIKeyEncrypted)
	}
	if enc[0].APIKeyEncrypted == "sk-plaintext-secret-value" {
		t.Fatal("GetServers leaked plaintext API key")
	}

	// Hot path: decrypted plaintext for provider auth.
	dec, err := st.GetServersWithDecryptedKeys(ctx, nil)
	if err != nil {
		t.Fatalf("GetServersWithDecryptedKeys: %v", err)
	}
	if dec[0].APIKeyEncrypted != "sk-plaintext-secret-value" {
		t.Fatalf("hot path should return plaintext, got %q", dec[0].APIKeyEncrypted)
	}

	// Single-server variant must behave the same way.
	one, err := st.GetServerWithDecryptedKey(ctx, "openai-main")
	if err != nil {
		t.Fatalf("GetServerWithDecryptedKey: %v", err)
	}
	if one.APIKeyEncrypted != "sk-plaintext-secret-value" {
		t.Fatalf("GetServerWithDecryptedKey should return plaintext, got %q", one.APIKeyEncrypted)
	}
}

func TestServerCRUD(t *testing.T) {
	ctx := context.Background()
	st := newTestStorage(t, "")

	if err := st.InsertServer(ctx, sampleServer("s1")); err != nil {
		t.Fatalf("InsertServer: %v", err)
	}
	got, err := st.GetServerByName(ctx, "s1")
	if err != nil {
		t.Fatalf("GetServerByName: %v", err)
	}
	if got == nil || got.Name != "s1" || got.ProviderType != "openai" {
		t.Fatalf("unexpected server: %+v", got)
	}

	// With encryption disabled, the key is stored as-is.
	if got.APIKeyEncrypted != "sk-plaintext-secret-value" {
		t.Fatalf("expected plaintext passthrough, got %q", got.APIKeyEncrypted)
	}

	if err := st.DeleteServer(ctx, "s1"); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}
	after, err := st.GetServers(ctx, nil)
	if err != nil {
		t.Fatalf("GetServers: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("expected 0 servers after delete, got %d", len(after))
	}
}
