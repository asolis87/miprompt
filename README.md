# miprompt

A fast, async, multi-shell prompt generator (zsh + fish), in the spirit of
Starship and Powerlevel10k. Built as a single static Go binary.

## Features

- **Async rendering** — the prompt appears instantly; expensive data (git dirty
  state, active node version) is computed in the background and the prompt is
  repainted when ready. Zero perceived lag even in large repos.
- **Segments** — current directory, git branch + dirty marker, Node.js version.
- **Two styles** — `plain` (colored text) and `powerline` (colored blocks with
  flowing separators).
- **Full theming** — every color is configurable via ANSI names (follow the
  terminal theme) or hex `#rrggbb` (exact truecolor).

## Install

```sh
go install .          # builds to ~/go/bin/miprompt
```

Make sure `~/go/bin` is on your `PATH`.

### zsh

Add to `~/.zshrc` (and remove any other prompt theme, e.g. Powerlevel10k —
two prompt managers fighting over `$PROMPT` will conflict):

```sh
eval "$(miprompt init zsh)"
```

### fish

Add to `~/.config/fish/config.fish`:

```fish
if status is-interactive
    miprompt init fish | source
end
```

> Note: `miprompt init fish | source`, NOT `eval`. fish's `eval` does not
> reliably handle the multiline function/event-handler block.

## Configuration

Optional file at `~/.config/miprompt/config.toml` (honors `$XDG_CONFIG_HOME`).
Everything has defaults; a missing or malformed file falls back to them — the
prompt never fails to render. See [`config.example.toml`](config.example.toml)
for all options.

```toml
style = "powerline"
symbol_on_new_line = false

[git]
color = "#bb9af7"      # hex (exact) or ANSI name ("magenta", "bright_blue")

[node]
show_active_version = true   # runs `node --version` (async) to show what's running
```

## How it works

- `init <shell>` prints the shell-specific integration snippet (it references
  the binary by absolute path, resolved at runtime).
- `prompt --shell <s> --exit-code <n>` renders the fast prompt synchronously.
- `compute <cache-file>` runs the expensive work in the background; the shell
  repaints when it finishes (zsh via `zle -F`, fish via `--on-process-exit` +
  `commandline -f repaint`).

## Development

```sh
go build -o miprompt .   # local build
go test ./...            # run tests
```
