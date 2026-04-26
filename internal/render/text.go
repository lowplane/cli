// Package render formats analysis results as plain text or JSON.
//
// **Every renderer must include the ±40% accuracy disclosure.** Removing
// it is a hard rule violation — see ../../CLAUDE.md. The disclosure is
// what makes the CLI a trustworthy funnel: we never overpromise.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/lowplane/sevro/internal/rules"
)

// AccuracyDisclosure is the mandatory line every output must contain.
const AccuracyDisclosure = "Sandbox accuracy: ±40%. Install the Sevro agent for exact numbers (sevro.dev/get)."

// Report is the renderer-facing view of an analysis run.
type Report struct {
	Source   string           `json:"source"`             // path or label of the input
	Workloads int             `json:"workloads_analyzed"`
	Findings []rules.Finding `json:"findings"`
}

// MonthlySavingsUSDCents totals the predicted savings across findings.
func (r Report) MonthlySavingsUSDCents() int64 {
	var sum int64
	for _, f := range r.Findings {
		sum += f.MonthlyUSDCents
	}
	return sum
}

// Text writes a human-readable ASCII report. Always includes the
// accuracy disclosure.
func Text(w io.Writer, r Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Sevro Sandbox Analysis  ─────────────────  sevro.dev\n")
	fmt.Fprintf(&b, "source: %s\n\n", r.Source)

	if len(r.Findings) == 0 {
		fmt.Fprintf(&b, "No findings across %d workload(s).\n\n", r.Workloads)
		fmt.Fprintf(&b, "%s\n", AccuracyDisclosure)
		_, err := io.WriteString(w, b.String())
		return err
	}

	fmt.Fprintf(&b, "Findings (%d)\n", len(r.Findings))
	for _, f := range r.Findings {
		savings := ""
		if f.MonthlyUSDCents > 0 {
			savings = fmt.Sprintf("save ~$%s/mo  ", formatCents(f.MonthlyUSDCents))
		}
		fmt.Fprintf(&b, "  [%s]  %-32s  %sworkload: %s  confidence: %s\n",
			f.Severity, f.Title, savings, f.Workload, f.Confidence)
	}

	total := r.MonthlySavingsUSDCents()
	if total > 0 {
		fmt.Fprintf(&b, "\nEstimated monthly savings: $%s\n", formatCents(total))
	}

	fmt.Fprintf(&b, "\n%s\n", AccuracyDisclosure)
	_, err := io.WriteString(w, b.String())
	return err
}

// JSON writes the report as machine-readable JSON. The disclosure rides
// as a top-level field so consumers cannot accidentally render numbers
// without context.
func JSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string  `json:"accuracy_disclosure"`
		Source             string  `json:"source"`
		Workloads          int     `json:"workloads_analyzed"`
		Findings           []rules.Finding `json:"findings"`
		MonthlySavingsUSD  float64 `json:"monthly_savings_usd"`
	}{
		AccuracyDisclosure: AccuracyDisclosure,
		Source:             r.Source,
		Workloads:          r.Workloads,
		Findings:           r.Findings,
		MonthlySavingsUSD:  float64(r.MonthlySavingsUSDCents()) / 100.0,
	})
}

func formatCents(c int64) string {
	dollars := c / 100
	cents := c % 100
	if cents == 0 {
		return fmt.Sprintf("%d", dollars)
	}
	return fmt.Sprintf("%d.%02d", dollars, cents)
}
