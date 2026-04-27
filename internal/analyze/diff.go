package analyze

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/lowplane/sevro/pkg/parser"
)

// DiffEntry is the per-workload change between two values files.
//
// Sets are computed by name; a workload present only in B is "added"
// (NewWorkload=true), only in A is "removed" (RemovedWorkload=true).
type DiffEntry struct {
	Name             string
	NewWorkload      bool
	RemovedWorkload  bool
	CPURequestDelta  int64 // millicores; B - A
	CPULimitDelta    int64
	MemoryRequestDelta int64 // bytes; B - A
	MemoryLimitDelta   int64
	// MonthlyUSDCentsDelta is a sandbox-grade dollar estimate based on
	// the resource deltas. ±40% disclosure caveat applies.
	MonthlyUSDCentsDelta int64
}

// DiffReport is the renderer-facing view of a diff run.
type DiffReport struct {
	A       string      `json:"a"`
	B       string      `json:"b"`
	Entries []DiffEntry `json:"entries"`
}

// Pricing constants must match those in internal/rules so the diff's
// monthly delta is consistent with the analyze report's savings.
const (
	cpuPriceCentsPerCoreHour = 4   // CPU $/vCPU-hour, AWS m5 baseline
	cpuMonthlyHours          = 730 // hours per month, AWS billing convention
	memPriceCentsPerGiBMonth = 350 // $3.50 per GiB-month
)

// Diff returns a DiffReport between two streams of Helm values.
func Diff(a, b io.Reader, aLabel, bLabel string) (DiffReport, error) {
	wlA, err := parser.ParseValues(a)
	if err != nil {
		return DiffReport{}, fmt.Errorf("diff: parse %q: %w", aLabel, err)
	}
	wlB, err := parser.ParseValues(b)
	if err != nil {
		return DiffReport{}, fmt.Errorf("diff: parse %q: %w", bLabel, err)
	}

	byName := map[string]*pair{}
	for i := range wlA {
		byName[wlA[i].Name] = &pair{a: &wlA[i]}
	}
	for i := range wlB {
		if p, ok := byName[wlB[i].Name]; ok {
			p.b = &wlB[i]
		} else {
			byName[wlB[i].Name] = &pair{b: &wlB[i]}
		}
	}

	entries := make([]DiffEntry, 0, len(byName))
	for name, p := range byName {
		entry := DiffEntry{Name: name}
		switch {
		case p.a == nil:
			entry.NewWorkload = true
		case p.b == nil:
			entry.RemovedWorkload = true
		}
		entry.CPURequestDelta = qDelta(get(p.a, reqCPU), get(p.b, reqCPU))
		entry.CPULimitDelta = qDelta(get(p.a, limCPU), get(p.b, limCPU))
		entry.MemoryRequestDelta = qDelta(get(p.a, reqMem), get(p.b, reqMem))
		entry.MemoryLimitDelta = qDelta(get(p.a, limMem), get(p.b, limMem))
		entry.MonthlyUSDCentsDelta = monthlyDelta(entry)
		// Suppress no-op entries: same workload on both sides with zero
		// deltas. Keep New/Removed entries even when the resource math
		// happens to net to zero — an added workload with no resources
		// is still useful to surface.
		if !entry.NewWorkload && !entry.RemovedWorkload &&
			entry.CPURequestDelta == 0 && entry.CPULimitDelta == 0 &&
			entry.MemoryRequestDelta == 0 && entry.MemoryLimitDelta == 0 {
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })

	return DiffReport{A: aLabel, B: bLabel, Entries: entries}, nil
}

// DiffPaths is the convenience wrapper used by `sevro diff <a> <b>`.
func DiffPaths(a, b string) (DiffReport, error) {
	fa, err := openValues(a)
	if err != nil {
		return DiffReport{}, err
	}
	defer fa.Close()
	fb, err := openValues(b)
	if err != nil {
		return DiffReport{}, err
	}
	defer fb.Close()
	return Diff(fa, fb, a, b)
}

// MonthlyUSDCentsDelta totals the per-entry deltas. Negative means
// the change is a saving (B costs less than A); positive means a
// regression.
func (r DiffReport) MonthlyUSDCentsDelta() int64 {
	var sum int64
	for _, e := range r.Entries {
		sum += e.MonthlyUSDCentsDelta
	}
	return sum
}

// pair is an internal helper; one workload-name → A side and B side.
type pair struct {
	a, b *parser.Workload
}

type which int

const (
	reqCPU which = iota
	limCPU
	reqMem
	limMem
)

func get(w *parser.Workload, k which) parser.Quantity {
	if w == nil {
		return parser.Quantity{}
	}
	switch k {
	case reqCPU:
		return w.Requests.CPU
	case limCPU:
		return w.Limits.CPU
	case reqMem:
		return w.Requests.Memory
	case limMem:
		return w.Limits.Memory
	}
	return parser.Quantity{}
}

func qDelta(a, b parser.Quantity) int64 {
	var av, bv int64
	if a.Set {
		av = a.Value
	}
	if b.Set {
		bv = b.Value
	}
	return bv - av
}

// monthlyDelta converts CPU and memory deltas into an estimated
// monthly USD-cents difference. Sandbox-grade per the ±40%
// disclosure; replaced by measured numbers when the agent ships.
func monthlyDelta(e DiffEntry) int64 {
	cpuMillicoreDelta := e.CPURequestDelta // request drives reserved capacity
	memBytesDelta := e.MemoryRequestDelta

	cpuCents := cpuMillicoreDelta * cpuMonthlyHours * cpuPriceCentsPerCoreHour / 1000
	memCents := int64(float64(memBytesDelta) / float64(1024*1024*1024) * float64(memPriceCentsPerGiBMonth))
	return cpuCents + memCents
}

func openValues(path string) (*os.File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("diff: abs %s: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("diff: stat %s: %w", abs, err)
	}
	target := abs
	if info.IsDir() {
		target = filepath.Join(abs, "values.yaml")
	}
	return os.Open(target) //nolint:gosec // user-specified analysis input
}
