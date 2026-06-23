package main

import (
	"fmt"
	"os"
)

// initScript returns the shell-specific snippet that wires this binary into the
// shell's prompt. The user adds a single `eval "$(miprompt init <shell>)"` line
// to their config; this function owns everything that line expands to.
//
// The snippet references the binary by its absolute path (resolved at runtime)
// rather than the bare name "miprompt". This makes integration robust: it works
// even when the shell's rc files rewrite PATH and the binary is not installed in
// a PATH directory yet — exactly the situation during local development.
func initScript(s Shell) string {
	bin, err := os.Executable()
	if err != nil {
		bin = "miprompt" // fall back to PATH lookup if resolution fails
	}

	switch s {
	case ShellZsh:
		// Asynchronous prompt (two-pass) for zsh.
		//
		// Pass 1 (precmd, instant): render the FAST prompt (branch only, no
		// working-tree scan) and show it immediately.
		//
		// Pass 2 (async): spawn the binary with --full writing to a temp
		// file, and register a `zle -F` handler on the job's file descriptor.
		// When the background job finishes, zsh invokes the handler, which reads
		// the FULL prompt and calls `zle reset-prompt` to repaint — the dirty
		// marker appears with zero perceived lag.
		return fmt.Sprintf(`_miprompt_bin=%q

# Pass 2: handler invoked by `+"`zle -F`"+` when the async job's fd is ready.
_miprompt_async_done() {
  # zsh passes the watched fd as $1. Capture it in a named variable so the
  # {fd}<&- close syntax (which needs an identifier, not $1) works.
  local fd=$1
  zle -F "$fd"        # unregister the watcher (fire once)
  exec {fd}<&-        # close the fd

  # The fd was just a readiness signal; the real prompt is in the temp file.
  if [[ -s "$_miprompt_async_file" ]]; then
    PROMPT="$(<"$_miprompt_async_file")"
    zle reset-prompt   # redraw the prompt with the freshly computed dirty state
  fi
  command rm -f "$_miprompt_async_file" 2>/dev/null
}

_miprompt_precmd() {
  local last=$?

  # Pass 1: instant FAST prompt (no dirty scan).
  PROMPT="$("$_miprompt_bin" prompt --shell zsh --exit-code $last)"

  # Pass 2: compute the FULL prompt (with dirty) in the background.
  _miprompt_async_file="${TMPDIR:-/tmp}/miprompt.$$"
  exec {_miprompt_fd}< <("$_miprompt_bin" prompt --shell zsh --exit-code $last --full > "$_miprompt_async_file"; printf '\n')
  zle -F "$_miprompt_fd" _miprompt_async_done
}

precmd_functions+=(_miprompt_precmd)`, bin)

	case ShellBash:
		return fmt.Sprintf(`_miprompt_precmd() {
  PS1="$(%q prompt --shell bash --exit-code $?)"
}
PROMPT_COMMAND="_miprompt_precmd"`, bin)

	case ShellFish:
		// Asynchronous prompt (two-pass) for fish.
		//
		// fish differs from zsh in two ways the docs make explicit: events do
		// NOT cross processes, and functions cannot run in the background. So
		// the background job (an external command, which CAN background) writes
		// the dirty state to a cache file, and `--on-process-exit` triggers
		// `commandline -f repaint` to re-run fish_prompt, which reads the cache.
		//
		// CRITICAL: `repaint` re-executes fish_prompt. If the async job were
		// launched inside fish_prompt, the repaint would relaunch it forever
		// (an infinite git-forking loop). Two defenses prevent this:
		//   1. The recompute is kicked off from a `_miprompt_pending` flag, set
		//      only by fish_postexec (fires once per command, NOT on repaint).
		//   2. The flag is cleared the moment the job launches, so a repaint
		//      re-running fish_prompt finds nothing pending and does not relaunch.
		return fmt.Sprintf(`set -g _miprompt_bin %q
set -g _miprompt_cache "$TMPDIR/miprompt-dirty.$fish_pid"
test -z "$TMPDIR"; and set -g _miprompt_cache "/tmp/miprompt-dirty.$fish_pid"
set -g _miprompt_pending 1   # compute on the very first prompt too

# Mark a fresh recompute as needed after each command (not on repaint).
function _miprompt_postexec --on-event fish_postexec
  set -g _miprompt_pending 1
end

function fish_prompt
  # Pass 1: instant prompt, reading the dirty marker from cache (if any).
  $_miprompt_bin prompt --shell fish --exit-code $status --dirty-from $_miprompt_cache

  # Pass 2: kick off the async recompute, but only when pending — this is what
  # makes the repaint safe (a repaint finds pending=0 and does nothing).
  if test "$_miprompt_pending" = 1
    set -g _miprompt_pending 0
    $_miprompt_bin compute $_miprompt_cache &
    set -g _miprompt_job $last_pid
    function _miprompt_on_done --on-process-exit $_miprompt_job
      functions -e _miprompt_on_done   # one-shot: remove self
      commandline -f repaint
    end
  end
end`, bin)
	}
	return fmt.Sprintf("# unsupported shell: %s", s)
}
