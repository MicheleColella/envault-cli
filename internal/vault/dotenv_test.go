package vault

import (
	"strings"
	"testing"
)

func TestParseDotenv_BasicPairs(t *testing.T) {
	in := "FOO=bar\nBAZ=qux\n"

	got, err := ParseDotenv(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseDotenv: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d vars, want 2", len(got))
	}
	if got[0] != (EnvVar{Key: "FOO", Value: "bar"}) {
		t.Errorf("first var = %+v", got[0])
	}
	if got[1] != (EnvVar{Key: "BAZ", Value: "qux"}) {
		t.Errorf("second var = %+v", got[1])
	}
}

func TestParseDotenv_SkipsBlankAndComments(t *testing.T) {
	in := "\n# a comment\nFOO=bar\n   \n# another\n"

	got, err := ParseDotenv(strings.NewReader(in))
	if err != nil {
		t.Fatalf("ParseDotenv: %v", err)
	}
	if len(got) != 1 || got[0].Key != "FOO" {
		t.Errorf("expected only FOO, got %+v", got)
	}
}

func TestParseDotenv_HonorsExportPrefix(t *testing.T) {
	got, err := ParseDotenv(strings.NewReader("export TOKEN=secret\n"))
	if err != nil {
		t.Fatalf("ParseDotenv: %v", err)
	}
	if len(got) != 1 || got[0].Key != "TOKEN" || got[0].Value != "secret" {
		t.Errorf("export prefix not handled: %+v", got)
	}
}

func TestParseDotenv_StripsMatchingQuotes(t *testing.T) {
	cases := map[string]string{
		`A="double"`:     "double",
		`B='single'`:     "single",
		`C=plain`:        "plain",
		`D="unbalanced'`: `"unbalanced'`,
	}
	for line, want := range cases {
		got, err := ParseDotenv(strings.NewReader(line + "\n"))
		if err != nil {
			t.Fatalf("ParseDotenv(%q): %v", line, err)
		}
		if got[0].Value != want {
			t.Errorf("ParseDotenv(%q) value = %q, want %q", line, got[0].Value, want)
		}
	}
}

func TestParseDotenv_KeepsValueWithEquals(t *testing.T) {
	got, err := ParseDotenv(strings.NewReader("QUERY=a=1&b=2&c=3\n"))
	if err != nil {
		t.Fatalf("ParseDotenv: %v", err)
	}
	if got[0].Value != "a=1&b=2&c=3" {
		t.Errorf("value with '=' mangled: %q", got[0].Value)
	}
}

func TestParseDotenv_ErrorsOnMissingEquals(t *testing.T) {
	_, err := ParseDotenv(strings.NewReader("NOTAVALIDLINE\n"))
	if err == nil {
		t.Fatal("expected error for line without '='")
	}
}

func TestParseDotenv_ErrorsOnEmptyKey(t *testing.T) {
	_, err := ParseDotenv(strings.NewReader("=value\n"))
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}
