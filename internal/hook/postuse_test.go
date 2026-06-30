package hook

import (
	"strings"
	"testing"
)

func TestMaskSecrets_ReplacesPlaintext(t *testing.T) {
	secrets := []secretValue{
		{Name: "DB_PASSWORD", Plaintext: []byte("s3cr3t!")},
	}
	input := `{"output": "Connected with password s3cr3t!"}`
	masked, names := maskSecrets(input, secrets)

	if strings.Contains(masked, "s3cr3t!") {
		t.Error("plaintext still present after masking")
	}
	if !strings.Contains(masked, "<ENVAULT:DB_PASSWORD>") {
		t.Error("placeholder not inserted")
	}
	if len(names) != 1 || names[0] != "DB_PASSWORD" {
		t.Errorf("unexpected replaced names: %v", names)
	}
}

func TestMaskSecrets_ReplacesBase64Variant(t *testing.T) {
	secrets := []secretValue{
		{Name: "API_KEY", Plaintext: []byte("mysecret")},
	}
	// base64("mysecret") = "bXlzZWNyZXQ="
	input := `token: bXlzZWNyZXQ=`
	masked, names := maskSecrets(input, secrets)

	if strings.Contains(masked, "bXlzZWNyZXQ=") {
		t.Error("base64 secret still present after masking")
	}
	if !strings.Contains(masked, "<ENVAULT:API_KEY|base64>") {
		t.Errorf("base64 placeholder not inserted; got: %s", masked)
	}
	if len(names) == 0 {
		t.Error("expected at least one replaced name")
	}
}

func TestMaskSecrets_NoMatchPassesThrough(t *testing.T) {
	secrets := []secretValue{
		{Name: "DB_PASSWORD", Plaintext: []byte("s3cr3t!")},
	}
	input := `{"output": "hello world"}`
	masked, names := maskSecrets(input, secrets)

	if masked != input {
		t.Errorf("unmodified text should pass through unchanged; got %q", masked)
	}
	if len(names) != 0 {
		t.Errorf("expected no replaced names, got %v", names)
	}
}

func TestMaskSecrets_EmptySecrets(t *testing.T) {
	input := `some output`
	masked, names := maskSecrets(input, nil)
	if masked != input || len(names) != 0 {
		t.Error("empty secrets should result in pass-through")
	}
}

func TestMaskSecrets_MultipleSecrets(t *testing.T) {
	secrets := []secretValue{
		{Name: "DB_PASS", Plaintext: []byte("pass1")},
		{Name: "API_KEY", Plaintext: []byte("key2")},
	}
	input := `db=pass1 key=key2`
	masked, names := maskSecrets(input, secrets)

	if strings.Contains(masked, "pass1") || strings.Contains(masked, "key2") {
		t.Error("secrets still present after masking")
	}
	if len(names) != 2 {
		t.Errorf("expected 2 replaced names, got %d: %v", len(names), names)
	}
}

func TestMaskSecrets_EmptyPlaintextSkipped(t *testing.T) {
	secrets := []secretValue{
		{Name: "EMPTY", Plaintext: []byte("")},
	}
	input := `some output`
	masked, _ := maskSecrets(input, secrets)
	if masked != input {
		t.Error("empty secret should not modify output")
	}
}
