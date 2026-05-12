// Package collector provides system inventory collection.
package collector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner defines the interface for executing external commands.
//
// Production: OSCommandRunner uses os/exec.CommandContext for real execution.
// Testing: a test double returns pre-recorded output from test fixtures.
//
// This enables dependency injection and consistent timeout handling
// across all three supported platforms (Linux, macOS, Windows).
type CommandRunner interface {
	// Run executes a command with the given context and returns stdout.
	// The context controls cancellation and timeout.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// OSCommandRunner is the production implementation that delegates to
// os/exec.CommandContext.
type OSCommandRunner struct{}

// Run executes the command via os/exec.CommandContext.
func (OSCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Predefined timeout durations for external commands.
//
// Use CmdTimeoutFast for simple local queries that complete almost instantly
// (hostname, uname, sysctl reads). Use CmdTimeoutMedium for moderately
// expensive commands (lspci, ip, ifconfig). Use CmdTimeoutSlow for long-
// running commands that can hang (system_profiler, PowerShell/WMI queries).
const (
	CmdTimeoutFast   = 5 * time.Second
	CmdTimeoutMedium = 15 * time.Second
	CmdTimeoutSlow   = 30 * time.Second
)

// PartialErrors collects non-fatal errors during inventory collection.
// Used across all platforms (Linux, macOS, Windows) for consistent
// partial error reporting.
type PartialErrors []string

// Add appends a formatted error message.
func (pe *PartialErrors) Add(format string, args ...interface{}) {
	*pe = append(*pe, fmt.Sprintf(format, args...))
}

// Err returns a combined error if any were collected, nil otherwise.
func (pe PartialErrors) Err() error {
	if len(pe) == 0 {
		return nil
	}
	return fmt.Errorf("errores parciales (%d): %s", len(pe), strings.Join(pe, "; "))
}
