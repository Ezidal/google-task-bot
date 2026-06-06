package telegram

import (
	"testing"
	"time"
)

func TestParseDueInput(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Moscow")

	cases := []struct {
		in   string
		ok   bool
		day  int
	}{
		{"15.06", true, 15},
		{"15.06.2026", true, 15},
		{"+3d", true, 0},
		{"+1w", true, 0},
		{"bad", false, 0},
	}
	for _, tc := range cases {
		got, err := ParseDueInput(tc.in, loc)
		if tc.ok && err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if tc.ok && tc.day > 0 && got.Day() != tc.day {
			t.Fatalf("%q: day=%d", tc.in, got.Day())
		}
	}
}
