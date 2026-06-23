package main

import (
	"strings"
	"testing"
)

// gitCfg returns a GitConfig with no icon (so tests assert on the branch/counts
// text without a Nerd Font glyph in the way) and the default ahead/behind icons.
func gitCfg() GitConfig {
	return GitConfig{Color: "yellow", AheadIcon: "↑", BehindIcon: "↓"}
}

func TestSegmentGitUpToDateHasNoCounts(t *testing.T) {
	// Up to date with upstream: branch only, no ahead/behind markers.
	seg := segmentGit(&gitInfo{branch: "main", hasUpstream: true}, gitCfg())
	if strings.ContainsAny(seg.text, "↑↓") {
		t.Errorf("up-to-date branch should show no ahead/behind: %q", seg.text)
	}
}

func TestSegmentGitAhead(t *testing.T) {
	seg := segmentGit(&gitInfo{branch: "main", hasUpstream: true, ahead: 2}, gitCfg())
	if !strings.Contains(seg.text, "↑2") {
		t.Errorf("expected ahead marker ↑2, got: %q", seg.text)
	}
	if strings.Contains(seg.text, "↓") {
		t.Errorf("ahead-only should not show behind marker: %q", seg.text)
	}
}

func TestSegmentGitDiverged(t *testing.T) {
	seg := segmentGit(&gitInfo{branch: "main", hasUpstream: true, ahead: 2, behind: 3}, gitCfg())
	if !strings.Contains(seg.text, "↑2") || !strings.Contains(seg.text, "↓3") {
		t.Errorf("diverged should show both ↑2 and ↓3, got: %q", seg.text)
	}
}

func TestSegmentGitNoUpstreamHidesCounts(t *testing.T) {
	// No upstream: even with nonzero counts (shouldn't happen, but be defensive),
	// no markers are shown — "not applicable" is distinct from "0/0".
	seg := segmentGit(&gitInfo{branch: "feature", hasUpstream: false, ahead: 5}, gitCfg())
	if strings.ContainsAny(seg.text, "↑↓") {
		t.Errorf("no-upstream branch must not show ahead/behind: %q", seg.text)
	}
	if !strings.Contains(seg.text, "feature") {
		t.Errorf("branch name missing: %q", seg.text)
	}
}

func TestSegmentGitDirtyRecolors(t *testing.T) {
	clean := segmentGit(&gitInfo{branch: "main", hasUpstream: true}, gitCfg())
	dirty := segmentGit(&gitInfo{branch: "main", hasUpstream: true, dirty: true}, gitCfg())
	if clean.fg == dirty.fg {
		t.Errorf("dirty tree should recolor the segment (clean fg %q == dirty fg %q)", clean.fg, dirty.fg)
	}
	if !strings.Contains(dirty.text, "●") {
		t.Errorf("dirty segment should include the ● marker: %q", dirty.text)
	}
}
