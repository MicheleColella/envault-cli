package ui

import (
	"bytes"
	"strings"
	"testing"
)

func captureOut(fn func()) string {
	orig := Out
	var buf bytes.Buffer
	Out = &buf
	fn()
	Out = orig
	return buf.String()
}

func captureErr(fn func()) string {
	orig := Err
	var buf bytes.Buffer
	Err = &buf
	fn()
	Err = orig
	return buf.String()
}

func TestOK(t *testing.T) {
	got := captureOut(func() { OK("all good") })
	if !strings.Contains(got, "✓") || !strings.Contains(got, "all good") {
		t.Errorf("OK() = %q, want ✓ and message", got)
	}
}

func TestFail(t *testing.T) {
	got := captureErr(func() { Fail("broken") })
	if !strings.Contains(got, "✗") || !strings.Contains(got, "broken") {
		t.Errorf("Fail() = %q, want ✗ and message", got)
	}
}

func TestInfo(t *testing.T) {
	got := captureOut(func() { Info("hint") })
	if !strings.Contains(got, "→") || !strings.Contains(got, "hint") {
		t.Errorf("Info() = %q, want → and message", got)
	}
}

func TestWarn(t *testing.T) {
	got := captureOut(func() { Warn("careful") })
	if !strings.Contains(got, "!") || !strings.Contains(got, "careful") {
		t.Errorf("Warn() = %q, want ! and message", got)
	}
}

func TestNotImplemented(t *testing.T) {
	got := captureErr(func() { NotImplemented("init") })
	if !strings.Contains(got, "init") || !strings.Contains(got, "not implemented yet") {
		t.Errorf("NotImplemented() = %q", got)
	}
}
