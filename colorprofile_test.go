package main

import (
	"strings"
	"testing"
)

// withProfile runs fn with activeProfile temporarily set to p, then restores it.
func withProfile(p colorProfile, fn func()) {
	orig := activeProfile
	activeProfile = p
	defer func() { activeProfile = orig }()
	fn()
}

func TestResolveNoColorProfileEmitsNothing(t *testing.T) {
	withProfile(profileNone, func() {
		if got := resolveFg("#7aa2f7"); got != "" {
			t.Errorf("NO_COLOR fg should be empty, got %q", got)
		}
		if got := resolveFg("red"); got != "" {
			t.Errorf("NO_COLOR named fg should be empty, got %q", got)
		}
		if got := resolveBg("#7aa2f7"); got != "" {
			t.Errorf("NO_COLOR bg should be empty, got %q", got)
		}
	})
}

func TestResolveTrueColorEmitsHex(t *testing.T) {
	withProfile(profileTrueColor, func() {
		got := resolveFg("#7aa2f7")
		if !strings.HasPrefix(got, "\x1b[38;2;") {
			t.Errorf("truecolor should emit 24-bit sequence, got %q", got)
		}
	})
}

func TestResolveANSIProfileDegradesHex(t *testing.T) {
	withProfile(profileANSI, func() {
		got := resolveFg("#000000") // pure black → "black" → 30
		if strings.Contains(got, "38;2;") {
			t.Errorf("ANSI profile must NOT emit truecolor, got %q", got)
		}
		if got != "\x1b[30m" {
			t.Errorf("black hex should degrade to ANSI black (30m), got %q", got)
		}
	})
}

func TestNearestANSIName(t *testing.T) {
	cases := []struct{ r, g, b int; want string }{
		{0, 0, 0, "black"},
		{255, 0, 0, "bright_red"},
		{0, 255, 0, "bright_green"},
		{10, 10, 240, "blue"}, // close to pure blue
	}
	for _, c := range cases {
		if got := nearestANSIName(c.r, c.g, c.b); got != c.want {
			t.Errorf("nearestANSIName(%d,%d,%d) = %q, want %q", c.r, c.g, c.b, got, c.want)
		}
	}
}
