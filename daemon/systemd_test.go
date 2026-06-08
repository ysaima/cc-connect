//go:build linux

package daemon

import (
	"strings"
	"testing"
)

func TestEscapeSystemdEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Simple path without special characters.
		{"plain", "/usr/bin:/usr/local/bin", "/usr/bin:/usr/local/bin"},
		// Path with spaces (WSL Windows paths). Spaces inside the
		// double-quoted systemd value do not need escaping.
		{"spaces", "/mnt/c/Program Files/Git/cmd", "/mnt/c/Program Files/Git/cmd"},
		// Path with a Windows-style backslash; each backslash must
		// be doubled so systemd does not treat it as the start of an
		// escape sequence.
		{"backslash", `C:\Windows\System32`, `C:\\Windows\\System32`},
		// Path with a double quote: must be escaped to keep the
		// surrounding systemd quoting intact.
		{"quote", `/path/with"quote`, `/path/with\"quote`},
		// Combined: spaces and a Windows-style backslash.
		{"spaces+backslash", `/mnt/c/Program Files\Git`, `/mnt/c/Program Files\\Git`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeSystemdEnvValue(tt.input)
			if got != tt.expected {
				t.Errorf("escapeSystemdEnvValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildUnit_PathWithSpaces(t *testing.T) {
	cfg := Config{
		BinaryPath: "/usr/local/bin/cc-connect",
		WorkDir:    "/home/user",
		LogFile:    "/tmp/cc-connect.log",
		LogMaxSize: 10485760,
		EnvPATH:    "/usr/bin:/mnt/c/Program Files/Git/cmd:/mnt/c/Program Files/nodejs",
	}

	m := &systemdManager{system: false}
	unit := m.buildUnit(cfg)

	// PATH should be wrapped in double quotes.
	if !strings.Contains(unit, `Environment="PATH=`) {
		t.Error("PATH should be wrapped in double quotes")
	}

	// The Windows path with spaces must be preserved verbatim.
	if !strings.Contains(unit, "/mnt/c/Program Files/Git/cmd") {
		t.Error("PATH should contain the Windows path with spaces")
	}

	// The unquoted variant must not be present.
	if strings.Contains(unit, "Environment=PATH=/usr/bin") {
		t.Error("PATH should not be unquoted")
	}
}

func TestBuildUnit_EscapesBackslashes(t *testing.T) {
	cfg := Config{
		BinaryPath: "/usr/local/bin/cc-connect",
		WorkDir:    "/home/user",
		LogFile:    "/tmp/cc-connect.log",
		LogMaxSize: 10485760,
		EnvPATH:    `/usr/bin:/path\with\backslash`,
	}

	m := &systemdManager{system: false}
	unit := m.buildUnit(cfg)

	// Backslashes must be doubled inside the quoted PATH value.
	// The full line is: Environment="PATH=/usr/bin:/path\\with\\backslash"
	want := `Environment="PATH=/usr/bin:/path\\with\\backslash"`
	if !strings.Contains(unit, want) {
		t.Errorf("Backslashes should be escaped in PATH line; want substring %q, got:\n%s", want, unit)
	}
}
