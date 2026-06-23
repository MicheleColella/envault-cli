package crypto

import (
	"testing"
)

// tamperByte returns a copy of b with byte at position i (mod len) XOR'd with 0xff.
// Fatally fails the test if b is empty, which would make the tamper a silent no-op.
func tamperByte(t *testing.T, b []byte, i int) []byte {
	t.Helper()
	if len(b) == 0 {
		t.Fatal("tamperByte: cannot tamper an empty slice")
	}
	out := make([]byte, len(b))
	copy(out, b)
	out[i%len(out)] ^= 0xff
	return out
}

func TestUnseal_TamperedCiphertext(t *testing.T) {
	for _, suite := range []CipherSuite{AES256GCM, ChaCha20Poly1305} {
		t.Run(string(suite), func(t *testing.T) {
			priv, pub, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair: %v", err)
			}
			env, err := Seal([]byte("super secret"), []PublicKey{pub}, suite)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}

			env.Ciphertext = tamperByte(t, env.Ciphertext, len(env.Ciphertext)/2)

			_, err = Unseal(env, priv)
			if err == nil {
				t.Error("Unseal with tampered ciphertext must return error")
			}
		})
	}
}

func TestUnseal_TruncatedCiphertext(t *testing.T) {
	for _, suite := range []CipherSuite{AES256GCM, ChaCha20Poly1305} {
		t.Run(string(suite), func(t *testing.T) {
			priv, pub, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair: %v", err)
			}
			env, err := Seal([]byte("hello world"), []PublicKey{pub}, suite)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}

			env.Ciphertext = env.Ciphertext[:len(env.Ciphertext)/2]

			_, err = Unseal(env, priv)
			if err == nil {
				t.Errorf("Unseal with truncated ciphertext must return error")
			}
		})
	}
}

func TestUnseal_EmptyCiphertext(t *testing.T) {
	for _, suite := range []CipherSuite{AES256GCM, ChaCha20Poly1305} {
		t.Run(string(suite), func(t *testing.T) {
			priv, pub, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair: %v", err)
			}
			env, err := Seal([]byte("hello"), []PublicKey{pub}, suite)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}

			env.Ciphertext = []byte{}

			_, err = Unseal(env, priv)
			if err == nil {
				t.Errorf("Unseal with empty ciphertext must return error")
			}
		})
	}
}

func TestUnseal_TamperedNonce(t *testing.T) {
	for _, suite := range []CipherSuite{AES256GCM, ChaCha20Poly1305} {
		t.Run(string(suite), func(t *testing.T) {
			priv, pub, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair: %v", err)
			}
			env, err := Seal([]byte("secret"), []PublicKey{pub}, suite)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}

			env.Nonce = tamperByte(t, env.Nonce, 0)

			_, err = Unseal(env, priv)
			if err == nil {
				t.Error("Unseal with tampered nonce must return error")
			}
		})
	}
}

func TestUnseal_TamperedWrappedKey(t *testing.T) {
	for _, suite := range []CipherSuite{AES256GCM, ChaCha20Poly1305} {
		t.Run(string(suite), func(t *testing.T) {
			priv, pub, err := GenerateKeyPair()
			if err != nil {
				t.Fatalf("GenerateKeyPair: %v", err)
			}
			env, err := Seal([]byte("secret"), []PublicKey{pub}, suite)
			if err != nil {
				t.Fatalf("Seal: %v", err)
			}

			env.Recipients[0].WrappedKey = tamperByte(t, env.Recipients[0].WrappedKey, 0)

			_, err = Unseal(env, priv)
			if err == nil {
				t.Error("Unseal with tampered WrappedKey must return error")
			}
		})
	}
}

func TestUnseal_TamperedEphemeralPublic(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	env.Recipients[0].EphemeralPublic = tamperByte(t, env.Recipients[0].EphemeralPublic, 0)

	_, err = Unseal(env, priv)
	if err == nil {
		t.Error("Unseal with tampered EphemeralPublic must return error")
	}
}

func TestUnseal_TamperedWrapNonce(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	env.Recipients[0].Nonce = tamperByte(t, env.Recipients[0].Nonce, 0)

	_, err = Unseal(env, priv)
	if err == nil {
		t.Error("Unseal with tampered recipient nonce must return error")
	}
}

func TestUnseal_TamperedSuite(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Switching Suite causes the AD string and the AEAD algorithm to both mismatch the ciphertext.
	env.Suite = ChaCha20Poly1305

	_, err = Unseal(env, priv)
	if err == nil {
		t.Error("Unseal with mismatched Suite must return error")
	}
}

func TestUnseal_NoRecipients(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	env.Recipients = nil

	_, err = Unseal(env, priv)
	if err == nil {
		t.Error("Unseal with empty recipient list must return error")
	}
}
