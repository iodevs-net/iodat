package collector

import (
	"context"
	"fmt"
	"strings"
)

// FakeCommandRunner records command invocations and returns canned responses.
// Used in tests to avoid executing real OS commands.
//
// Responses are keyed by "name arg1 arg2 ...". Calls records every invocation
// in order for assertion.
type FakeCommandRunner struct {
	Responses map[string]string
	Calls     []string
}

// Run returns the canned response for the given command, or an error if no
// response is registered.
func (f *FakeCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := name + " " + strings.Join(args, " ")
	f.Calls = append(f.Calls, key)
	if resp, ok := f.Responses[key]; ok {
		return []byte(resp), nil
	}
	return nil, fmt.Errorf("FakeCommandRunner: no response for %q", key)
}

// Reset clears all recorded calls and responses.
func (f *FakeCommandRunner) Reset() {
	f.Calls = nil
	f.Responses = nil
}
