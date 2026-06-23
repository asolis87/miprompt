package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// colReset is the ANSI "reset all attributes" sequence. Concrete colors are
// resolved from config names/hex by resolveFg/resolveBg (see color.go).
const colReset = "\x1b[0m"

// fallbackPrompt is the last-resort prompt emitted when rendering panics. Ugly
// but always usable — a bare symbol so the user can keep typing. It carries no
// color (nothing to reset) and is wrapped per shell only for the symbol itself.
const fallbackPrompt = "❯ "

// renderImpl is the function safeRenderPrompt calls to do the actual work. It is
// a variable (not a direct call) so tests can swap in a panicking implementation
// to verify the recovery net actually catches it.
var renderImpl = renderPrompt

// safeRenderPrompt is the single entry point for prompt rendering. It guarantees
// the cardinal rule — the prompt ALWAYS renders something — by recovering from
// any panic inside the render (a nil deref in a segment, a bad config value,
// anything) and emitting a minimal fallback instead of a blank prompt or a stack
// trace. A prompt runs on every keystroke; a crash here would make the terminal
// unusable, so this net is non-negotiable.
func safeRenderPrompt(s Shell, exitCode, cmdDuration int, mode expensiveMode, cacheFile string, cfg Config) (out string) {
	defer func() {
		if r := recover(); r != nil {
			out = fallbackPrompt
		}
	}()
	return renderImpl(s, exitCode, cmdDuration, mode, cacheFile, cfg)
}

// renderPrompt builds the full prompt for the given shell and previous exit
// code. It now works in two phases: buildSegments produces the QUÉ (content +
// intended colors), then the configured Renderer produces the CÓMO (plain vs
// powerline). Adding a visual style means adding a Renderer, not touching any
// segment.
//
// mode + cacheFile decide how EXPENSIVE data is resolved: expensiveNone for the
// instant fast prompt, expensiveCompute for the synchronous full prompt, and
// expensiveCache to read async-computed results (the repaint path).
func renderPrompt(s Shell, exitCode, cmdDuration int, mode expensiveMode, cacheFile string, cfg Config) string {
	status := buildStatusSegments(cmdDuration, mode, cacheFile, cfg)
	symbol := segmentSymbol(exitCode, cfg.Symbol)

	statusLine := rendererFor(cfg).Render(s, status)

	// Layout of the prompt symbol — a CÓMO decision, so it lives here, not in a
	// segment. Three cases:
	//   - on a new line   → status row, then "❯ " below (both styles)
	//   - plain, inline   → "❯" follows the status on the same line (original look)
	//   - powerline,inline→ symbol omitted; the closing arrow IS the prompt end
	sym := colorize(s, resolveFg(symbol.fg), symbol.text)

	if cfg.SymbolOnNewLine {
		return statusLine + "\n" + sym + " "
	}
	if cfg.Style == "powerline" {
		return statusLine + " "
	}
	return statusLine + " " + sym + " "
}

// buildStatusSegments assembles the ordered STATUS segments (everything except
// the prompt symbol). Each helper returns plain content and color *names*; no
// ANSI is emitted here.
func buildStatusSegments(cmdDuration int, mode expensiveMode, cacheFile string, cfg Config) []Segment {
	var segs []Segment

	segs = append(segs, segmentCwd(cfg))

	// The git segment only appears when we are inside a repository.
	if git := readGitInfo(mode, cacheFile); git != nil {
		segs = append(segs, segmentGit(git, cfg.Git))
	}

	// The node segment only appears inside a Node project. The active version is
	// an opt-in expensive feature, so we only pass the expensive mode through
	// when the user enabled it; otherwise node stays in the cheap (file) path.
	nodeMode := expensiveNone
	if cfg.Node.ShowActiveVersion {
		nodeMode = mode
	}
	if node := readNodeInfo(nodeMode); node != nil {
		segs = append(segs, segmentNode(node, cfg.Node))
	}

	// The python segment only appears when a virtualenv/conda env is active —
	// read straight from the inherited environment, the cheapest source there is.
	if py := readPythonInfo(); py != nil {
		segs = append(segs, segmentPython(py, cfg.Python))
	}

	// The duration segment only appears when the last command was slow enough.
	if dur := segmentDuration(cmdDuration, cfg.Duration); dur != nil {
		segs = append(segs, *dur)
	}

	return segs
}

