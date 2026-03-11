package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := make([]byte, 32) // AES-256 requires 32-byte key
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("my-secret-oauth-token")

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Ciphertext should be different from plaintext
	if bytes.Equal(plaintext, ciphertext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	ciphertext, err := Encrypt([]byte{}, key)
	if err != nil {
		t.Fatalf("Encrypt of empty plaintext failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt of empty plaintext failed: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("expected empty decrypted, got %v", decrypted)
	}
}

func TestEncrypt_UniqueNonce(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("same-input-each-time")

	c1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatal(err)
	}

	// Same plaintext + same key should produce different ciphertext due to random nonce
	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of the same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	if _, err := rand.Read(key1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(key2); err != nil {
		t.Fatal(err)
	}

	ciphertext, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("Decrypt with wrong key should return error")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	ciphertext, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	// Flip a byte in the ciphertext (after the nonce)
	if len(ciphertext) > 13 {
		ciphertext[13] ^= 0xFF
	}

	_, err = Decrypt(ciphertext, key)
	if err == nil {
		t.Error("Decrypt of tampered ciphertext should return error")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	// AES-GCM nonce is 12 bytes, so anything shorter should fail
	_, err := Decrypt([]byte("short"), key)
	if err == nil {
		t.Error("Decrypt of too-short ciphertext should return error")
	}
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	// AES only accepts 16, 24, or 32 byte keys
	_, err := Encrypt([]byte("test"), []byte("too-short"))
	if err == nil {
		t.Error("Encrypt with invalid key length should return error")
	}
}
