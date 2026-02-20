package services

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := "0123456789abcdef0123456789abcdef" // 32 bytes
	svc, err := NewEncryptionService(key)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	plaintext := []byte("hello world in Germany")
	ciphertext, err := svc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Error("ciphertext should not be same as plaintext")
	}

	decrypted, err := svc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("decrypted data %s does not match original %s", decrypted, plaintext)
	}
}

func TestWrongKeyLength(t *testing.T) {
	key := "short-key"
	_, err := NewEncryptionService(key)
	if err == nil {
		t.Error("Should have failed with short key")
	}
}
