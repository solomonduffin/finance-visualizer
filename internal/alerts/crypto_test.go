package alerts

import (
	"testing"
)

func TestDeriveKey(t *testing.T) {
	key := DeriveKey("test-secret")
	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d bytes", len(key))
	}

	// Same input must produce same output (deterministic).
	key2 := DeriveKey("test-secret")
	if string(key) != string(key2) {
		t.Error("DeriveKey is not deterministic: same input produced different outputs")
	}

	// Different input must produce different output.
	key3 := DeriveKey("other-secret")
	if string(key) == string(key3) {
		t.Error("different secrets produced the same key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := DeriveKey("test-secret")
	plaintext := "my-password"

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if ciphertext == "" {
		t.Fatal("Encrypt returned empty ciphertext")
	}

	// Ciphertext should be base64-encoded (no raw bytes).
	if ciphertext == plaintext {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptWrongKey(t *testing.T) {
	key1 := DeriveKey("key-one")
	key2 := DeriveKey("key-two")

	ciphertext, err := Encrypt("secret-data", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("expected error when decrypting with wrong key, got nil")
	}
}

func TestEncryptDifferentNonces(t *testing.T) {
	key := DeriveKey("test-secret")
	plaintext := "same-input"

	ct1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("first Encrypt failed: %v", err)
	}
	ct2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("second Encrypt failed: %v", err)
	}

	if ct1 == ct2 {
		t.Error("two encryptions of the same plaintext should produce different ciphertexts (random nonce)")
	}
}
