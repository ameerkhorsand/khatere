package domain

import "testing"

func TestValid(t *testing.T) {
	cases := map[float64]bool{
		0: true, 0.5: true, 5: true, 4.5: true,
		-0.5: false, 5.5: false, 3.3: false, 2.25: false,
	}
	for score, want := range cases {
		if got := Valid(score); got != want {
			t.Errorf("Valid(%v) = %v, want %v", score, got, want)
		}
	}
}
