package crypto

import (
	"strings"
	"testing"
)

const testKey = "unit-test-encryption-key-0123456789"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	enc, err := NewEncryptor(testKey)
	if err != nil {
		t.Fatalf("NewEncryptor: %v", err)
	}
	if !enc.IsEnabled() {
		t.Fatal("expected encryption to be enabled with a non-empty key")
	}

	plain := "sk-proj-super-secret-value"
	ct, err := enc.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if ct == plain {
		t.Fatal("ciphertext must not equal plaintext when encryption is enabled")
	}
	if !strings.HasPrefix(ct, "enc:") {
		t.Fatalf("ciphertext missing enc: prefix, got %q", ct)
	}

	got, err := enc.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plain)
	}
}

func TestEncryptNonDeterministic(t *testing.T) {
	enc, _ := NewEncryptor(testKey)
	a, _ := enc.Encrypt("same-input")
	b, _ := enc.Encrypt("same-input")
	if a == b {
		t.Fatal("expected distinct ciphertexts due to random nonce")
	}
}

func TestDisabledEncryptorPassesThrough(t *testing.T) {
	enc, err := NewEncryptor("")
	if err != nil {
		t.Fatalf("NewEncryptor(\"\"): %v", err)
	}
	if enc.IsEnabled() {
		t.Fatal("empty key must yield a disabled encryptor")
	}
	ct, _ := enc.Encrypt("plaintext")
	if ct != "plaintext" {
		t.Fatalf("disabled encryptor should pass through, got %q", ct)
	}
	pt, _ := enc.Decrypt("plaintext")
	if pt != "plaintext" {
		t.Fatalf("disabled decrypt should pass through, got %q", pt)
	}
}

func TestDecryptLegacyPlaintext(t *testing.T) {
	enc, _ := NewEncryptor(testKey)
	// A value without the enc: prefix is treated as legacy plaintext.
	got, err := enc.Decrypt("sk-legacy-plaintext")
	if err != nil {
		t.Fatalf("Decrypt legacy: %v", err)
	}
	if got != "sk-legacy-plaintext" {
		t.Fatalf("got %q", got)
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	enc, _ := NewEncryptor(testKey)
	if _, err := enc.Decrypt("enc:not-valid-base64!!!"); err == nil {
		t.Fatal("expected error decoding corrupted base64")
	}
	if _, err := enc.Decrypt("enc:AAAA"); err == nil {
		t.Fatal("expected error on too-short ciphertext")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	enc1, _ := NewEncryptor(testKey)
	enc2, _ := NewEncryptor("a-different-key-9876543210abcdef")
	ct, _ := enc1.Encrypt("secret")
	if _, err := enc2.Decrypt(ct); err == nil {
		t.Fatal("decrypting with the wrong key must fail (GCM auth)")
	}
}

func TestShortKeyRejected(t *testing.T) {
	if _, err := NewEncryptor("tooshort"); err == nil {
		t.Fatal("expected ErrInvalidKey for short key")
	}
}

func TestMaskAPIKey(t *testing.T) {
	cases := map[string]string{
		"":                         "",
		"enc:abcdef":               "[encrypted]",
		"short":                    "****",
		"sk-proj-1234567890abcdef": "sk-proj-...cdef",
	}
	for in, want := range cases {
		if got := MaskAPIKey(in); got != want {
			t.Errorf("MaskAPIKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGenerateKeyValid(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if err := ValidateKey(k); err != nil {
		t.Fatalf("generated key failed ValidateKey: %v", err)
	}
	enc, err := NewEncryptor(k)
	if err != nil || !enc.IsEnabled() {
		t.Fatalf("generated key not usable: %v", err)
	}
}
