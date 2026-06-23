package main

import (
	"strings"
	"testing"
)

// These tests assert PROPERTIES of the rendered output (content present, right
// number of separators, correct shell escaping) rather than exact byte strings.
// Property tests survive cosmetic tweaks and only fail on real regressions.

// --- Plain renderer ----------------------------------------------------------

func TestPlainRenderShowsAllSegmentText(t *testing.T) {
	segs := []Segment{
		{text: "~/dev", fg: "blue", bg: "blue"},
		{text: "main", fg: "yellow", bg: "yellow"},
	}
	out := plainRenderer{}.Render(ShellZsh, segs)

	for _, seg := range segs {
		if !strings.Contains(out, seg.text) {
			t.Errorf("plain output missing segment text %q\ngot: %q", seg.text, out)
		}
	}
}

func TestPlainRenderUncoloredSegmentHasNoAnsi(t *testing.T) {
	// A segment with no fg should render as bare text — no escape sequences.
	out := plainRenderer{}.Render(ShellZsh, []Segment{{text: "plain"}})
	if strings.Contains(out, "\x1b[") {
		t.Errorf("uncolored segment emitted ANSI: %q", out)
	}
}

// --- Shell escaping (the easiest thing to break silently) --------------------

func TestColorWrappingPerShell(t *testing.T) {
	segs := []Segment{{text: "x", fg: "red", bg: "red"}}

	zsh := plainRenderer{}.Render(ShellZsh, segs)
	if !strings.Contains(zsh, "%{") || !strings.Contains(zsh, "%}") {
		t.Errorf("zsh render must wrap escapes in %%{ %%}: %q", zsh)
	}

	fish := plainRenderer{}.Render(ShellFish, segs)
	if strings.Contains(fish, "%{") {
		t.Errorf("fish render must NOT use zsh %%{ %%} delimiters: %q", fish)
	}
}

// --- Powerline renderer ------------------------------------------------------

func TestPowerlineHasSeparatorBetweenBlocks(t *testing.T) {
	// N blocks → N separators: one between each pair PLUS the closing cap on the
	// last block. (Closing arrow is part of the powerline look.)
	segs := []Segment{
		{text: "a", fg: "white", bg: "blue"},
		{text: "b", fg: "white", bg: "green"},
		{text: "c", fg: "white", bg: "red"},
	}
	out := powerlineRenderer{}.Render(ShellFish, segs)

	got := strings.Count(out, powerlineRight)
	if got != len(segs) {
		t.Errorf("powerline with %d blocks: got %d separators, want %d\n%q",
			len(segs), got, len(segs), out)
	}
}

func TestPowerlineShowsAllSegmentText(t *testing.T) {
	segs := []Segment{
		{text: "~/dev", fg: "white", bg: "blue"},
		{text: "main", fg: "white", bg: "green"},
	}
	out := powerlineRenderer{}.Render(ShellFish, segs)

	for _, seg := range segs {
		if !strings.Contains(out, seg.text) {
			t.Errorf("powerline output missing segment text %q\ngot: %q", seg.text, out)
		}
	}
}

func TestPowerlineSkipsBackgroundlessSegments(t *testing.T) {
	// Segments without a bg are not part of the powerline chain (the prompt
	// symbol is handled separately). They contribute no separator.
	segs := []Segment{
		{text: "block", fg: "white", bg: "blue"},
		{text: "loose", fg: "green"}, // no bg
	}
	out := powerlineRenderer{}.Render(ShellFish, segs)

	// One block → one separator (its closing cap), not two.
	if got := strings.Count(out, powerlineRight); got != 1 {
		t.Errorf("expected 1 separator for 1 block, got %d: %q", got, out)
	}
}

// --- Symbol layout (renderPrompt orchestration) ------------------------------

func TestSymbolLayoutNewLine(t *testing.T) {
	cfg := defaultConfig()
	cfg.SymbolOnNewLine = true
	out := renderPrompt(ShellFish, 0, 0, expensiveNone, "", cfg)

	if !strings.Contains(out, "\n") {
		t.Errorf("symbol_on_new_line should put the symbol on its own line: %q", out)
	}
	if !strings.Contains(out, cfg.Symbol.Char) {
		t.Errorf("symbol char %q missing: %q", cfg.Symbol.Char, out)
	}
}

func TestSymbolOmittedInPowerlineInline(t *testing.T) {
	cfg := defaultConfig()
	cfg.Style = "powerline"
	cfg.SymbolOnNewLine = false
	cfg.Symbol.Char = "❯"
	out := renderPrompt(ShellFish, 0, 0, expensiveNone, "", cfg)

	// Powerline inline ends on the closing arrow; the symbol is omitted.
	if strings.Contains(out, "❯") {
		t.Errorf("powerline inline should omit the prompt symbol: %q", out)
	}
}
