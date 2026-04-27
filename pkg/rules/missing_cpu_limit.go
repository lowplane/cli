package rules

import "github.com/lowplane/sevro/pkg/parser"

// missingCPULimit fires when a workload sets a CPU request but no
// limit. Modern guidance (Tim Hockin et al) is to actually omit CPU
// limits in many cases — but the CLI's sandbox-grade view cannot
// tell whether a workload is throttling-tolerant. We surface this as
// LOW severity informational so platform teams can decide.
type missingCPULimit struct{}

func newMissingCPULimit() Detector { return missingCPULimit{} }

func (missingCPULimit) ID() string   { return "missing-cpu-limit" }
func (missingCPULimit) Name() string { return "Missing CPU limit" }

func (missingCPULimit) Run(w parser.Workload) []Finding {
	if !w.Requests.CPU.Set || w.Limits.CPU.Set {
		return nil
	}
	return []Finding{{
		DetectorID: "missing-cpu-limit",
		Workload:   w.Name,
		Title:      "CPU limit not set",
		Detail:     "This workload declares a CPU request but no limit. Some teams omit CPU limits intentionally to avoid throttling; others expect predictable scheduling. Confirm this is the intended posture.",
		Severity:   SeverityLow,
		Confidence: ConfidenceMed,
	}}
}
