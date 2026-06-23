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
	// Style selects the visual renderer: "plain" (text colored, space-joined,
	// the original look) or "powerline" (colored blocks joined by flowing
	// separator glyphs). Unknown values fall back to plain.
	Style string `toml:"style"`
	// SymbolOnNewLine puts the prompt symbol (❯) on its own second line, so you
	// type below the status row. When false (default): plain keeps the symbol at
	// the end of the line as before; powerline omits the symbol entirely and ends
	// on its closing arrow.
	SymbolOnNewLine bool         `toml:"symbol_on_new_line"`
	Cwd             CwdConfig    `toml:"cwd"`
	Git             GitConfig    `toml:"git"`
	Node            NodeConfig   `toml:"node"`
	Symbol          SymbolConfig `toml:"symbol"`
}

// CwdConfig configures the current-directory segment.
type CwdConfig struct {
	Color string `toml:"color"` // ANSI name or hex (see color.go)
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
	Color      string `toml:"color"`       // ANSI name or hex (see color.go)
	DirtyColor string `toml:"dirty_color"` // color for the dirty marker
}

// NodeConfig configures the Node.js segment.
type NodeConfig struct {
	Icon  string `toml:"icon"`  // glyph shown for a Node project
	Color string `toml:"color"` // color name (see colorByName)
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

// defaultConfig returns the built-in defaults. These are the values used when
// no config file exists or when a field is left unset.
func defaultConfig() Config {
	return Config{
		Style: "plain",
		Cwd: CwdConfig{
			Color: "blue",
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
		},
		Node: NodeConfig{
			Icon:  "", // nf-dev-nodejs_small
			Color:         "green",
			MismatchColor: "red",
		},
	}
}

// loadConfig reads and decodes the config file, layering it over the defaults.
// A missing file is normal (returns defaults). A malformed file is tolerated:
// we keep whatever decoded plus defaults, rather than failing the prompt.
func loadConfig() Config {
	cfg := defaultConfig()

	path := configPath()
	if path == "" {
		return cfg
	}
	// Decode on top of cfg: present keys override defaults, absent keys keep them.
	// On a decode error we deliberately ignore it and return what we have.
	_, _ = toml.DecodeFile(path, &cfg)
	return cfg
}

// configPath returns the location of the config file, honoring XDG_CONFIG_HOME
// and falling back to ~/.config. Returns "" if no home can be determined.
func configPath() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "miprompt", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "miprompt", "config.toml")
}

