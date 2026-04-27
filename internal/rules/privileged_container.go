package rules

import "github.com/lowplane/sevro/internal/parser"

// privilegedContainer fires when securityContext.privileged is true.
// Privileged containers bypass nearly every isolation control the
// kernel provides; they are appropriate only for storage drivers and
// node-level agents and should never appear in application charts.
type privilegedContainer struct{}

func newPrivilegedContainer() Detector { return privilegedContainer{} }

func (privilegedContainer) ID() string   { return "privileged-container" }
func (privilegedContainer) Name() string { return "Privileged container" }

func (privilegedContainer) Run(w parser.Workload) []Finding {
	if w.Security.Privileged == nil || !*w.Security.Privileged {
		return nil
	}
	return []Finding{{
		DetectorID: "privileged-container",
		Workload:   w.Name,
		Title:      "Container declared privileged",
		Detail:     "securityContext.privileged=true bypasses every Linux capability check. Privileged is correct only for node-level agents (CSI drivers, node exporters); application workloads should set privileged=false and request the specific capabilities they actually need.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
