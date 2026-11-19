package category

import "testing"

func TestDetect(t *testing.T) {
	tests := map[string]string{
		"chai":              Food,
		"uber to airport":   Travel,
		"Amazon order":      Shopping,
		"wifi, march":       Bills,
		"birthday present":  Other,
		"chainsaw":          Other,
		"dinner after cab.": Food,
	}
	for in, want := range tests {
		if got := Detect(in); got != want {
			t.Errorf("Detect(%q) = %q, want %q", in, got, want)
		}
	}
}
