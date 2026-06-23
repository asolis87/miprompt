package main

import "fmt"

// formatDuration turns a duration in milliseconds into a compact, human-readable
// string: "850ms", "1.5s", "2m30s", "1h2m". Raw milliseconds ("5400000ms") tell
// you nothing at a glance; humanized units do. This is pure logic, fully tested.
//
// Rules:
//   - under 1s            → whole milliseconds ("850ms")
//   - under 1m            → seconds with one decimal ("1.5s", "12.0s")
//   - under 1h            → minutes + whole seconds ("2m30s")
//   - 1h and over         → hours + minutes ("1h2m")
func formatDuration(ms int) string {
	if ms < 0 {
		ms = 0
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}

	totalSec := ms / 1000
	if totalSec < 60 {
		// One decimal of seconds, e.g. 1500ms -> "1.5s".
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}

	if totalSec < 3600 {
		m := totalSec / 60
		s := totalSec % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}

	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	return fmt.Sprintf("%dh%dm", h, m)
}

// segmentDuration describes the last-command-duration segment, shown only when
// the duration meets the configured threshold (quiet for quick commands). A nil
// result means "don't show this segment".
func segmentDuration(ms int, cfg DurationConfig) *Segment {
	threshold := cfg.MinMs
	if threshold <= 0 {
		threshold = 2000 // default: only commands slower than 2s
	}
	if ms < threshold {
		return nil
	}

	label := formatDuration(ms)
	if cfg.Icon != "" {
		label = cfg.Icon + " " + label
	}
	fg := orDefault(cfg.Color, "yellow")
	return &Segment{text: label, fg: fg, bg: orDefault(cfg.Bg, fg)}
}
