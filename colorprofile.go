package main

import (
	"math"
	"os"
)

// colorProfile is the terminal's effective color capability, detected once from
// the environment. It decides how resolveFg/resolveBg emit color:
//   - profileNone     → no color at all (the user set NO_COLOR)
//   - profileANSI     → only the 16 named colors; hex is degraded to the nearest
//   - profileTrueColor→ full 24-bit hex, as authored
type colorProfile int

const (
	profileTrueColor colorProfile = iota
	profileANSI
	profileNone
)

// activeProfile is resolved once at startup (see detectColorProfile). resolveFg/
// resolveBg consult it. A package var keeps the resolve functions' signatures
// unchanged while still being overridable in tests.
var activeProfile = detectColorProfile()

// detectColorProfile reads the environment to decide the color capability,
// following the de-facto conventions other tools use:
//   - NO_COLOR present (any value) → disable color entirely (no-color.org).
//   - COLORTERM == "truecolor"/"24bit" → full 24-bit.
//   - otherwise → assume only the 16 ANSI colors and degrade hex to them.
func detectColorProfile() colorProfile {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return profileNone
	}
	switch os.Getenv("COLORTERM") {
	case "truecolor", "24bit":
		return profileTrueColor
	default:
		return profileANSI
	}
}

// ansi16 maps each of the 16 ANSI color names to a representative RGB, used to
// find the nearest named color when degrading a hex value. Names match the keys
// resolveFg/resolveBg already understand (base + "bright_").
var ansi16 = []struct {
	name    string
	r, g, b int
}{
	{"black", 0, 0, 0}, {"red", 205, 0, 0}, {"green", 0, 205, 0},
	{"yellow", 205, 205, 0}, {"blue", 0, 0, 238}, {"magenta", 205, 0, 205},
	{"cyan", 0, 205, 205}, {"white", 229, 229, 229},
	{"bright_black", 127, 127, 127}, {"bright_red", 255, 0, 0},
	{"bright_green", 0, 255, 0}, {"bright_yellow", 255, 255, 0},
	{"bright_blue", 92, 92, 255}, {"bright_magenta", 255, 0, 255},
	{"bright_cyan", 0, 255, 255}, {"bright_white", 255, 255, 255},
}

// nearestANSIName returns the ANSI color name closest to the given RGB, by
// Euclidean distance in RGB space. Simple and good enough for prompt segments;
// perceptual color spaces would be overkill here.
func nearestANSIName(r, g, b int) string {
	best := ansi16[0].name
	bestDist := math.MaxFloat64
	for _, c := range ansi16 {
		dr, dg, db := float64(r-c.r), float64(g-c.g), float64(b-c.b)
		d := dr*dr + dg*dg + db*db
		if d < bestDist {
			bestDist = d
			best = c.name
		}
	}
	return best
}
