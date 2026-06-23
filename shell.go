package main

// Shell identifies which shell we are talking to. Each shell hooks into the
// prompt differently and uses different delimiters to mark zero-width
// (invisible) sequences like color codes, so almost every decision in this
// tool branches on it.
type Shell string

const (
	ShellZsh  Shell = "zsh"
	ShellBash Shell = "bash"
	ShellFish Shell = "fish"
)

func parseShell(s string) (Shell, bool) {
	switch Shell(s) {
	case ShellZsh, ShellBash, ShellFish:
		return Shell(s), true
	default:
		return "", false
	}
}

// wrapInvisible wraps a non-printing sequence (e.g. an ANSI color code) in the
// delimiters the shell needs to correctly compute the prompt's display width.
// Get this wrong and the cursor position breaks when navigating history.
func (s Shell) wrapInvisible(seq string) string {
	switch s {
	case ShellZsh:
		return "%{" + seq + "%}"
	case ShellBash:
		return "\\[" + seq + "\\]"
	default: // fish handles width without delimiters
		return seq
	}
}
