package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// nodeCachePrefix is the filename prefix shared by every per-directory node
// cache file, used both to build and to sweep them.
const nodeCachePrefix = "miprompt-node."

// nodeInfo holds the Node.js context for the prompt. A nil *nodeInfo means
// "this is not a Node project" and the segment is hidden entirely.
type nodeInfo struct {
	version string // the version the project requests, or "" if undeclared
	active  string // the ACTIVE node version (`node --version`), when resolved
}

// nodeMarkers are the files whose presence marks a directory as a Node project.
// package.json is the canonical one; the version files cover nvm/fnm/asdf users.
var nodeMarkers = []string{"package.json", ".nvmrc", ".node-version"}

// readNodeInfo returns the Node context for the current directory, or nil when
// not inside a Node project.
//
// The requested version (from a file) is always cheap, so it is read inline.
// The ACTIVE version requires running `node` (~19ms), so it follows the same
// expensive-data discipline as the git dirty marker: mode selects whether to
// skip it, compute it now (off the critical path), or read it from a cache.
//
// Cache asymmetry vs git dirty: the dirty cache is per-session because it is
// recomputed every prompt. The active-version cache is per-DIRECTORY (keyed by
// a hash of the cwd) because it is reused to avoid the fork, and the active
// version differs between projects — a single shared file would show project
// A's version while standing in project B.
func readNodeInfo(mode expensiveMode) *nodeInfo {
	root, ok := findNodeProjectRoot()
	if !ok {
		return nil // not a Node project: hide the segment
	}

	info := &nodeInfo{version: readRequestedNodeVersion(root)}
	switch mode {
	case expensiveCompute:
		info.active = activeNodeVersion() // EXPENSIVE: forks `node`.
	case expensiveCache:
		info.active = readVersionFile(nodeCachePath()) // cheap: reads a file.
	}
	return info
}

// versionsMatch reports whether the active version satisfies the requested one.
//
// It is deliberately NOT a semver engine. Implementing caret/tilde/comparator
// ranges correctly needs numeric version comparison and is overkill for a
// prompt. The guiding principle is "never cry wolf": a false mismatch warning
// is worse than silently missing one. So we only warn on a PINNED requirement
// — an exact version the project commits to ("18", "18.0", "v18.0.0"). Any range
// syntax (^, ~, >=, *, x, ||, ...) is intentionally flexible, so we never warn
// about it. Empty or non-numeric ("lts/hydrogen") requirements never warn.
func versionsMatch(requested, active string) bool {
	if requested == "" || active == "" {
		return true
	}
	if isVersionRange(requested) {
		return true // ranges are flexible by design: do not warn
	}

	req := leadingVersionParts(requested)
	act := leadingVersionParts(active)
	if len(req) == 0 || len(act) == 0 {
		return true // nothing numeric to compare: don't warn
	}

	// Pinned requirement: every part the project specifies must match.
	for i, part := range req {
		if i >= len(act) || part != act[i] {
			return false
		}
	}
	return true
}

// isVersionRange reports whether a requirement string uses any semver range
// syntax, meaning it is NOT a single pinned version. A leading "v" (e.g.
// "v18.0.0") is still pinned.
func isVersionRange(req string) bool {
	return strings.ContainsAny(req, "^~><=*xX |-") &&
		!strings.HasPrefix(strings.TrimSpace(req), "v")
}

// leadingVersionParts extracts the leading dotted-number components of a version
// string, e.g. ">=18.0" -> ["18","0"], "v20.11.0" -> ["20","11","0"].
func leadingVersionParts(v string) []string {
	// Find the first digit, then read the dotted-number run from there.
	start := strings.IndexFunc(v, func(r rune) bool { return r >= '0' && r <= '9' })
	if start < 0 {
		return nil
	}
	var parts []string
	for _, seg := range strings.Split(v[start:], ".") {
		num := ""
		for _, r := range seg {
			if r < '0' || r > '9' {
				break
			}
			num += string(r)
		}
		if num == "" {
			break
		}
		parts = append(parts, num)
	}
	return parts
}

// activeNodeVersion runs `node --version` and returns the version without the
// leading "v" (e.g. "20.11.0"). Returns "" if node is not on PATH. This is the
// expensive call the async pass keeps off the critical path.
func activeNodeVersion() string {
	out, err := exec.Command("node", "--version").Output()
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

// cacheDir is the directory holding all per-directory node caches: $TMPDIR, or
// /tmp when unset.
func cacheDir() string {
	if dir := os.Getenv("TMPDIR"); dir != "" {
		return dir
	}
	return "/tmp"
}

// nodeCachePath returns the per-directory cache file for the active node
// version: a hash of the cwd keeps each project's cached version separate.
func nodeCachePath() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}
	sum := sha1.Sum([]byte(cwd))
	key := hex.EncodeToString(sum[:8]) // 8 bytes is plenty to avoid collisions
	return filepath.Join(cacheDir(), nodeCachePrefix+key)
}

// writeNodeCache computes the active node version for the current directory and
// writes it to that directory's cache file. This is what the async background
// job runs; the next prompt reads it via expensiveCache. A nil result (node not
// found) writes an empty file so the segment simply omits the active version.
func writeNodeCache() {
	_ = os.WriteFile(nodeCachePath(), []byte(activeNodeVersion()+"\n"), 0o600)
}

// sweepNodeCaches deletes per-directory node caches older than maxAge. Because
// each visited directory leaves its own file, they would otherwise accumulate.
//
// It runs from `compute` (the async background job), never from prompt render,
// so its filesystem listing cost never touches the critical path. The TTL also
// doubles as expiry: a stale cache is removed and recomputed on the next visit.
func sweepNodeCaches(maxAge time.Duration, now time.Time) {
	entries, err := os.ReadDir(cacheDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), nodeCachePrefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > maxAge {
			_ = os.Remove(filepath.Join(cacheDir(), e.Name()))
		}
	}
}

// findNodeProjectRoot walks up from the cwd looking for a Node project marker,
// returning the directory that contains it. The nearest marker wins.
func findNodeProjectRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		for _, marker := range nodeMarkers {
			if _, err := os.Lstat(filepath.Join(dir, marker)); err == nil {
				return dir, true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false // reached the filesystem root
		}
		dir = parent
	}
}

// readRequestedNodeVersion resolves the version the project asks for, trying the
// cheapest and most explicit sources first. Returns "" when none declare one.
//
// Order: .nvmrc and .node-version are dedicated version files (most explicit);
// package.json's engines.node is a constraint (e.g. ">=18") used as a fallback.
func readRequestedNodeVersion(root string) string {
	for _, f := range []string{".nvmrc", ".node-version"} {
		if v := readVersionFile(filepath.Join(root, f)); v != "" {
			return v
		}
	}
	return readEnginesNode(filepath.Join(root, "package.json"))
}

// readVersionFile reads a single-line version file, stripping a leading "v".
func readVersionFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
}

// readEnginesNode extracts engines.node from package.json. We decode only the
// field we need rather than the whole file's shape.
func readEnginesNode(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal(b, &pkg); err != nil {
		return "" // malformed package.json: just omit the version
	}
	return strings.TrimSpace(pkg.Engines.Node)
}
