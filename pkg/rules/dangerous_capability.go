package rules

import (
	"fmt"
	"strings"

	"github.com/lowplane/sevro/pkg/parser"
)

// dangerousCapabilityAdded fires when capabilities.add includes one of
// the well-known escalation-class capabilities. CIS Kubernetes
// Benchmark 5.2.7 and the NSA/CISA Kubernetes Hardening Guide both
// flag these explicitly.
type dangerousCapabilityAdded struct{}

func newDangerousCapabilityAdded() Detector { return dangerousCapabilityAdded{} }

func (dangerousCapabilityAdded) ID() string   { return "dangerous-capability-added" }
func (dangerousCapabilityAdded) Name() string { return "Dangerous Linux capability added" }

// dangerousCaps is the closed set we treat as escalation-class. All of
// these grant abilities far beyond what application workloads need;
// most legitimate users are CSI drivers, network plugins, or
// monitoring agents — and those should run as DaemonSets the platform
// team explicitly trusts.
var dangerousCaps = map[string]string{
	"SYS_ADMIN":     "near-root: lets the container mount filesystems, change kernel parameters, etc.",
	"NET_ADMIN":     "configure network interfaces, manipulate routing tables, intercept traffic",
	"NET_RAW":       "craft raw packets — useful for ARP spoofing and other L2 attacks",
	"SYS_PTRACE":    "attach to other processes' memory; can read secrets out of co-tenant pods",
	"SYS_MODULE":    "load kernel modules; effectively root on the node",
	"DAC_READ_SEARCH": "bypass DAC permission checks on file reads",
	"DAC_OVERRIDE":  "bypass DAC permission checks on file writes",
	"SYS_BOOT":      "reboot the node",
	"SYS_TIME":      "change system time; breaks timing-based security guarantees",
	"BPF":           "load BPF programs; can read kernel memory",
	"PERFMON":       "perf_events syscall; can leak kernel address layouts",
}

func (dangerousCapabilityAdded) Run(w parser.Workload) []Finding {
	if len(w.Security.CapabilitiesAdd) == 0 {
		return nil
	}
	var hits []string
	for _, c := range w.Security.CapabilitiesAdd {
		if reason, ok := dangerousCaps[normaliseCap(c)]; ok {
			hits = append(hits, c+" ("+reason+")")
		}
	}
	if len(hits) == 0 {
		return nil
	}
	return []Finding{{
		DetectorID: "dangerous-capability-added",
		Workload:   w.Name,
		Title:      "Dangerous Linux capability added",
		Detail:     fmt.Sprintf("securityContext.capabilities.add includes %d escalation-class capability(s): %s. Each grants abilities far beyond what an application workload should need. Drop them and reach for a sidecar / DaemonSet that runs in a more privileged ServiceAccount if the underlying need is legitimate.", len(hits), strings.Join(hits, "; ")),
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}

// normaliseCap strips an optional CAP_ prefix and uppercases the rest
// so we accept both "CAP_NET_ADMIN" and "net_admin" forms.
func normaliseCap(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	c = strings.TrimPrefix(c, "CAP_")
	return c
}
