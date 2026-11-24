package export

import "testing"

func TestSafeCell(t *testing.T) {
	tests := map[string]string{
		"":                  "",
		"chai":              "chai",
		"=HYPERLINK(\"x\")": "'=HYPERLINK(\"x\")",
		"+1":                "'+1",
		"-1":                "'-1",
		"@cmd":              "'@cmd",
	}
	for in, want := range tests {
		if got := safeCell(in); got != want {
			t.Errorf("safeCell(%q) = %q, want %q", in, got, want)
		}
	}
}
