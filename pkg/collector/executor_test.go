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

// FakeFileSystem returns pre-loaded content for files and directories.
// Used in tests to avoid reading real /proc and /sys files.
type FakeFileSystem struct {
	Files  map[string]string   // path → file content
	Dirs   map[string][]string // path → directory entries
	Called []string            // record of accessed paths
}

// ReadFile returns the canned content for the given path, or an error.
func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	f.Called = append(f.Called, path)
	data, ok := f.Files[path]
	if !ok {
		return nil, fmt.Errorf("FakeFileSystem: file not found: %s", path)
	}
	return []byte(data), nil
}

// ReadDir returns the canned directory entries for the given path.
func (f *FakeFileSystem) ReadDir(path string) ([]string, error) {
	f.Called = append(f.Called, path)
	entries, ok := f.Dirs[path]
	if !ok {
		return nil, fmt.Errorf("FakeFileSystem: dir not found: %s", path)
	}
	return entries, nil
}

// Reset clears all recorded calls, files, and dirs.
func (f *FakeFileSystem) Reset() {
	f.Called = nil
	f.Files = nil
	f.Dirs = nil
}

// AddFile is a convenience method to register a file.
func (f *FakeFileSystem) AddFile(path, content string) {
	if f.Files == nil {
		f.Files = make(map[string]string)
	}
	f.Files[path] = content
}

// AddDir is a convenience method to register a directory.
func (f *FakeFileSystem) AddDir(path string, entries []string) {
	if f.Dirs == nil {
		f.Dirs = make(map[string][]string)
	}
	f.Dirs[path] = entries
}
