package main

import (
	"strings"
	"testing"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct{ ms int; want string }{
		{0, "0ms"},
		{42, "42ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{12300, "12.3s"},
		{59900, "59.9s"},
		{60000, "1m0s"},
		{90000, "1m30s"},
		{3599000, "59m59s"}, // just under an hour
		{3600000, "1h0m"},
		{3725000, "1h2m"},
		{-5, "0ms"}, // defensive: negative clamps to 0
	}
	for _, c := range cases {
		if got := formatDuration(c.ms); got != c.want {
			t.Errorf("formatDuration(%d) = %q, want %q", c.ms, got, c.want)
		}
	}
}

func TestSegmentDurationBelowThresholdHidden(t *testing.T) {
	// Default threshold 2000ms: a quick command shows nothing.
	if seg := segmentDuration(500, DurationConfig{}); seg != nil {
		t.Errorf("500ms under default 2s threshold should be hidden, got %+v", seg)
	}
}

func TestSegmentDurationAboveThresholdShown(t *testing.T) {
	seg := segmentDuration(3000, DurationConfig{Color: "yellow"})
	if seg == nil {
		t.Fatal("3000ms over default 2s threshold should be shown, got nil")
	}
	if !strings.Contains(seg.text, "3.0s") {
		t.Errorf("expected humanized 3.0s, got %q", seg.text)
	}
}

func TestSegmentDurationCustomThreshold(t *testing.T) {
	// With a 100ms threshold, a 500ms command shows.
	if seg := segmentDuration(500, DurationConfig{MinMs: 100}); seg == nil {
		t.Error("500ms over custom 100ms threshold should be shown, got nil")
	}
	// And a 50ms one still hides.
	if seg := segmentDuration(50, DurationConfig{MinMs: 100}); seg != nil {
		t.Errorf("50ms under 100ms threshold should be hidden, got %+v", seg)
	}
}
