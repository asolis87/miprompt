package main

import "testing"

func TestResolveFg(t *testing.T) {
	cases := []struct{ in, want string }{
		// ANSI base names → 30–37.
		{"red", "\x1b[31m"},
		{"blue", "\x1b[34m"},
		// Bright variants → 90–97.
		{"bright_green", "\x1b[92m"},
		{"bright_black", "\x1b[90m"},
		// Hex full and shorthand → truecolor 38;2.
		{"#7aa2f7", "\x1b[38;2;122;162;247m"},
		{"#fff", "\x1b[38;2;255;255;255m"},
		// Unknown / empty → no color.
		{"", ""},
		{"chartreuse", ""},
		{"#zzz", ""},
	}
	for _, c := range cases {
		if got := resolveFg(c.in); got != c.want {
			t.Errorf("resolveFg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveBg(t *testing.T) {
	cases := []struct{ in, want string }{
		{"red", "\x1b[41m"},          // base bg → 40–47
		{"bright_blue", "\x1b[104m"}, // bright bg → 100–107
		{"#7aa2f7", "\x1b[48;2;122;162;247m"},
		{"", ""},
	}
	for _, c := range cases {
		if got := resolveBg(c.in); got != c.want {
			t.Errorf("resolveBg(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
