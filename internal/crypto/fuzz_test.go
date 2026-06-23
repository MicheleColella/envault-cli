package crypto

import (
	"testing"
)

// FuzzUnseal verifies that Unseal never panics regardless of how the Envelope
// fields are mutated or truncated. numRec (mod 4) controls how many recipient
// blocks are constructed, covering the zero, one, and multi-recipient paths.
func FuzzUnseal(f *testing.F) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		f.Fatalf("GenerateKeyPair: %v", err)
	}

	// Seed: valid AES-256-GCM envelope, one recipient.
	env, err := Seal([]byte("fuzz seed"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		f.Fatalf("Seal: %v", err)
	}
	r := env.Recipients[0]
	f.Add(env.Version, string(env.Suite), env.Nonce, env.Ciphertext,
		r.EphemeralPublic, r.Nonce, r.WrappedKey, uint8(1))

	// Seed: valid ChaCha20-Poly1305 envelope, one recipient.
	envCC, err := Seal([]byte("fuzz seed cc"), []PublicKey{pub}, ChaCha20Poly1305)
	if err != nil {
		f.Fatalf("Seal ChaCha20: %v", err)
	}
	rCC := envCC.Recipients[0]
	f.Add(envCC.Version, string(envCC.Suite), envCC.Nonce, envCC.Ciphertext,
		rCC.EphemeralPublic, rCC.Nonce, rCC.WrappedKey, uint8(1))

	// Seed: zero recipients (exercises nil-recipient early-exit path).
	f.Add(1, "aes-256-gcm", env.Nonce, env.Ciphertext,
		r.EphemeralPublic, r.Nonce, r.WrappedKey, uint8(0))

	// Seed: minimal / empty fields.
	f.Add(0, "", []byte{}, []byte{}, []byte{}, []byte{}, []byte{}, uint8(1))
	f.Add(999, "unknown-suite", make([]byte, 12), make([]byte, 64),
		make([]byte, 32), make([]byte, 12), make([]byte, 48), uint8(3))

	f.Fuzz(func(t *testing.T,
		version int, suiteStr string,
		nonce, ciphertext, ephPub, wrapNonce, wrappedKey []byte,
		numRec uint8,
	) {
		count := int(numRec % 4)
		recipients := make([]Recipient, count)
		for i := range recipients {
			recipients[i] = Recipient{
				EphemeralPublic: ephPub,
				Nonce:           wrapNonce,
				WrappedKey:      wrappedKey,
			}
		}
		candidate := &Envelope{
			Version:    version,
			Suite:      CipherSuite(suiteStr),
			Nonce:      nonce,
			Ciphertext: ciphertext,
			Recipients: recipients,
		}
		// Must never panic — errors are expected for malformed input.
		_, _ = Unseal(candidate, priv)
	})
}

// FuzzTryUnwrap verifies that tryUnwrap never panics on arbitrary recipient blocks.
func FuzzTryUnwrap(f *testing.F) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		f.Fatalf("GenerateKeyPair: %v", err)
	}

	env, err := Seal([]byte("fuzz"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		f.Fatalf("Seal: %v", err)
	}
	r := env.Recipients[0]
	f.Add(r.EphemeralPublic, r.Nonce, r.WrappedKey, string(env.Suite))
	f.Add([]byte{}, []byte{}, []byte{}, "aes-256-gcm")
	// 48 = 32-byte DEK + 16-byte GCM tag (the exact output size of wrapDEK).
	f.Add(make([]byte, 32), make([]byte, 12), make([]byte, 48), "chacha20-poly1305")

	f.Fuzz(func(t *testing.T, ephPub, nonce, wrappedKey []byte, suiteStr string) {
		recipient := Recipient{
			EphemeralPublic: ephPub,
			Nonce:           nonce,
			WrappedKey:      wrappedKey,
		}
		_, _ = tryUnwrap(recipient, priv, CipherSuite(suiteStr))
	})
}
