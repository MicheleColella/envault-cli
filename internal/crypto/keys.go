package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// PrivateKey is a 32-byte X25519 private scalar.
type PrivateKey [32]byte

// PublicKey is a 32-byte X25519 public key.
type PublicKey [32]byte

// GenerateKeyPair generates a random X25519 keypair.
func GenerateKeyPair() (PrivateKey, PublicKey, error) {
	var priv PrivateKey
	if _, err := io.ReadFull(rand.Reader, priv[:]); err != nil {
		return PrivateKey{}, PublicKey{}, err
	}

	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return PrivateKey{}, PublicKey{}, err
	}

	var pubKey PublicKey
	copy(pubKey[:], pub)
	return priv, pubKey, nil
}

// DerivePublicKey computes the X25519 public key corresponding to priv.
func DerivePublicKey(priv PrivateKey) (PublicKey, error) {
	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return PublicKey{}, err
	}
	var pubKey PublicKey
	copy(pubKey[:], pub)
	return pubKey, nil
}

// deriveWrappingKey produces a 32-byte wrapping key via HKDF-SHA256.
// recipientPub is bound into the info field to prevent cross-recipient key
// substitution (the same ephemeral key cannot be repurposed for a different recipient).
func deriveWrappingKey(sharedSecret, ephemeralPub, recipientPub []byte) ([]byte, error) {
	info := make([]byte, 0, len("envault-v1-wrap")+len(recipientPub))
	info = append(info, "envault-v1-wrap"...)
	info = append(info, recipientPub...)

	r := hkdf.New(sha256.New, sharedSecret, ephemeralPub, info)
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}
