package rules

import "github.com/lowplane/sevro/pkg/parser"

// hostPID and hostIPC fire when the workload joins the node's PID
// or IPC namespace. Either flag is a serious isolation breach:
// hostPID lets a pod see / signal every process on the node;
// hostIPC lets it read / write SysV shared memory used by other pods.
//
// CIS Kubernetes Benchmark 5.2.2 / 5.2.3 ("Minimize the admission of
// containers wishing to share the host process / IPC namespace").

type hostPID struct{}

func newHostPID() Detector { return hostPID{} }

func (hostPID) ID() string   { return "host-pid" }
func (hostPID) Name() string { return "hostPID enabled" }

func (hostPID) Run(w parser.Workload) []Finding {
	if w.Security.HostPID == nil || !*w.Security.HostPID {
		return nil
	}
	return []Finding{{
		DetectorID: "host-pid",
		Workload:   w.Name,
		Title:      "Pod shares the host PID namespace",
		Detail:     "hostPID=true gives the pod's processes visibility into every other process on the node — including kubelet, the container runtime, and every co-tenant pod. Required for a tiny set of node-level agents (debuggers, profilers); never appropriate for application workloads.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}

type hostIPC struct{}

func newHostIPC() Detector { return hostIPC{} }

func (hostIPC) ID() string   { return "host-ipc" }
func (hostIPC) Name() string { return "hostIPC enabled" }

func (hostIPC) Run(w parser.Workload) []Finding {
	if w.Security.HostIPC == nil || !*w.Security.HostIPC {
		return nil
	}
	return []Finding{{
		DetectorID: "host-ipc",
		Workload:   w.Name,
		Title:      "Pod shares the host IPC namespace",
		Detail:     "hostIPC=true puts the pod in the node's IPC namespace, so it can read and write SysV shared memory and POSIX message queues used by other pods on the same node. Almost never required; turn it off unless the workload's documented dependency demands it.",
		Severity:   SeverityHigh,
		Confidence: ConfidenceHigh,
	}}
}
