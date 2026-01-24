// Package crypto provides encryption utilities for sensitive data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

var (
	ErrInvalidKey         = errors.New("invalid encryption key: must be at least 16 characters")
	ErrInvalidCiphertext  = errors.New("invalid ciphertext")
	ErrEncryptionDisabled = errors.New("encryption is disabled: no key configured")
)

// Encryptor handles encryption and decryption of sensitive data.
type Encryptor struct {
	mu      sync.RWMutex
	key     []byte
	enabled bool
}

// NewEncryptor creates a new encryptor with the given key.
// The key is hashed using SHA-256 to ensure a valid AES-256 key size.
func NewEncryptor(key string) (*Encryptor, error) {
	if key == "" {
		// Encryption disabled
		return &Encryptor{enabled: false}, nil
	}

	if len(key) < 16 {
		return nil, ErrInvalidKey
	}

	// Hash the key to get exactly 32 bytes for AES-256
	hash := sha256.Sum256([]byte(key))
	return &Encryptor{
		key:     hash[:],
		enabled: true,
	}, nil
}

// IsEnabled returns whether encryption is enabled.
func (e *Encryptor) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns base64-encoded ciphertext with "enc:" prefix.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		// Return plaintext if encryption is disabled
		return plaintext, nil
	}

	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return "enc:" + encoded, nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// Handles both encrypted (with "enc:" prefix) and plaintext values.
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if ciphertext == "" {
		return "", nil
	}

	// Check if this is encrypted data
	if !strings.HasPrefix(ciphertext, "enc:") {
		// Return as-is (legacy plaintext or encryption disabled)
		return ciphertext, nil
	}

	if !e.enabled {
		return "", ErrEncryptionDisabled
	}

	// Remove prefix and decode base64
	encoded := strings.TrimPrefix(ciphertext, "enc:")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// RotateKey re-encrypts data with a new key.
// Returns the new encrypted value.
func (e *Encryptor) RotateKey(currentCiphertext, newKey string) (string, error) {
	// Decrypt with current key
	plaintext, err := e.Decrypt(currentCiphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt with current key: %w", err)
	}

	// Create new encryptor with new key
	newEnc, err := NewEncryptor(newKey)
	if err != nil {
		return "", fmt.Errorf("failed to create new encryptor: %w", err)
	}

	// Encrypt with new key
	return newEnc.Encrypt(plaintext)
}

// MaskAPIKey returns a masked version of an API key for display.
func MaskAPIKey(key string) string {
	if key == "" {
		return ""
	}

	// Handle encrypted keys
	if strings.HasPrefix(key, "enc:") {
		return "[encrypted]"
	}

	// Show first 8 and last 4 characters
	if len(key) <= 12 {
		return "****"
	}

	return key[:8] + "..." + key[len(key)-4:]
}

// GenerateKey generates a random encryption key.
func GenerateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// ValidateKey checks if a key is valid for encryption.
func ValidateKey(key string) error {
	if len(key) < 16 {
		return ErrInvalidKey
	}
	return nil
}

// Global encryptor instance
var globalEncryptor *Encryptor
var globalEncryptorOnce sync.Once

// InitGlobalEncryptor initializes the global encryptor.
func InitGlobalEncryptor(key string) error {
	var initErr error
	globalEncryptorOnce = sync.Once{} // Reset for re-initialization
	globalEncryptorOnce.Do(func() {
		globalEncryptor, initErr = NewEncryptor(key)
	})
	return initErr
}

// GlobalEncryptor returns the global encryptor instance.
func GlobalEncryptor() *Encryptor {
	if globalEncryptor == nil {
		// Return a disabled encryptor
		globalEncryptor = &Encryptor{enabled: false}
	}
	return globalEncryptor
}

// EncryptAPIKey encrypts an API key using the global encryptor.
func EncryptAPIKey(key string) (string, error) {
	return GlobalEncryptor().Encrypt(key)
}

// DecryptAPIKey decrypts an API key using the global encryptor.
func DecryptAPIKey(key string) (string, error) {
	return GlobalEncryptor().Decrypt(key)
}
