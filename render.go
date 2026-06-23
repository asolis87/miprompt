package main

import "strings"

// Segment is the QUÉ: a piece of prompt content plus its intended colors, with
// no knowledge of how it will be drawn. Segments describe themselves; a Renderer
// decides how to compose them (plain spacing vs powerline blocks). This split is
// what lets one set of segments support multiple visual styles.
type Segment struct {
	text string // visible content
	fg   string // foreground color name (see colorByName); "" = terminal default
	bg   string // background color name; "" = no background (transparent)
}

// Renderer is the CÓMO: it turns a list of segments into the final prompt string
// for a given shell. Different renderers produce different visual styles from
// the same segments.
type Renderer interface {
	Render(s Shell, segs []Segment) string
}

// rendererFor selects the renderer for the configured style, defaulting to
// plain for empty or unknown values.
func rendererFor(cfg Config) Renderer {
	switch cfg.Style {
	case "powerline":
		return powerlineRenderer{}
	default:
		return plainRenderer{}
	}
}

// Color values (fg/bg) are resolved via resolveFg/resolveBg (see color.go),
// which accept both ANSI names ("blue", "bright_green") and hex ("#7aa2f7").

// ---- Plain renderer: the original look, segments joined by spaces -----------

type plainRenderer struct{}

// Render draws each segment's text in its foreground color (backgrounds are
// ignored in plain style), separated by single spaces — reproducing the prompt
// exactly as it was before theming existed.
func (plainRenderer) Render(s Shell, segs []Segment) string {
	var b strings.Builder
	for i, seg := range segs {
		if i > 0 {
			b.WriteString(" ")
		}
		if fg := resolveFg(seg.fg); fg != "" {
			b.WriteString(colorize(s, fg, seg.text))
		} else {
			b.WriteString(seg.text)
		}
	}
	return b.String()
}

// ---- Powerline renderer: colored blocks joined by flowing separators --------

// Powerline glyphs (Nerd Font private-use area).
const (
	powerlineRight = "" //  separator pointing right
)

type powerlineRenderer struct {
	separator string // override glyph; defaults to powerlineRight when empty
}

// Render draws each segment as a colored block (text on its background, padded
// with spaces), joined by a separator glyph. The trick that makes a block look
// like it "flows" into the next: the separator is drawn in the CURRENT block's
// background color over the NEXT block's background. So a blue block ending
// before a yellow block emits a blue arrow on a yellow field.
//
// Only segments that declare a background take part in the chain; any without
// one are skipped (the prompt symbol is handled separately by renderPrompt).
func (p powerlineRenderer) Render(s Shell, segs []Segment) string {
	sep := p.separator
	if sep == "" {
		sep = powerlineRight
	}

	var chain []Segment
	for _, seg := range segs {
		if seg.bg != "" {
			chain = append(chain, seg)
		}
	}

	var b strings.Builder
	for i, seg := range chain {
		// The block: " text " over this segment's bg. Text color: use the
		// segment's own fg when it differs from the bg (themes set an explicit
		// light-on-dark fg); otherwise fall back to an auto contrast color, since
		// fg==bg would be invisible.
		textColor := resolveFg(seg.fg)
		if seg.fg == "" || seg.fg == seg.bg {
			textColor = textColorOn(seg.bg)
		}
		b.WriteString(seq(s, resolveBg(seg.bg)))
		b.WriteString(seq(s, textColor))
		b.WriteString(" " + seg.text + " ")

		// The separator between this block and the next.
		if i+1 < len(chain) {
			next := chain[i+1]
			// Arrow in THIS bg color, over the NEXT bg color: the flow trick.
			b.WriteString(seq(s, resolveFg(seg.bg)))
			b.WriteString(seq(s, resolveBg(next.bg)))
			b.WriteString(sep)
		} else {
			// Final cap: the arrow is the last block's color flowing into the
			// terminal background. Reset FIRST (clears the block's bg so the
			// arrow sits on transparency), THEN set only the arrow's foreground.
			b.WriteString(seq(s, colReset))
			b.WriteString(seq(s, resolveFg(seg.bg)))
			b.WriteString(sep)
		}
	}
	b.WriteString(seq(s, colReset))
	return b.String()
}

// textColorOn returns a readable text-color ANSI sequence for a given block
// background — black on light backgrounds, white on dark — so the label stays
// legible regardless of the segment's identity color. Works for both hex (via
// luminance) and ANSI names (via a small lookup of the "light" ones).
func textColorOn(bg string) string {
	light := false
	if r, g, b, ok := parseHex(bg); ok {
		// Rec. 601 luma; > ~0.5 of full range reads as a light background.
		light = (299*r+587*g+114*b)/1000 > 140
	} else {
		switch bg {
		case "white", "yellow", "cyan", "green",
			"bright_white", "bright_yellow", "bright_cyan", "bright_green":
			light = true
		}
	}
	if light {
		return "\x1b[30m" // black text on light backgrounds
	}
	return "\x1b[97m" // bright white text on dark backgrounds
}

// seq wraps an ANSI sequence in the shell's invisible-width delimiters. An empty
// sequence — or color disabled (NO_COLOR) — yields nothing, so no stray escapes
// (including resets) leak into a no-color prompt.
func seq(s Shell, ansi string) string {
	if ansi == "" || activeProfile == profileNone {
		return ""
	}
	return s.wrapInvisible(ansi)
}
