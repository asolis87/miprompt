package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "prompt":
		runPrompt(os.Args[2:])
	case "compute":
		runCompute(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

// runCompute runs the EXPENSIVE work and writes results to caches. This is the
// command the async background job invokes (fish's path); the next prompt render
// picks up the results from cache. It is the unit of work the shell pushes off
// the critical path.
//
//   - dirty: written to the given per-session cache file (read via --dirty-from)
//   - node active version: written to a per-directory cache, but only when the
//     user opted in, to avoid a wasteful `node` fork when the feature is off.
func runCompute(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: huginn compute <dirty-cache-file>")
		os.Exit(1)
	}
	writeDirtyCache(args[0])

	if loadConfig().Node.ShowActiveVersion {
		writeNodeCache()
		// Opportunistically clean up stale per-directory caches. Safe here:
		// compute runs in the background, off the prompt's critical path.
		sweepNodeCaches(24*time.Hour, time.Now())
	}
}

// runInit prints the integration snippet for the requested shell.
func runInit(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: huginn init <zsh|bash|fish>")
		os.Exit(1)
	}
	shell, ok := parseShell(args[0])
	if !ok {
		fmt.Fprintf(os.Stderr, "unsupported shell: %s\n", args[0])
		os.Exit(1)
	}
	fmt.Println(initScript(shell))
}

// runPrompt renders the prompt. Flags are parsed by hand to keep startup fast
// and the mechanism visible: --shell <s> --exit-code <n>.
func runPrompt(args []string) {
	shell := ShellZsh
	exitCode := 0
	cmdDuration := 0      // ms; measured by the shell and passed in
	mode := expensiveNone // default: fast prompt, no expensive forks
	cacheFile := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--shell":
			i++
			if i < len(args) {
				if s, ok := parseShell(args[i]); ok {
					shell = s
				}
			}
		case "--exit-code":
			i++
			if i < len(args) {
				exitCode, _ = strconv.Atoi(args[i])
			}
		case "--cmd-duration":
			// Duration of the previous command in ms. The binary can't measure
			// this (it's an ephemeral process); only the shell knows it.
			i++
			if i < len(args) {
				cmdDuration, _ = strconv.Atoi(args[i])
			}
		case "--full":
			// Compute all expensive data now (synchronous full prompt, e.g.
			// zsh's async background render).
			mode = expensiveCompute
		case "--dirty-from":
			// Read expensive results from cache (fish's fast prompt reads what
			// the async job left behind). The arg is the per-session dirty file;
			// the node active version uses its own per-directory cache.
			i++
			if i < len(args) {
				mode = expensiveCache
				cacheFile = args[i]
			}
		}
	}

	// Use Print, not Println: a trailing newline would push the prompt onto
	// its own line. The shell decides spacing.
	fmt.Print(renderPrompt(shell, exitCode, cmdDuration, mode, cacheFile, loadConfig()))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: huginn <init|prompt> [args]")
}
