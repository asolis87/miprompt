package main

import (
	"strings"
	"testing"
)

// TestSafeRenderRecoversFromPanic verifies the cardinal rule: even if rendering
// panics, the user still gets a usable prompt — never a blank line or a stack
// trace. We inject a panicking render implementation and confirm the fallback.
func TestSafeRenderRecoversFromPanic(t *testing.T) {
	orig := renderImpl
	defer func() { renderImpl = orig }() // restore for other tests

	renderImpl = func(Shell, int, int, expensiveMode, string, Config) string {
		panic("boom: simulated segment crash")
	}

	out := safeRenderPrompt(ShellZsh, 0, 0, expensiveNone, "", defaultConfig())
	if out != fallbackPrompt {
		t.Errorf("panic should yield the fallback prompt %q, got %q", fallbackPrompt, out)
	}
	if out == "" {
		t.Error("recovery must never produce an empty prompt")
	}
}

// TestSafeRenderPassesThroughNormally confirms the net is transparent on the
// happy path: no panic, the real render result flows through unchanged.
func TestSafeRenderPassesThroughNormally(t *testing.T) {
	out := safeRenderPrompt(ShellZsh, 0, 0, expensiveNone, "", defaultConfig())
	if out == fallbackPrompt {
		t.Error("normal render should not return the fallback")
	}
	if !strings.Contains(out, "❯") {
		t.Errorf("normal render should contain the prompt symbol: %q", out)
	}
}
