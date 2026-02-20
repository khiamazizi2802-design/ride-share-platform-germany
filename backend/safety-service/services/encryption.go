package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// EncryptionService provides AES-256-GCM encryption and decryption.
//
// Why AES-256-GCM?
//   - AES-256: 256-bit key length meets BSI (German Federal Office for Information
//     Security) recommendations for strong symmetric encryption.
//   - GCM (Galois/Counter Mode): Provides authenticated encryption (AEAD), meaning
//     it guarantees both confidentiality AND integrity/authenticity of the data.
//     Any tampering with the ciphertext will be detected on decryption.
//
// GDPR compliance note:
//   - Encrypting personal documents at rest satisfies Art. 32 GDPR
//     (security of processing).
//   - The encryption key must be stored separately from the data
//     (e.g., in a KMS like AWS KMS or HashiCorp Vault).
type EncryptionService struct {
	gcm cipher.AEAD
}

// NewEncryptionService creates a new EncryptionService.
// key must be exactly 32 bytes for AES-256.
func NewEncryptionService(key string) (*EncryptionService, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be exactly 32 bytes for AES-256, got %d bytes", len(key))
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM wrapper: %w", err)
	}

	return &EncryptionService{gcm: gcm}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
//
// Output format: [ nonce (12 bytes) | ciphertext+tag ]
//
// The nonce is randomly generated per encryption call, ensuring that
// encrypting the same plaintext twice produces different ciphertexts
// (semantic security / IND-CPA).
//
// The GCM tag (16 bytes) is appended by cipher.AEAD.Seal automatically
// and verified automatically by Open.
func (e *EncryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext must not be empty")
	}

	// Generate a cryptographically random nonce.
	nonce := make([]byte, e.gcm.NonceSize()) // 12 bytes for GCM
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the encrypted ciphertext and GCM authentication tag to nonce.
	// dst is nonce (pre-allocated), so the final layout is: nonce || ciphertext+tag
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt decrypts a ciphertext produced by Encrypt.
//
// Input format: [ nonce (12 bytes) | ciphertext+tag ]
//
// Returns an error if the ciphertext has been tampered with (GCM authentication
// failure), is too short, or any other decryption failure occurs.
func (e *EncryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()

	// Minimum valid ciphertext: nonce + GCM overhead (tag = 16 bytes)
	minSize := nonceSize + e.gcm.Overhead()
	if len(ciphertext) < minSize {
		return nil, fmt.Errorf("ciphertext too short: need at least %d bytes, got %d", minSize, len(ciphertext))
	}

	// Split the nonce and the actual encrypted payload.
	nonce, encryptedPayload := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Open decrypts and authenticates the payload.
	// If the tag does not match, Open returns an error — this is the tamper-detection mechanism.
	plaintext, err := e.gcm.Open(nil, nonce, encryptedPayload, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (possible data tampering or wrong key): %w", err)
	}

	return plaintext, nil
}
