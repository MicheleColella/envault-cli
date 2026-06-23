package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// CipherSuite selects the AEAD algorithm used for both payload and key wrapping.
type CipherSuite string

const (
	AES256GCM        CipherSuite = "aes-256-gcm"
	ChaCha20Poly1305 CipherSuite = "chacha20-poly1305"
)

// newAEAD constructs a cipher.AEAD for the given 32-byte key and suite.
func newAEAD(key []byte, suite CipherSuite) (cipher.AEAD, error) {
	switch suite {
	case AES256GCM:
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		return cipher.NewGCM(block)
	case ChaCha20Poly1305:
		return chacha20poly1305.New(key)
	default:
		return nil, fmt.Errorf("unknown cipher suite %q", suite)
	}
}

// validateSuite returns an error if suite is not a recognised CipherSuite.
func validateSuite(suite CipherSuite) error {
	switch suite {
	case AES256GCM, ChaCha20Poly1305:
		return nil
	default:
		return fmt.Errorf("unknown cipher suite %q", suite)
	}
}
