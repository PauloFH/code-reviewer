package storage

import (
	"encoding/base64"
	"testing"
)

func testKey() string {
	return base64.StdEncoding.EncodeToString(make([]byte, 32))
}

func TestEncryptorRoundTrip(t *testing.T) {
	enc, err := newEncryptor(testKey())
	if err != nil {
		t.Fatalf("newEncryptor: %v", err)
	}

	const secret = "sk-super-secret-token"
	ciphertext, err := enc.encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == secret || ciphertext == "" {
		t.Fatalf("ciphertext should differ from plaintext and be non-empty, got %q", ciphertext)
	}

	plaintext, err := enc.decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != secret {
		t.Fatalf("round-trip mismatch: got %q want %q", plaintext, secret)
	}
}

func TestEncryptorNonceIsRandom(t *testing.T) {
	enc, _ := newEncryptor(testKey())
	a, _ := enc.encrypt("same")
	b, _ := enc.encrypt("same")
	if a == b {
		t.Fatal("two encryptions of the same plaintext must differ (random nonce)")
	}
}

func TestEncryptorDecryptGarbage(t *testing.T) {
	enc, _ := newEncryptor(testKey())
	if _, err := enc.decrypt("not-base64-$$$"); err == nil {
		t.Fatal("expected error decrypting invalid base64")
	}
	if _, err := enc.decrypt(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("expected error decrypting too-short ciphertext")
	}
}

func TestNewEncryptorInvalidKey(t *testing.T) {
	if _, err := newEncryptor("%%%not-base64"); err == nil {
		t.Fatal("expected error for non-base64 key")
	}
	if _, err := newEncryptor(base64.StdEncoding.EncodeToString(make([]byte, 16))); err == nil {
		t.Fatal("expected error for 16-byte key (must be 32)")
	}
}

func TestRepoFieldHelpersRequireKey(t *testing.T) {
	r := &SQLiteRepository{} // no encryptor configured
	if _, err := r.encryptField("secret"); err != errNoEncryptionKey {
		t.Fatalf("encryptField without key: got %v want errNoEncryptionKey", err)
	}
	if _, err := r.decryptField("secret"); err != errNoEncryptionKey {
		t.Fatalf("decryptField without key: got %v want errNoEncryptionKey", err)
	}
	// Empty values never need a key.
	if v, err := r.encryptField(""); err != nil || v != "" {
		t.Fatalf("encryptField(\"\") = %q, %v; want \"\", nil", v, err)
	}
	if v, err := r.decryptField(""); err != nil || v != "" {
		t.Fatalf("decryptField(\"\") = %q, %v; want \"\", nil", v, err)
	}
}
