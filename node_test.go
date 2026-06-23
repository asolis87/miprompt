package main

import "testing"

func TestVersionsMatch(t *testing.T) {
	cases := []struct {
		requested, active string
		want              bool
	}{
		// Exact and prefix matches.
		{"20.11.0", "20.11.0", true},
		{"20", "20.11.0", true},       // .nvmrc often holds just the major
		{"20.11", "20.11.0", true},    // major.minor
		{"v20.11.0", "20.11.0", true}, // leading v
		// Ranges are flexible by design → never warn, regardless of active.
		{">=18", "18.4.0", true},
		{">=18", "26.3.0", true}, // the case that slipped past the first test
		{"^20.0.0", "20.11.0", true},
		{"^20.0.0", "26.3.0", true}, // even a "wrong" major: caret is a range, don't cry wolf
		{"~16.1.0", "16.1.9", true},
		{">=20", "18.4.0", true}, // we don't evaluate the comparator, just don't warn
		// Real mismatches — PINNED requirement vs different active.
		{"18.0.0", "26.3.0", false},
		{"18", "26.3.0", false},
		{"v18.0.0", "26.3.0", false}, // leading v is still pinned
		// Edge cases: nothing to compare against → never warn.
		{"", "26.3.0", true},
		{"26.3.0", "", true},
		{"lts/hydrogen", "20.11.0", true}, // non-numeric label: cannot compare
	}
	for _, c := range cases {
		if got := versionsMatch(c.requested, c.active); got != c.want {
			t.Errorf("versionsMatch(%q, %q) = %v, want %v", c.requested, c.active, got, c.want)
		}
	}
}
