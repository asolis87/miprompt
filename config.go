package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user-overridable settings. Every field has a sensible default
// (see defaultConfig), so the prompt renders correctly even with no config file
// at all. The file only overrides defaults — a prompt must never fail to draw
// because its configuration is missing or malformed.
type Config struct {
	// Theme is a named color preset applied before the user's own keys (see
	// theme.go). Defaults to "huginn". Set "" or an unknown name to keep the
	// plain built-in defaults. User config keys still override the theme.
	Theme string `toml:"theme"`
	// Style selects the visual renderer: "plain" (text colored, space-joined,
	// the original look) or "powerline" (colored blocks joined by flowing
	// separator glyphs). Unknown values fall back to plain.
	Style string `toml:"style"`
	// SymbolOnNewLine puts the prompt symbol (❯) on its own second line, so you
	// type below the status row. When false (default): plain keeps the symbol at
	// the end of the line as before; powerline omits the symbol entirely and ends
	// on its closing arrow.
	SymbolOnNewLine bool           `toml:"symbol_on_new_line"`
	Cwd             CwdConfig      `toml:"cwd"`
	Git             GitConfig      `toml:"git"`
	Node            NodeConfig     `toml:"node"`
	Duration        DurationConfig `toml:"duration"`
	Symbol          SymbolConfig   `toml:"symbol"`
}

// DurationConfig configures the last-command-duration segment.
type DurationConfig struct {
	MinMs int    `toml:"min_ms"` // only show when the command took at least this long
	Icon  string `toml:"icon"`   // glyph before the duration
	Color string `toml:"color"`  // text/fg color
	Bg    string `toml:"bg"`     // powerline block bg; "" = same as Color
}

// CwdConfig configures the current-directory segment.
type CwdConfig struct {
	Color string `toml:"color"` // text/fg color: ANSI name or hex (see color.go)
	Bg    string `toml:"bg"`    // powerline block background; "" = same as Color
}

// SymbolConfig configures the prompt character (❯) and its colors.
type SymbolConfig struct {
	Char         string `toml:"char"`          // the prompt glyph
	SuccessColor string `toml:"success_color"` // color when last command succeeded
	ErrorColor   string `toml:"error_color"`   // color when last command failed
}

// GitConfig configures the git segment.
type GitConfig struct {
	Icon       string `toml:"icon"`        // glyph shown before the branch name
	Color      string `toml:"color"`       // text/fg color: ANSI name or hex
	Bg         string `toml:"bg"`          // powerline block bg; "" = same as Color
	DirtyColor string `toml:"dirty_color"` // text/fg color when the tree is dirty
	DirtyBg    string `toml:"dirty_bg"`    // powerline block bg when dirty; "" = DirtyColor
	AheadIcon  string `toml:"ahead_icon"`  // glyph before the ahead count (commits to push)
	BehindIcon string `toml:"behind_icon"` // glyph before the behind count (commits to pull)
}

// NodeConfig configures the Node.js segment.
type NodeConfig struct {
	Icon  string `toml:"icon"`  // glyph shown for a Node project
	Color string `toml:"color"` // text/fg color: ANSI name or hex
	Bg    string `toml:"bg"`    // powerline block bg; "" = same as Color
	// ShowActiveVersion, when true, resolves the ACTIVE node version by running
	// `node --version` (~19ms, off the critical path via the async pass) instead
	// of only the version the project declares in a file. Off by default because
	// it costs a fork; opt in when you want to catch "wrong version active".
	ShowActiveVersion bool `toml:"show_active_version"`
	// MismatchColor is the color used when the active version differs from the
	// version the project requests — the whole point of ShowActiveVersion. Only
	// meaningful with that feature on.
	MismatchColor string `toml:"mismatch_color"`
}

// defaultConfig returns the built-in base defaults. These are the bare values
// before any theme; "huginn" is the default theme layered on top (see
// loadConfig). They double as the fallback when theme = "" or unknown.
func defaultConfig() Config {
	return Config{
		Theme: "huginn",
		Style: "plain",
		Cwd: CwdConfig{
			Color: "blue",
		},
		Duration: DurationConfig{
			MinMs: 2000, // only commands slower than 2s
			Icon:  "",   // nf-md-timer_outline
			Color: "yellow",
		},
		Symbol: SymbolConfig{
			Char:         "❯",
			SuccessColor: "green",
			ErrorColor:   "red",
		},
		Git: GitConfig{
			Icon:       "", // nf-pl-branch (branch glyph)
			Color:      "yellow",
			DirtyColor: "red",
			AheadIcon:  "↑",
			BehindIcon: "↓",
		},
		Node: NodeConfig{
			Icon:          "", // nf-dev-nodejs_small
			Color:         "green",
			MismatchColor: "red",
		},
	}
}

// loadConfig builds the effective config in three cascading layers:
//
//	base defaults  →  named theme  →  user config.toml
//	  (weakest)                          (strongest)
//
// The theme is chosen by the user's `theme` key (default "huginn"), so we read
// the file once to learn the theme, seed the theme colors, then decode the file
// again on top so per-field user overrides still win. A missing file yields the
// default theme; a malformed file is tolerated (we keep what we have).
func loadConfig() Config {
	cfg := defaultConfig()

	path := configPath()
	if path == "" {
		applyTheme(&cfg, cfg.Theme) // no file: still apply the default theme
		return cfg
	}

	// Pass 1: learn which theme the user wants (their key overrides the default).
	_, _ = toml.DecodeFile(path, &cfg)

	// Layer the theme over the base defaults.
	applyTheme(&cfg, cfg.Theme)

	// Pass 2: re-apply the user's file so explicit per-field overrides beat the
	// theme. Absent keys keep the theme's values.
	_, _ = toml.DecodeFile(path, &cfg)
	return cfg
}

// configPath returns the location of the config file, honoring XDG_CONFIG_HOME
// and falling back to ~/.config. Returns "" if no home can be determined.
func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "huginn", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "huginn", "config.toml")
}
