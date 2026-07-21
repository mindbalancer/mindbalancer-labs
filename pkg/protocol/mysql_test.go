package protocol

import (
	"crypto/sha1"
	"testing"
)

// clientNativePassword reproduces the mysql_native_password client scramble that
// go-sql-driver sends, so we can assert the server verifier accepts it.
func clientNativePassword(password string, scramble []byte) []byte {
	if password == "" {
		return nil
	}
	stage1 := sha1.Sum([]byte(password))
	stage2 := sha1.Sum(stage1[:])
	h := sha1.New()
	h.Write(scramble)
	h.Write(stage2[:])
	inner := h.Sum(nil)
	out := make([]byte, sha1.Size)
	for i := range out {
		out[i] = stage1[i] ^ inner[i]
	}
	return out
}

func TestVerifyNativePassword(t *testing.T) {
	scramble := []byte("0123456789abcdefghij") // 20 bytes
	const pw = "testpass123"

	if !verifyNativePassword(pw, scramble, clientNativePassword(pw, scramble)) {
		t.Fatal("correct password should verify")
	}
	if verifyNativePassword(pw, scramble, clientNativePassword("wrongpass", scramble)) {
		t.Fatal("wrong password must not verify")
	}
	if verifyNativePassword(pw, scramble, nil) {
		t.Fatal("empty auth response must not verify against a non-empty password")
	}
	// Different scramble => response computed for another challenge must fail.
	other := []byte("zyxwvutsrqponmlkjihg")
	if verifyNativePassword(pw, scramble, clientNativePassword(pw, other)) {
		t.Fatal("response for a different scramble must not verify (replay protection)")
	}
}

func TestVerifyNativePasswordEmptyCredential(t *testing.T) {
	scramble := []byte("0123456789abcdefghij")
	// An empty configured password accepts an empty auth response.
	if !verifyNativePassword("", scramble, nil) {
		t.Fatal("empty password should accept empty auth response")
	}
}

func TestParseHandshakeResponse(t *testing.T) {
	// Build a minimal protocol-41 response: 32-byte fixed header, username, then
	// a 1-byte-length-prefixed auth response.
	buf := make([]byte, 32)
	buf = append(buf, []byte("admin")...)
	buf = append(buf, 0) // username terminator
	auth := []byte("\x01\x02\x03\x04")
	buf = append(buf, byte(len(auth)))
	buf = append(buf, auth...)

	user, resp, err := parseHandshakeResponse(buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if user != "admin" {
		t.Fatalf("username = %q, want admin", user)
	}
	if string(resp) != string(auth) {
		t.Fatalf("authResp = %x, want %x", resp, auth)
	}
}
