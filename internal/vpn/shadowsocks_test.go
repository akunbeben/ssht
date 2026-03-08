package vpn

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestEvpBytesToKey(t *testing.T) {
	password := "testpassword"
	keyLen := 32
	key := evpBytesToKey(password, keyLen)
	if len(key) != keyLen {
		t.Errorf("expected key length %d, got %d", keyLen, len(key))
	}

	// Known hash for first 20 bytes: sha1("testpassword")
	h := sha1.New()
	h.Write([]byte(password))
	expectedFirst20 := h.Sum(nil)

	if hex.EncodeToString(key[:20]) != hex.EncodeToString(expectedFirst20) {
		t.Errorf("first 20 bytes of key do not match sha1(password)")
	}
}

func TestParseSSUri(t *testing.T) {
	uri := "ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@1.2.3.4:8388"
	cfg, err := parseSSUri(uri)
	if err != nil {
		t.Fatalf("failed to parse uri: %v", err)
	}

	if cfg.Method != "aes-256-gcm" {
		t.Errorf("expected method aes-256-gcm, got %s", cfg.Method)
	}
	if cfg.Password != "password" {
		t.Errorf("expected password password, got %s", cfg.Password)
	}
	if cfg.Server != "1.2.3.4:8388" {
		t.Errorf("expected server 1.2.3.4:8388, got %s", cfg.Server)
	}
}
