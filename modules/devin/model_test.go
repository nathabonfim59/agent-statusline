package devin

import "testing"

func TestHumanTokensCeil(t *testing.T) {
	cases := map[int]string{
		999:    "999",
		1_000:  "1k",
		24_001: "25k",
		25_000: "25k",
	}

	for tokens, want := range cases {
		if got := humanTokensCeil(tokens); got != want {
			t.Fatalf("humanTokensCeil(%d) = %q; want %q", tokens, got, want)
		}
	}
}
