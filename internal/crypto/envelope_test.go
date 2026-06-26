package crypto

import (
	"bytes"
	"testing"
)

func TestSealUnseal_SingleRecipient_AES256GCM(t *testing.T) {
	testSealUnsealSingle(t, AES256GCM)
}

func TestSealUnseal_SingleRecipient_ChaCha20Poly1305(t *testing.T) {
	testSealUnsealSingle(t, ChaCha20Poly1305)
}

func testSealUnsealSingle(t *testing.T, suite CipherSuite) {
	t.Helper()

	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	payload := []byte("super secret value")

	env, err := Seal(payload, []PublicKey{pub}, suite)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if env.Version != envelopeVersion {
		t.Errorf("Version = %d, want %d", env.Version, envelopeVersion)
	}
	if env.Suite != suite {
		t.Errorf("Suite = %q, want %q", env.Suite, suite)
	}
	if len(env.Recipients) != 1 {
		t.Errorf("Recipients count = %d, want 1", len(env.Recipients))
	}

	got, err := Unseal(env, priv)
	if err != nil {
		t.Fatalf("Unseal: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("Unseal = %q, want %q", got, payload)
	}
}

func TestSealUnseal_MultipleRecipients(t *testing.T) {
	priv1, pub1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair 1: %v", err)
	}
	priv2, pub2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair 2: %v", err)
	}
	priv3, pub3, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair 3: %v", err)
	}

	payload := []byte("shared team secret")

	env, err := Seal(payload, []PublicKey{pub1, pub2, pub3}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if len(env.Recipients) != 3 {
		t.Errorf("Recipients = %d, want 3", len(env.Recipients))
	}

	for i, priv := range []PrivateKey{priv1, priv2, priv3} {
		got, err := Unseal(env, priv)
		if err != nil {
			t.Errorf("Unseal key%d: %v", i+1, err)
			continue
		}
		if !bytes.Equal(got, payload) {
			t.Errorf("Unseal key%d = %q, want %q", i+1, got, payload)
		}
	}
}

func TestUnseal_WrongKey(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	wrongPriv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair wrong: %v", err)
	}

	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	_, err = Unseal(env, wrongPriv)
	if err == nil {
		t.Fatal("Unseal with wrong key should return error")
	}
}

func TestSeal_CiphertextRandomness(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	payload := []byte("same payload sealed twice")

	env1, err := Seal(payload, []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal 1: %v", err)
	}
	env2, err := Seal(payload, []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal 2: %v", err)
	}

	if bytes.Equal(env1.Nonce, env2.Nonce) {
		t.Error("Seal produced identical nonces for two seals")
	}
	if bytes.Equal(env1.Ciphertext, env2.Ciphertext) {
		t.Error("Seal produced identical ciphertexts for the same payload")
	}
}

func TestUnseal_BadVersion(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	env.Version = 999
	_, err = Unseal(env, priv)
	if err == nil {
		t.Fatal("Unseal with bad version should return error")
	}
}

func TestSealUnseal_EmptyPayload(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	env, err := Seal([]byte{}, []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal empty payload: %v", err)
	}

	got, err := Unseal(env, priv)
	if err != nil {
		t.Fatalf("Unseal empty payload: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Unseal empty = %q, want empty", got)
	}
}

