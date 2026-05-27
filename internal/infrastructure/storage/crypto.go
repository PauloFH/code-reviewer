package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// errNoEncryptionKey is returned when an AI config holds a secret but no
// AI_CONFIG_ENC_KEY was configured at boot.
var errNoEncryptionKey = errors.New("storage: AI_CONFIG_ENC_KEY is required to read/write AI credentials")

// encryptor seals and opens secrets with AES-256-GCM. The stored format is
// base64(nonce || ciphertext).
type encryptor struct {
	gcm cipher.AEAD
}

// newEncryptor builds an encryptor from a base64-encoded 32-byte key.
func newEncryptor(base64Key string) (*encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("storage: AI_CONFIG_ENC_KEY is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("storage: AI_CONFIG_ENC_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("storage: building AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("storage: building GCM: %w", err)
	}
	return &encryptor{gcm: gcm}, nil
}

func (e *encryptor) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("storage: generating nonce: %w", err)
	}
	sealed := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (e *encryptor) decrypt(b64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("storage: stored secret is not valid base64: %w", err)
	}
	ns := e.gcm.NonceSize()
	if len(data) < ns {
		return "", errors.New("storage: stored secret is too short")
	}
	nonce, ciphertext := data[:ns], data[ns:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("storage: decrypting secret: %w", err)
	}
	return string(plaintext), nil
}

// encryptField encrypts a possibly-empty secret. Empty stays empty (no key needed).
func (r *SQLiteRepository) encryptField(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if r.enc == nil {
		return "", errNoEncryptionKey
	}
	return r.enc.encrypt(plaintext)
}

// decryptField decrypts a possibly-empty stored secret. Empty stays empty.
func (r *SQLiteRepository) decryptField(b64 string) (string, error) {
	if b64 == "" {
		return "", nil
	}
	if r.enc == nil {
		return "", errNoEncryptionKey
	}
	return r.enc.decrypt(b64)
}
