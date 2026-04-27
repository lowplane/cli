package rules

import "github.com/lowplane/sevro/pkg/parser"

// hostPathVolume fires when a workload mounts any volume from the
// node's filesystem. Most application charts do not need this; when
// they do, it's almost always reading `/var/log` or `/etc/passwd`
// in a way that should be replaced with downward-API or a sidecar.
type hostPathVolume struct{}

func newHostPathVolume() Detector { return hostPathVolume{} }

func (hostPathVolume) ID() string   { return "host-path-volume" }
func (hostPathVolume) Name() string { return "hostPath volume mounted" }

func (hostPathVolume) Run(w parser.Workload) []Finding {
	if w.Security.HostPath == nil || !*w.Security.HostPath {
		return nil
	}
	return []Finding{{
		DetectorID: "host-path-volume",
		Workload:   w.Name,
		Title:      "Pod mounts a hostPath volume",
		Detail:     "A volume of type hostPath exposes the node's filesystem to the pod. Required for some node-level agents (log collectors, CNI helpers) but a sharp edge for application workloads — a path-traversal bug suddenly becomes a node compromise. Replace with PVC, configMap, or downward API where possible.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
