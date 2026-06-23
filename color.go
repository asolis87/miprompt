package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Color values in config may be either:
//   - an ANSI color NAME ("blue", "bright_green") — resolved by the terminal's
//     theme, so the prompt follows theme changes automatically; or
//   - a HEX value ("#7aa2f7") — an exact 24-bit (truecolor) RGB color, fixed
//     regardless of the terminal theme.
//
// resolveFg / resolveBg turn either form into the right ANSI escape for the
// foreground or background, honoring the terminal's color profile (see
// colorprofile.go): no color when NO_COLOR is set, and hex degraded to the
// nearest named color on terminals without truecolor. Empty / unrecognized
// input yields "" (no color).

// resolveFg returns the foreground ANSI sequence for a color value.
func resolveFg(value string) string {
	if activeProfile == profileNone {
		return ""
	}
	if r, g, b, ok := parseHex(value); ok {
		if activeProfile == profileTrueColor {
			return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
		}
		// No truecolor: degrade the hex to the nearest of the 16 ANSI colors.
		return ansiNameFg(nearestANSIName(r, g, b))
	}
	return ansiNameFg(value)
}

// resolveBg returns the background ANSI sequence for a color value.
func resolveBg(value string) string {
	if activeProfile == profileNone {
		return ""
	}
	if r, g, b, ok := parseHex(value); ok {
		if activeProfile == profileTrueColor {
			return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
		}
		return ansiNameBg(nearestANSIName(r, g, b))
	}
	return ansiNameBg(value)
}

// parseHex parses "#rrggbb" (or "#rgb" shorthand) into RGB components. The bool
// reports whether the input was a valid hex color.
func parseHex(s string) (r, g, b int, ok bool) {
	if !strings.HasPrefix(s, "#") {
		return 0, 0, 0, false
	}
	h := s[1:]
	// Expand shorthand "#abc" -> "aabbcc".
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, true
}

// ansiNameFg maps a color name to its foreground ANSI sequence. The base 8 use
// 30–37; the "bright_" variants use 90–97. Unknown names yield "".
func ansiNameFg(name string) string {
	if code, ok := ansiBase[name]; ok {
		return fmt.Sprintf("\x1b[%dm", 30+code)
	}
	if base, ok := strings.CutPrefix(name, "bright_"); ok {
		if code, ok := ansiBase[base]; ok {
			return fmt.Sprintf("\x1b[%dm", 90+code)
		}
	}
	return ""
}

// ansiNameBg maps a color name to its background ANSI sequence (foreground + 10:
// base 40–47, bright 100–107).
func ansiNameBg(name string) string {
	if code, ok := ansiBase[name]; ok {
		return fmt.Sprintf("\x1b[%dm", 40+code)
	}
	if base, ok := strings.CutPrefix(name, "bright_"); ok {
		if code, ok := ansiBase[base]; ok {
			return fmt.Sprintf("\x1b[%dm", 100+code)
		}
	}
	return ""
}

// ansiBase maps the 8 base color names to their ANSI offset (0–7).
var ansiBase = map[string]int{
	"black": 0, "red": 1, "green": 2, "yellow": 3,
	"blue": 4, "magenta": 5, "cyan": 6, "white": 7,
}
