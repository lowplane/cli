// Package analyze orchestrates one Sevro CLI run: parser → rules → render.
//
// Callers (cmd/sevro/main.go, tests, future SDK consumers) hand in a
// values reader and an Options struct, get back a render.Report.
package analyze

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/pkg/parser"
	"github.com/lowplane/sevro/pkg/rules"
)

// Options controls a single analysis run.
type Options struct {
	// Source identifies what was analysed (a file path or "stdin").
	// Surfaced verbatim in the report header.
	Source string

	// Detectors is the set to apply. nil → rules.All().
	Detectors []rules.Detector
}

// Run reads a Helm values document from r and returns the populated
// report. Errors come from YAML parse failures or IO.
func Run(r io.Reader, opts Options) (render.Report, error) {
	wls, err := parser.ParseValues(r)
	if err != nil {
		return render.Report{}, err
	}
	dets := opts.Detectors
	if dets == nil {
		dets = rules.All()
	}
	return render.Report{
		Source:    opts.Source,
		Workloads: len(wls),
		Findings:  rules.Run(wls, dets),
	}, nil
}

// RunPath is the convenience entrypoint used by the `analyze`
// subcommand. It accepts either a file or a directory; for a directory
// it reads `values.yaml` at the root.
func RunPath(path string) (render.Report, error) {
	info, err := os.Stat(path)
	if err != nil {
		return render.Report{}, fmt.Errorf("analyze: stat %s: %w", path, err)
	}
	target := path
	if info.IsDir() {
		target = filepath.Join(path, "values.yaml")
	}
	f, err := os.Open(target) //nolint:gosec // user-specified analysis input
	if err != nil {
		return render.Report{}, fmt.Errorf("analyze: open %s: %w", target, err)
	}
	defer func() { _ = f.Close() }()
	return Run(f, Options{Source: target})
}
