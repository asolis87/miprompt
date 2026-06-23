package main

import (
	"strings"
	"testing"
)

func TestReadPythonInfoNone(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	if py := readPythonInfo(); py != nil {
		t.Errorf("no active env should return nil, got %+v", py)
	}
}

func TestReadPythonInfoVenv(t *testing.T) {
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", "/home/me/project/.venv")
	py := readPythonInfo()
	if py == nil {
		t.Fatal("active VIRTUAL_ENV should return info, got nil")
	}
	// We show the base name, not the full path.
	if py.env != ".venv" {
		t.Errorf("env = %q, want base name \".venv\"", py.env)
	}
}

func TestReadPythonInfoConda(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_DEFAULT_ENV", "ml")
	py := readPythonInfo()
	if py == nil || py.env != "ml" {
		t.Fatalf("conda env should surface as \"ml\", got %+v", py)
	}
}

func TestReadPythonInfoVenvWinsOverConda(t *testing.T) {
	// When both are set, venv (the more specific) takes precedence.
	t.Setenv("VIRTUAL_ENV", "/x/venv")
	t.Setenv("CONDA_DEFAULT_ENV", "ml")
	if py := readPythonInfo(); py == nil || py.env != "venv" {
		t.Fatalf("venv should win over conda, got %+v", py)
	}
}

func TestSegmentPythonIncludesEnv(t *testing.T) {
	seg := segmentPython(&pythonInfo{env: "myenv"}, PythonConfig{Color: "cyan"})
	if !strings.Contains(seg.text, "myenv") {
		t.Errorf("python segment should show env name: %q", seg.text)
	}
}
