package analyze

import (
	"strings"

	"github.com/lowplane/sevro/internal/render"
	"github.com/lowplane/sevro/internal/rules"
)

// FilterOptions narrows a Report's findings before rendering.
//
//   - MinSeverity drops findings whose severity is below the threshold.
//   - DetectorIDs (when non-empty) keeps only findings emitted by the
//     listed detectors.
//   - SecurityOnly keeps only the security-class detectors that the
//     `sevro audit` command surfaces.
type FilterOptions struct {
	MinSeverity  rules.Severity
	DetectorIDs  []string
	SecurityOnly bool
}

// SecurityDetectorIDs is the closed set the audit command considers
// "security-class". New security detectors must be added here so the
// audit command picks them up.
var SecurityDetectorIDs = map[string]bool{
	"missing-memory-limit": true,
	"missing-cpu-limit":    true,
	"image-pinned-latest":  true,
}

// Filter applies the options to a Report's findings and returns a
// new Report. The original Report is not mutated.
func Filter(r render.Report, opts FilterOptions) render.Report {
	if opts.MinSeverity == "" && len(opts.DetectorIDs) == 0 && !opts.SecurityOnly {
		return r
	}
	allowed := map[string]bool{}
	for _, d := range opts.DetectorIDs {
		allowed[strings.TrimSpace(d)] = true
	}
	out := make([]rules.Finding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if opts.SecurityOnly && !SecurityDetectorIDs[f.DetectorID] {
			continue
		}
		if len(allowed) > 0 && !allowed[f.DetectorID] {
			continue
		}
		if opts.MinSeverity != "" && severityRank(f.Severity) < severityRank(opts.MinSeverity) {
			continue
		}
		out = append(out, f)
	}
	r.Findings = out
	return r
}

func severityRank(s rules.Severity) int {
	switch s {
	case rules.SeverityHigh:
		return 3
	case rules.SeverityMed:
		return 2
	case rules.SeverityLow:
		return 1
	case rules.SeverityInfo:
		return 0
	}
	return 0
}
