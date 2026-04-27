// Package render formats analysis results as styled text or JSON.
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

	"github.com/lowplane/sevro/internal/render/style"
	"github.com/lowplane/sevro/pkg/rules"
)

// AccuracyDisclosure is the mandatory line every output must contain.
const AccuracyDisclosure = "Sandbox accuracy: ±40%. Install the Sevro agent for exact numbers (sevro.dev/get)."

// Brand strings used in the header banner.
const (
	BrandName    = "sevro"
	BrandTagline = "Helm chart cost & security analysis"
	GetURL       = "https://sevro.dev/get"
)

// Report is the renderer-facing view of an analysis run.
type Report struct {
	Source    string          `json:"source"`             // path or label of the input
	Workloads int             `json:"workloads_analyzed"`
	Findings  []rules.Finding `json:"findings"`
}

// Options controls how a Report is rendered. Callers (cmd/sevro/main.go)
// detect TTY + NO_COLOR + --no-color and set Color accordingly.
type Options struct {
	Color bool // false → plain ASCII, no ANSI; true → branded styled output
	Width int  // terminal width; 0 → default 72
}

// MonthlySavingsUSDCents totals the predicted savings across findings.
func (r Report) MonthlySavingsUSDCents() int64 {
	var sum int64
	for _, f := range r.Findings {
		sum += f.MonthlyUSDCents
	}
	return sum
}

// Text writes the styled human-readable report.
//
// Layout:
//
//	╭─ sevro ────────────  v ───────╮
//	│ <tagline>                     │
//	╰───────────────────────────────╯
//	source: …
//	N workloads · M findings
//
//	[ HIGH ] worker
//	  Memory limit not set
//	  <detail>                      ●●● high
//
//	[ MED  ] api          save ~$X /mo
//	  CPU request appears overprovisioned
//	  <detail>                      ●● medium
//
//	───────────────────────────────
//	estimated monthly savings: $X (±40%)
//	→ install agent for exact numbers: sevro.dev/get
func Text(w io.Writer, r Report, opts Options) error {
	t := style.NewTheme(opts.Color)
	width := opts.Width
	if width <= 0 {
		width = 72
	}

	var b strings.Builder

	// Header
	writeHeader(&b, t, width)

	// Source + counts
	srcLabel := r.Source
	if srcLabel == "" {
		srcLabel = "(stdin)"
	}
	fmt.Fprintf(&b, "  %s %s\n",
		t.Muted.Render("source:"),
		t.Workload.Render(srcLabel),
	)
	fmt.Fprintf(&b, "  %s %s%s%s\n\n",
		t.Muted.Render(plural(r.Workloads, "workload", "workloads")),
		t.Muted.Render("· "),
		t.Title.Render(plural(len(r.Findings), "finding", "findings")),
		"",
	)

	// Empty state — celebrate.
	if len(r.Findings) == 0 {
		fmt.Fprintf(&b, "  %s\n\n", t.OK.Render("✓ Clean. No findings."))
		writeFooter(&b, t, width, 0)
		_, err := io.WriteString(w, b.String())
		return err
	}

	// Findings — one block per finding, blank line between.
	totalSavings := r.MonthlySavingsUSDCents()
	for i, f := range r.Findings {
		writeFinding(&b, t, f, width)
		if i < len(r.Findings)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	writeFooter(&b, t, width, totalSavings)
	_, err := io.WriteString(w, b.String())
	return err
}

func writeHeader(b *strings.Builder, t style.Theme, width int) {
	brand := t.Brand.Render(BrandName)
	tagline := t.Tagline.Render(BrandTagline)
	div := t.DividerLine(width)
	fmt.Fprintf(b, "%s\n", div)
	fmt.Fprintf(b, "  %s   %s\n", brand, tagline)
	fmt.Fprintf(b, "%s\n\n", div)
}

func writeFinding(b *strings.Builder, t style.Theme, f rules.Finding, width int) {
	badge := t.SeverityBadge(string(f.Severity))
	wl := t.Workload.Render(f.Workload)

	// First line: badge | workload | savings (right-anchored when present)
	right := ""
	switch {
	case f.MonthlyUSDCents > 0:
		right = t.Savings.Render("save ~$" + formatCents(f.MonthlyUSDCents) + " /mo")
	}

	if right != "" {
		fmt.Fprintf(b, "  %s  %s   %s\n", badge, wl, right)
	} else {
		fmt.Fprintf(b, "  %s  %s\n", badge, wl)
	}

	// Title (bold) + detail (muted)
	fmt.Fprintf(b, "    %s\n", t.Title.Render(f.Title))
	for _, line := range wrap(f.Detail, width-6) {
		fmt.Fprintf(b, "    %s\n", t.Detail.Render(line))
	}

	// Confidence row
	fmt.Fprintf(b, "    %s %s\n",
		t.Muted.Render("confidence:"),
		t.ConfidenceDots(string(f.Confidence)),
	)
}

func writeFooter(b *strings.Builder, t style.Theme, width int, totalCents int64) {
	fmt.Fprintf(b, "%s\n", t.DividerLine(width))
	if totalCents > 0 {
		fmt.Fprintf(b, "  %s %s   %s\n",
			t.Muted.Render("estimated monthly savings:"),
			t.Savings.Render("$"+formatCents(totalCents)+" /mo"),
			t.Muted.Render("(±40%)"),
		)
	}
	fmt.Fprintf(b, "  %s\n", t.Disclosure.Render(AccuracyDisclosure))
	// Style the label first; then wrap with OSC 8. Wrapping the
	// hyperlink escape with another lipgloss render breaks the
	// sequence because lipgloss styles each character.
	linkLabel := t.CallToLink.Render("sevro.dev/get")
	fmt.Fprintf(b, "  %s %s\n",
		t.Muted.Render("→ install the agent for exact numbers:"),
		t.Hyperlink(linkLabel, GetURL),
	)
}

// JSON writes the report as machine-readable JSON. Always disclosure-
// gated. Never colored — JSON output is for piping.
func JSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		AccuracyDisclosure string          `json:"accuracy_disclosure"`
		Source             string          `json:"source"`
		Workloads          int             `json:"workloads_analyzed"`
		Findings           []rules.Finding `json:"findings"`
		MonthlySavingsUSD  float64         `json:"monthly_savings_usd"`
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

func plural(n int, singular, pluralForm string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, pluralForm)
}

// wrap breaks s into lines no wider than width runes. Naïve word wrap;
// good enough for finding details which are sentences, not paragraphs.
func wrap(s string, width int) []string {
	if width <= 0 || len([]rune(s)) <= width {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	words := strings.Fields(s)
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if cur.Len()+1+len(w) > width {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
			continue
		}
		cur.WriteString(" ")
		cur.WriteString(w)
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}
