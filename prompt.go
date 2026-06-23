package main

import (
	"os"
	"path/filepath"
	"strings"
)

// colReset is the ANSI "reset all attributes" sequence. Concrete colors are
// resolved from config names/hex by resolveFg/resolveBg (see color.go).
const colReset = "\x1b[0m"

// renderPrompt builds the full prompt for the given shell and previous exit
// code. It now works in two phases: buildSegments produces the QUÉ (content +
// intended colors), then the configured Renderer produces the CÓMO (plain vs
// powerline). Adding a visual style means adding a Renderer, not touching any
// segment.
//
// mode + cacheFile decide how EXPENSIVE data is resolved: expensiveNone for the
// instant fast prompt, expensiveCompute for the synchronous full prompt, and
// expensiveCache to read async-computed results (the repaint path).
func renderPrompt(s Shell, exitCode int, mode expensiveMode, cacheFile string, cfg Config) string {
	status := buildStatusSegments(mode, cacheFile, cfg)
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
func buildStatusSegments(mode expensiveMode, cacheFile string, cfg Config) []Segment {
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
	color := orDefault(cfg.Cwd.Color, "blue")
	return Segment{text: cwd, fg: color, bg: color}
}

// segmentGit describes the git state: a configurable icon, the branch name (or
// short hash in detached HEAD), and a dot when dirty. A dirty tree recolors the
// whole segment (dirty_color) — a single uniform color per segment is what lets
// it render as one powerline block.
func segmentGit(git *gitInfo, cfg GitConfig) Segment {
	label := git.branch
	if cfg.Icon != "" {
		label = cfg.Icon + " " + label
	}

	color := orDefault(cfg.Color, "yellow")
	if git.dirty {
		label += " ●"
		color = orDefault(cfg.DirtyColor, "red")
	}
	return Segment{text: label, fg: color, bg: color}
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

	color := orDefault(cfg.Color, "green")
	if mismatch {
		color = orDefault(cfg.MismatchColor, "red")
	}
	return Segment{text: label, fg: color, bg: color}
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
// shell computes the prompt width correctly.
func colorize(s Shell, color, text string) string {
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