func TestSealUnseal_LargePayload(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	payload := make([]byte, 1<<20) // 1 MB
	for i := range payload {
		payload[i] = byte(i % 251)
	}

	env, err := Seal(payload, []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal large: %v", err)
	}

	got, err := Unseal(env, priv)
	if err != nil {
		t.Fatalf("Unseal large: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("large payload round-trip failed")
	}
}

func TestSeal_NoRecipients(t *testing.T) {
	_, err := Seal([]byte("secret"), []PublicKey{}, AES256GCM)
	if err == nil {
		t.Fatal("Seal with no recipients should return error")
	}
}

func TestRewrap_ChangeRecipient(t *testing.T) {
	// Seal for alice, then rewrap for bob — bob should be able to decrypt.
	alice, alicePub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair alice: %v", err)
	}
	_, bobPub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair bob: %v", err)
	}
	bob2, bob2Pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair bob2: %v", err)
	}

	payload := []byte("team secret")

	env, err := Seal(payload, []PublicKey{alicePub, bobPub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	// Replace bob with bob2 — alice rewraps.
	rewrapped, err := Rewrap(env, alice, []PublicKey{alicePub, bob2Pub})
	if err != nil {
		t.Fatalf("Rewrap: %v", err)
	}

	// Ciphertext must be identical (same payload, same nonce).
	if !bytes.Equal(rewrapped.Ciphertext, env.Ciphertext) {
		t.Error("Rewrap changed ciphertext — it should not re-encrypt the payload")
	}
	if !bytes.Equal(rewrapped.Nonce, env.Nonce) {
		t.Error("Rewrap changed nonce — it should preserve the payload nonce")
	}
	if len(rewrapped.Recipients) != 2 {
		t.Errorf("Recipients = %d, want 2", len(rewrapped.Recipients))
	}

	// alice can still decrypt.
	got, err := Unseal(rewrapped, alice)
	if err != nil {
		t.Fatalf("Unseal alice after Rewrap: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("alice got %q, want %q", got, payload)
	}

	// bob2 can decrypt.
	got, err = Unseal(rewrapped, bob2)
	if err != nil {
		t.Fatalf("Unseal bob2 after Rewrap: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("bob2 got %q, want %q", got, payload)
	}
}

func TestRewrap_WrongKey(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	wrong, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair wrong: %v", err)
	}

	_, err = Rewrap(env, wrong, []PublicKey{pub})
	if err == nil {
		t.Fatal("Rewrap with wrong key should return error")
	}
}

func TestRewrap_NoRecipients(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	env, err := Seal([]byte("secret"), []PublicKey{pub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	_, err = Rewrap(env, priv, []PublicKey{})
	if err == nil {
		t.Fatal("Rewrap with no recipients should return error")
	}
}

func TestRewrap_AddRecipient(t *testing.T) {
	// Seal for alice only, then rewrap to add bob.
	alice, alicePub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair alice: %v", err)
	}
	bob, bobPub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair bob: %v", err)
	}

	payload := []byte("secret for more people")
	env, err := Seal(payload, []PublicKey{alicePub}, AES256GCM)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	rewrapped, err := Rewrap(env, alice, []PublicKey{alicePub, bobPub})
	if err != nil {
		t.Fatalf("Rewrap: %v", err)
	}
	if len(rewrapped.Recipients) != 2 {
		t.Errorf("Recipients = %d, want 2", len(rewrapped.Recipients))
	}

	got, err := Unseal(rewrapped, bob)
	if err != nil {
		t.Fatalf("Unseal bob: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("bob got %q, want %q", got, payload)
	}
}

func TestSeal_NonceUniqueness(t *testing.T) {
	_, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	const iterations = 200
	payloadNonces := make(map[string]struct{}, iterations)
	wrapNonces := make(map[string]struct{}, iterations)
	payload := []byte("repeated payload")

	for i := range iterations {
		env, err := Seal(payload, []PublicKey{pub}, AES256GCM)
		if err != nil {
			t.Fatalf("Seal %d: %v", i, err)
		}

		pKey := string(env.Nonce)
		if _, dup := payloadNonces[pKey]; dup {
			t.Fatalf("duplicate payload nonce at iteration %d: %x", i, env.Nonce)
		}
		payloadNonces[pKey] = struct{}{}

		wKey := string(env.Recipients[0].Nonce)
		if _, dup := wrapNonces[wKey]; dup {
			t.Fatalf("duplicate wrap nonce at iteration %d: %x", i, env.Recipients[0].Nonce)
		}
		wrapNonces[wKey] = struct{}{}
	}
}
