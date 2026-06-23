package main

import (
	"os"
	"path/filepath"
)

// pythonInfo holds the active Python environment for the prompt. A nil
// *pythonInfo means "no environment active" and the segment is hidden.
type pythonInfo struct {
	env string // the active environment name (venv dir name or conda env name)
}

// readPythonInfo returns the active Python environment, or nil when none is.
//
// This is the cheapest possible source of prompt data: the info already lives
// in THIS process's environment, inherited from the shell. Activating a venv
// exports $VIRTUAL_ENV; conda exports $CONDA_DEFAULT_ENV. No filesystem walk, no
// fork — just an env lookup. We surface the environment NAME, not the full path.
func readPythonInfo() *pythonInfo {
	// venv / virtualenv: $VIRTUAL_ENV is the env directory; its base name is the
	// conventional display name (e.g. ".../project/venv" -> "venv").
	if ve := os.Getenv("VIRTUAL_ENV"); ve != "" {
		return &pythonInfo{env: filepath.Base(ve)}
	}

	// conda: $CONDA_DEFAULT_ENV is already just the env name.
	if ce := os.Getenv("CONDA_DEFAULT_ENV"); ce != "" {
		return &pythonInfo{env: ce}
	}

	return nil // no active Python environment
}

// segmentPython describes the active-Python-environment segment. Icon and color
// come from config, falling back to built-in defaults.
func segmentPython(py *pythonInfo, cfg PythonConfig) Segment {
	label := py.env
	if cfg.Icon != "" {
		label = cfg.Icon + " " + label
	}
	fg := orDefault(cfg.Color, "cyan")
	return Segment{text: label, fg: fg, bg: orDefault(cfg.Bg, fg)}
}
