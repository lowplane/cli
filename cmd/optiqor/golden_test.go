package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -update regenerates the golden files from current output. Use
// sparingly: only after a deliberate UX change. The test diffs against
// the recorded output otherwise.
var update = flag.Bool("update", false, "update golden files")

// goldenDir holds the recorded outputs for stability tests. Adding a
// new test case is one fixture and one test entry — no hand-curating
// of expected strings.
const goldenDir = "../../testdata/golden"

type goldenCase struct {
	name string
	args []string
}

var goldenCases = []goldenCase{
	{name: "demo_plain", args: []string{"--no-color", "demo"}},
	{name: "demo_json", args: []string{"demo", "--json"}},
	{name: "analyze_fixture_plain", args: []string{"--no-color", "analyze", "../../testdata/fixtures/basic-chart/values.yaml"}},
	{name: "analyze_fixture_severity_high", args: []string{"--no-color", "analyze", "../../testdata/fixtures/basic-chart/values.yaml", "--severity", "high"}},
	{name: "analyze_fixture_detector_filter", args: []string{"--no-color", "analyze", "../../testdata/fixtures/basic-chart/values.yaml", "--detector", "image-pinned-latest"}},
	{name: "score_fixture_plain", args: []string{"--no-color", "score", "../../testdata/fixtures/basic-chart/values.yaml"}},
	{name: "score_fixture_json", args: []string{"score", "../../testdata/fixtures/basic-chart/values.yaml", "--json"}},
	{name: "audit_fixture_plain", args: []string{"--no-color", "audit", "../../testdata/fixtures/basic-chart/values.yaml", "--fail-on", ""}},
}

func TestGolden(t *testing.T) {
	if err := os.MkdirAll(goldenDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pin terminal width so the boxed-card layout is deterministic
	// across machines. Without this the developer's $COLUMNS leaks
	// into golden output and CI (no TTY → fallback 80) diverges.
	t.Setenv("COLUMNS", "80")

	for _, tc := range goldenCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tc.args)
			// Tests must run in the cmd/optiqor directory so the
			// "../../testdata/..." paths resolve.
			_ = cmd.Execute()
			got := normalize(buf.String())

			path := filepath.Join(goldenDir, tc.name+".txt")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s (run with -update to create): %v", path, err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- want\n%s\n--- got\n%s", tc.name, want, got)
			}
		})
	}
}

// normalize replaces filesystem-dependent paths with stable placeholders
// so the golden file is portable across machines and CI runners.
func normalize(s string) string {
	cwd, err := os.Getwd()
	if err == nil {
		s = strings.ReplaceAll(s, cwd, "<CWD>")
	}
	return s
}
