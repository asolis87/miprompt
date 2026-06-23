package main

import (
	"os"
	"testing"
)

// TestMain pins the color profile to truecolor for the whole test suite, so
// tests never pass or fail by luck of the host's $COLORTERM. Tests that need a
// different profile set it locally via withProfile.
func TestMain(m *testing.M) {
	activeProfile = profileTrueColor
	os.Exit(m.Run())
}
