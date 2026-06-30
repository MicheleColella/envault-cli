package protect

import "testing"

// FuzzContainsProtectedPath verifies the Bash-command tokenizer never panics
// on adversarial input and respects the no-match guarantee for empty patterns.
func FuzzContainsProtectedPath(f *testing.F) {
	// Seed corpus: representative cases from the red-team matrix.
	seeds := [][2]string{
		{"cat .env", ".env"},
		{`python3 -c "open('.env').read()"`, ".env"},
		{"git show HEAD:.env", ".env"},
		{"FILE=.env; cat $FILE", ".env"},
		{"$(cat .env)", ".env"},
		{"", ""},
		{"normal command with no paths", "secret.txt"},
		{"cat /etc/passwd", "*.conf"},
		{"\x00\xff\xfe", ".env"},
		{"cat " + string([]byte{0xc0, 0xaf}) + ".env", ".env"}, // invalid UTF-8
		{"cat .env" + string(make([]byte, 8192)), ".env"},      // very long command
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}

	f.Fuzz(func(t *testing.T, text, pattern string) {
		// Must never panic.
		ContainsProtectedPath(text, []string{pattern})

		// An empty pattern must never match a non-empty text (filepath.Match("", x) is false).
		if pattern == "" && text != "" {
			if _, _, found := ContainsProtectedPath(text, []string{""}); found {
				t.Errorf("empty pattern matched non-empty text %q", text)
			}
		}
	})
}

// FuzzMatchesAny verifies MatchesAny never panics on arbitrary input.
func FuzzMatchesAny(f *testing.F) {
	f.Add(".env", ".env")
	f.Add("config/secrets.json", "config/")
	f.Add("", "")
	f.Add("../../../etc/passwd", "*.passwd")
	f.Add(string(make([]byte, 4096)), "a")

	f.Fuzz(func(t *testing.T, path, pattern string) {
		MatchesAny(path, []string{pattern})
	})
}
