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

// ErrNoEncryptionKey is returned when a secret must be read or written but
// Open got no key.
var ErrNoEncryptionKey = errors.New("storage: an encryption key is required to read/write stored credentials (AI configs and client tokens)")

// encryptor seals and opens secrets with AES-256-GCM. The stored format is
// base64(nonce || ciphertext).
type encryptor struct {
	gcm cipher.AEAD
}

func newEncryptor(base64Key string) (*encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("storage: encryption key is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("storage: encryption key must decode to 32 bytes, got %d", len(key))
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
		return "", ErrNoEncryptionKey
	}
	return r.enc.encrypt(plaintext)
}

// decryptField decrypts a possibly-empty stored secret. Empty stays empty.
func (r *SQLiteRepository) decryptField(b64 string) (string, error) {
	if b64 == "" {
		return "", nil
	}
	if r.enc == nil {
		return "", ErrNoEncryptionKey
	}
	return r.enc.decrypt(b64)
}

func (r *SQLiteRepository) encryptEach(fields ...*string) error {
	for _, f := range fields {
		v, err := r.encryptField(*f)
		if err != nil {
			return err
		}
		*f = v
	}
	return nil
}

func (r *SQLiteRepository) decryptEach(fields ...*string) error {
	for _, f := range fields {
		v, err := r.decryptField(*f)
		if err != nil {
			return err
		}
		*f = v
	}
	return nil
}
