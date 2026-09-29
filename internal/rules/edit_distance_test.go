package rules

import "testing"

func TestEditDistance(t *testing.T) {
	cases := []struct {
		a, b string
		cap  int
		want int
	}{
		{"requests", "requests", 2, 0},
		{"requests", "reqeusts", 2, 1}, // adjacent transposition, the brief's own demo typo
		{"requests", "request", 2, 1},  // one deletion
		{"colour", "color", 2, 1},
		{"lodash", "lodahs", 2, 1}, // transposition of the last two letters
		{"", "", 2, 0},
		{"a", "", 2, 1},
		{"kitten", "sitting", 1, 2}, // true distance is 3, capped result is cap+1
	}
	for _, c := range cases {
		got := EditDistance(c.a, c.b, c.cap)
		if got != c.want {
			t.Errorf("EditDistance(%q, %q, %d) = %d, want %d", c.a, c.b, c.cap, got, c.want)
		}
	}
}

func TestEditDistanceSymmetric(t *testing.T) {
	pairs := [][2]string{{"express", "expres"}, {"react", "reactt"}, {"numpy", "nunpy"}, {"django", "djagno"}}
	for _, p := range pairs {
		d1 := EditDistance(p[0], p[1], 3)
		d2 := EditDistance(p[1], p[0], 3)
		if d1 != d2 {
			t.Errorf("EditDistance not symmetric for %v: %d vs %d", p, d1, d2)
		}
	}
}
