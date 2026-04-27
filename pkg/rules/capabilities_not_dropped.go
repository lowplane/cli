package rules

import "github.com/lowplane/sevro/pkg/parser"

// capabilitiesNotDroppedAll fires when capabilities.drop does not
// include "ALL". CIS Kubernetes Benchmark 5.2.8 ("Minimize the
// admission of containers with capabilities assigned") and the NSA
// hardening guide both ask for `drop: [ALL]` as the baseline, with
// `add: [...]` re-granting only what the workload actually needs.
//
// We treat the absence of `drop: [ALL]` (or equivalent) as MED — most
// charts that haven't done this haven't deliberately rejected the
// hardening, they just haven't applied it yet.
type capabilitiesNotDroppedAll struct{}

func newCapabilitiesNotDroppedAll() Detector { return capabilitiesNotDroppedAll{} }

func (capabilitiesNotDroppedAll) ID() string   { return "capabilities-not-dropped-all" }
func (capabilitiesNotDroppedAll) Name() string { return "capabilities.drop does not include ALL" }

func (capabilitiesNotDroppedAll) Run(w parser.Workload) []Finding {
	for _, c := range w.Security.CapabilitiesDrop {
		if normaliseCap(c) == "ALL" {
			return nil
		}
	}
	return []Finding{{
		DetectorID: "capabilities-not-dropped-all",
		Workload:   w.Name,
		Title:      "capabilities.drop does not include ALL",
		Detail:     "Best-practice baseline (CIS 5.2.8) is securityContext.capabilities.drop: [ALL] with add: [...] re-granting only the specific capabilities required. The current chart does not drop ALL, so the container inherits the default capability set including CHOWN, SETUID, SETGID, NET_BIND_SERVICE, etc.",
		Severity:   SeverityMed,
		Confidence: ConfidenceHigh,
	}}
}
