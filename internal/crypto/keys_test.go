package crypto

import (
	"bytes"
	"testing"

	"golang.org/x/crypto/curve25519"
)

func TestGenerateKeyPair_UniqueKeys(t *testing.T) {
	priv1, pub1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	priv2, pub2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	if priv1 == priv2 {
		t.Error("GenerateKeyPair produced identical private keys")
	}
	if pub1 == pub2 {
		t.Error("GenerateKeyPair produced identical public keys")
	}
}

func TestGenerateKeyPair_PublicKeyDerivedFromPrivate(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	expected, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		t.Fatalf("curve25519.X25519: %v", err)
	}
	if !bytes.Equal(pub[:], expected) {
		t.Error("PublicKey does not match expected derivation from PrivateKey")
	}
}

func TestDeriveWrappingKey_Deterministic(t *testing.T) {
	shared := make([]byte, 32)
	ephPub := make([]byte, 32)
	recipPub := make([]byte, 32)
	shared[0] = 42

	k1, err := deriveWrappingKey(shared, ephPub, recipPub)
	if err != nil {
		t.Fatalf("deriveWrappingKey: %v", err)
	}
	k2, err := deriveWrappingKey(shared, ephPub, recipPub)
	if err != nil {
		t.Fatalf("deriveWrappingKey: %v", err)
	}

	if !bytes.Equal(k1, k2) {
		t.Error("deriveWrappingKey is not deterministic")
	}
	if len(k1) != 32 {
		t.Errorf("deriveWrappingKey returned %d bytes, want 32", len(k1))
	}
}

func TestDeriveWrappingKey_DifferentSharedSecrets(t *testing.T) {
	ephPub := make([]byte, 32)
	recipPub := make([]byte, 32)
	shared1 := make([]byte, 32)
	shared2 := make([]byte, 32)
	shared2[0] = 1

	k1, _ := deriveWrappingKey(shared1, ephPub, recipPub)
	k2, _ := deriveWrappingKey(shared2, ephPub, recipPub)

	if bytes.Equal(k1, k2) {
		t.Error("deriveWrappingKey produced same key for different shared secrets")
	}
}

func TestDeriveWrappingKey_DifferentEphemeralPubs(t *testing.T) {
	shared := make([]byte, 32)
	shared[0] = 7
	recipPub := make([]byte, 32)
	eph1 := make([]byte, 32)
	eph2 := make([]byte, 32)
	eph2[0] = 1

	k1, _ := deriveWrappingKey(shared, eph1, recipPub)
	k2, _ := deriveWrappingKey(shared, eph2, recipPub)

	if bytes.Equal(k1, k2) {
		t.Error("deriveWrappingKey produced same key for different ephemeral public keys")
	}
}

func TestDeriveWrappingKey_DifferentRecipientPubs(t *testing.T) {
	shared := make([]byte, 32)
	shared[0] = 3
	ephPub := make([]byte, 32)
	recip1 := make([]byte, 32)
	recip2 := make([]byte, 32)
	recip2[0] = 1

	k1, _ := deriveWrappingKey(shared, ephPub, recip1)
	k2, _ := deriveWrappingKey(shared, ephPub, recip2)

	if bytes.Equal(k1, k2) {
		t.Error("deriveWrappingKey produced same key for different recipient public keys")
	}
}

func TestX25519_SharedSecretSymmetry(t *testing.T) {
	privA, pubA, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair A: %v", err)
	}
	privB, pubB, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair B: %v", err)
	}

	sharedAB, err := curve25519.X25519(privA[:], pubB[:])
	if err != nil {
		t.Fatalf("X25519(privA, pubB): %v", err)
	}
	sharedBA, err := curve25519.X25519(privB[:], pubA[:])
	if err != nil {
		t.Fatalf("X25519(privB, pubA): %v", err)
	}

	if !bytes.Equal(sharedAB, sharedBA) {
		t.Error("X25519 shared secret is not symmetric")
	}
}
