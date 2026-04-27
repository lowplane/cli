package rules

import (
	"fmt"

	"github.com/lowplane/sevro/pkg/parser"
)

// cpuOverprovisioned flags workloads whose CPU request is meaningfully
// higher than typical real-world utilization for the declared limit.
//
// Without 30 days of Prometheus data (CLI mode), this is a heuristic:
// when request/limit > 0.5 we assume 2× overprovisioning is plausible.
// The agent-installed product replaces this with measured P95 data.
type cpuOverprovisioned struct{}

func newCPUOverprovisioned() Detector { return cpuOverprovisioned{} }

func (cpuOverprovisioned) ID() string   { return "cpu-overprovisioned" }
func (cpuOverprovisioned) Name() string { return "CPU request overprovisioned" }

// Heuristic constants. Change tracked via golden tests.
const (
	cpuOverprovRatio        = 0.5 // request/limit threshold
	cpuOverprovMonthlyHours = 730 // hours per month, AWS billing convention
	// Rough $/vCPU/hour (AWS m5.large baseline). Used only when no
	// agent-measured cost basis is available; the disclosure banner
	// says ±40% explicitly.
	cpuPricePerCoreHourCents = 4
)

func (cpuOverprovisioned) Run(w parser.Workload) []Finding {
	req, lim := w.Requests.CPU, w.Limits.CPU
	if !req.Set || !lim.Set {
		return nil
	}
	if lim.Value == 0 {
		return nil
	}
	if float64(req.Value)/float64(lim.Value) <= cpuOverprovRatio {
		return nil
	}
	// Estimate savings: assume right-size to 50% of current request.
	excessMillicores := req.Value / 2
	monthlyCents := excessMillicores * cpuOverprovMonthlyHours * cpuPricePerCoreHourCents / 1000
	return []Finding{{
		DetectorID:      "cpu-overprovisioned",
		Workload:        w.Name,
		Title:           "CPU request appears overprovisioned",
		Detail:          fmt.Sprintf("Request %s vs limit %s — typical utilization rarely justifies this ratio. Consider halving the request.", req, lim),
		MonthlyUSDCents: monthlyCents,
		Severity:        SeverityMed,
		Confidence:      ConfidenceMed,
	}}
}