// segmentCwd describes the current directory, with $HOME collapsed to ~.
func segmentCwd(cfg Config) Segment {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "?"
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + strings.TrimPrefix(cwd, home)
	} else {
		cwd = filepath.Clean(cwd)
	}
	fg := orDefault(cfg.Cwd.Color, "blue")
	return Segment{text: cwd, fg: fg, bg: orDefault(cfg.Cwd.Bg, fg)}
}

// segmentGit describes the git state: a configurable icon, the branch name (or
// short hash in detached HEAD), ahead/behind counts vs upstream, and a dot when
// dirty. A dirty tree recolors the whole segment (dirty_color) — a single
// uniform color per segment is what lets it render as one powerline block.
func segmentGit(git *gitInfo, cfg GitConfig) Segment {
	label := git.branch
	if cfg.Icon != "" {
		label = cfg.Icon + " " + label
	}

	// ahead/behind only shows when there's an upstream AND a divergence — being
	// up to date adds no marker, keeping the prompt quiet in the common case.
	if git.hasUpstream {
		if git.ahead > 0 {
			label += " " + orDefault(cfg.AheadIcon, "↑") + strconv.Itoa(git.ahead)
		}
		if git.behind > 0 {
			label += " " + orDefault(cfg.BehindIcon, "↓") + strconv.Itoa(git.behind)
		}
	}

	fg := orDefault(cfg.Color, "yellow")
	bg := orDefault(cfg.Bg, fg)
	if git.dirty {
		label += " ●"
		fg = orDefault(cfg.DirtyColor, "red")
		bg = orDefault(cfg.DirtyBg, fg)
	}
	return Segment{text: label, fg: fg, bg: bg}
}

// segmentNode describes the Node project marker: a configurable icon plus a
// version. When the active version differs from the requested one it shows both
// ("18.0.0 ≠ 26.3.0") in the mismatch color.
func segmentNode(node *nodeInfo, cfg NodeConfig) Segment {
	version, mismatch := nodeVersionLabel(node)

	label := cfg.Icon
	if version != "" {
		if label != "" {
			label += " "
		}
		label += version
	}

	fg := orDefault(cfg.Color, "green")
	bg := orDefault(cfg.Bg, fg)
	if mismatch {
		fg = orDefault(cfg.MismatchColor, "red")
		bg = fg // mismatch always recolors the whole block
	}
	return Segment{text: label, fg: fg, bg: bg}
}

// nodeVersionLabel decides what version text to show and whether it represents
// a requested-vs-active mismatch. Active wins when versions agree (or no active
// is known); on mismatch it returns "requested ≠ active".
func nodeVersionLabel(node *nodeInfo) (text string, mismatch bool) {
	if node.active == "" {
		return node.version, false // no active resolved: just the requested one
	}
	if versionsMatch(node.version, node.active) {
		return node.active, false
	}
	return node.version + " ≠ " + node.active, true
}

// segmentSymbol describes the prompt character: success color when the previous
// command succeeded, error color when it failed. Char and colors come from
// config. It has no background, so it renders outside the powerline chain.
func segmentSymbol(exitCode int, cfg SymbolConfig) Segment {
	color := orDefault(cfg.SuccessColor, "green")
	if exitCode != 0 {
		color = orDefault(cfg.ErrorColor, "red")
	}
	return Segment{text: orDefault(cfg.Char, "❯"), fg: color}
}

// colorize wraps text in a color, marking the color codes as invisible so the
// shell computes the prompt width correctly. With color disabled (NO_COLOR) it
// emits the bare text — no color, no reset, no stray escapes. An empty color
// likewise skips the wrapping (and the now-pointless reset).
func colorize(s Shell, color, text string) string {
	if activeProfile == profileNone || color == "" {
		return text
	}
	return s.wrapInvisible(color) + text + s.wrapInvisible(colReset)
}

// orDefault returns name when non-empty, otherwise def. Lets segments carry a
// color *name* while still honoring a built-in fallback.
func orDefault(name, def string) string {
	if name == "" {
		return def
	}
	return name
}
