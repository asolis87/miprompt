package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// gitInfo holds the git state we want to surface in the prompt. It grows as we
// add segments (stashes, ...). A nil *gitInfo means "not in a repo".
type gitInfo struct {
	branch      string // branch name, or short commit hash when in detached HEAD
	dirty       bool   // true when the working tree has uncommitted changes
	ahead       int    // local commits not in upstream ("need to push")
	behind      int    // upstream commits not local ("need to pull")
	hasUpstream bool   // false when the branch tracks no upstream (a/b N/A)
}

// expensiveMode selects how an EXPENSIVE piece of prompt data is resolved.
// It governs every segment whose data costs a fork (git dirty via `git status`,
// the active node version via `node --version`, ...). Cheap data (git branch,
// the project's requested node version) is always resolved inline regardless.
type expensiveMode int

const (
	expensiveNone    expensiveMode = iota // skip it (fast prompt, no fork)
	expensiveCompute                      // compute it now (the expensive fork)
	expensiveCache                        // read a precomputed result from cache
)

// readGitInfo returns the git state for the current directory, or nil if the
// directory is not inside a git repository.
//
// Before shelling out to `git` (an expensive fork+exec), it first checks the
// filesystem for a .git entry by walking up the directory tree. A stat is
// orders of magnitude cheaper than a process spawn, so when you are NOT in a
// repo the prompt pays no git cost at all.
//
// mode decides how the dirty marker is resolved (see expensiveMode). cacheFile
// is only used when mode is expensiveCache.
func readGitInfo(mode expensiveMode, cacheFile string) *gitInfo {
	if !insideGitRepo() {
		return nil // cheap exit: no fork to git when there is no repo
	}

	branch, ok := gitBranch()
	if !ok {
		return nil // .git found but git could not read it (corrupt/unavailable)
	}

	info := &gitInfo{branch: branch}

	// ahead/behind is cheap (compares two refs, no working-tree scan — measured
	// ~3ms over the branch), and it never touches the network, so we resolve it
	// synchronously alongside the branch rather than on the async path.
	info.ahead, info.behind, info.hasUpstream = gitAheadBehind()

	switch mode {
	case expensiveCompute:
		info.dirty = gitDirty() // EXPENSIVE: scans the working tree.
	case expensiveCache:
		info.dirty = readDirtyCache(cacheFile) // cheap: reads a one-byte file.
	}
	return info
}

// gitAheadBehind returns how many commits the current branch is ahead and behind
// its upstream, and whether an upstream exists at all.
//
// It compares against the upstream ref git ALREADY knows locally — it does NOT
// fetch. A prompt must never hit the network on every keystroke, so the counts
// reflect the last time you fetched/pulled, which is the correct tradeoff.
//
// When the branch tracks no upstream (e.g. a fresh local branch never pushed),
// the command fails and we report hasUpstream=false — distinct from "0 ahead,
// 0 behind", which means up to date.
func gitAheadBehind() (ahead, behind int, hasUpstream bool) {
	// Output is "<behind>\t<ahead>": left side = commits in upstream not in HEAD,
	// right side = commits in HEAD not in upstream.
	out, err := exec.Command("git", "rev-list", "--count", "--left-right", "@{upstream}...HEAD").Output()
	if err != nil {
		return 0, 0, false // no upstream (or other failure): a/b does not apply
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return 0, 0, false
	}
	behind, _ = strconv.Atoi(fields[0])
	ahead, _ = strconv.Atoi(fields[1])
	return ahead, behind, true
}

// readDirtyCache reads a dirty result previously written by `compute-dirty`.
// A missing or unreadable file means "assume clean" — the async pass will
// correct it on the next repaint.
func readDirtyCache(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(string(b)), "1")
}

// writeDirtyCache computes the dirty state for the current directory and writes
// it ("1" or "0") to path. This is what the async background job runs; the next
// prompt render reads the result via readDirtyCache. Returns the computed state.
func writeDirtyCache(path string) bool {
	dirty := false
	if insideGitRepo() {
		dirty = gitDirty()
	}
	val := "0"
	if dirty {
		val = "1"
	}
	_ = os.WriteFile(path, []byte(val+"\n"), 0o600)
	return dirty
}

// gitDirty reports whether the working tree has uncommitted changes.
//
// This is the EXPENSIVE git query: `git status` must inspect the working tree,
// which on a large repo means stat-ing many files. It is run synchronously here
// on purpose, so its cost is visible before we move it off the critical path.
//
// Flags chosen for speed:
//
//	--porcelain          stable, parseable output (not for humans)
//	--untracked-files=no  do NOT scan for untracked files; we only care whether
//	                      tracked files changed. Scanning untracked content is
//	                      often the slowest part of status in repos with huge
//	                      ignored trees (node_modules, build output).
func gitDirty() bool {
	out, err := exec.Command("git", "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil {
		return false
	}
	return len(out) > 0 // any output means there is at least one change
}

// insideGitRepo reports whether the current directory is inside a git
// repository, by walking up from the cwd looking for a .git entry.
//
// It checks for existence, not "is a directory": in linked worktrees and
// submodules, .git is a regular FILE pointing at the real git dir, not a
// directory. Treating it as dir-only would miss those repos.
func insideGitRepo() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	for {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false // reached the filesystem root
		}
		dir = parent
	}
}

// gitBranch returns the current branch name. In detached HEAD state (parked on
// a commit rather than a branch) it falls back to the short commit hash, since
// `--abbrev-ref` would otherwise return the useless literal "HEAD".
func gitBranch() (string, bool) {
	// Cheapest possible "what branch am I on?" query.
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", false // not inside a work tree
	}
	name := strings.TrimSpace(string(out))

	if name == "HEAD" {
		// Detached HEAD: show the short hash instead.
		if hash, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output(); err == nil {
			return strings.TrimSpace(string(hash)), true
		}
	}
	return name, true
}
