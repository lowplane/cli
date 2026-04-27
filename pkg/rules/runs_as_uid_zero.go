package rules

import "github.com/lowplane/sevro/pkg/parser"

// runsAsUIDZero fires when runAsUser is explicitly 0 (root). Distinct
// from `run-as-root` (which fires on `runAsNonRoot=false`) — this one
// looks for an explicit `runAsUser: 0` declaration which is more
// load-bearing because it overrides the image's USER directive even
// when the image is built non-root.
//
// CIS Kubernetes Benchmark 5.2.6.
type runsAsUIDZero struct{}

func newRunsAsUIDZero() Detector { return runsAsUIDZero{} }

func (runsAsUIDZero) ID() string   { return "runs-as-uid-zero" }
func (runsAsUIDZero) Name() string { return "runAsUser is 0 (root)" }

func (runsAsUIDZero) Run(w parser.Workload) []Finding {
	if w.Security.RunAsUser == nil || *w.Security.RunAsUser != 0 {
		return nil
	}
	return []Finding{{
		DetectorID: "runs-as-uid-zero",
		Workload:   w.Name,
		Title:      "runAsUser explicitly set to 0",
		Detail:     "securityContext.runAsUser is set to 0. This forces the container to run as UID 0 (root) regardless of the image's USER directive — a hard override that's almost always wrong for application workloads. Set runAsUser to a non-zero UID (1000+) and runAsNonRoot=true.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
