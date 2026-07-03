// Package resourcesampler reports coarse CPU and memory usage for a running
// process and its descendants. It backs the per-service resource readout in
// the info command and the dashboard.
//
// Every call is fail-soft: a process that has already exited (or that the
// current user cannot query) yields zero usage rather than an error, so
// callers can sample on each status refresh without having to special-case
// the race between reading the registry and the process going away.
package resourcesampler

import (
	"github.com/shirou/gopsutil/v4/process"
)

// maxProcesses caps how many processes a single Sample call will visit while
// walking a process tree. It guards against a pathological tree, or PID reuse
// producing a cycle, turning a routine status refresh into an unbounded walk.
const maxProcesses = 256

// Usage is a point-in-time resource snapshot for a process tree.
type Usage struct {
	// CPUPercent is the summed CPU usage of the process tree as a percentage
	// of a single core, averaged over each process's lifetime. It can exceed
	// 100 when work is spread across multiple cores.
	CPUPercent float64
	// MemoryBytes is the summed resident set size of the process tree in bytes.
	MemoryBytes uint64
}

// Sample walks the process tree rooted at pid and returns the summed CPU and
// resident memory usage. A non-positive pid, an exited process, or any lookup
// error results in a zero Usage.
func Sample(pid int) Usage {
	if pid <= 0 {
		return Usage{}
	}

	root, err := process.NewProcess(int32(pid))
	if err != nil {
		// Process is gone or not queryable. Report nothing rather than
		// surfacing a transient error to the caller.
		return Usage{}
	}

	var usage Usage
	visited := make(map[int32]struct{})
	queue := []*process.Process{root}

	for len(queue) > 0 && len(visited) < maxProcesses {
		p := queue[0]
		queue = queue[1:]

		if _, seen := visited[p.Pid]; seen {
			continue
		}
		visited[p.Pid] = struct{}{}

		if cpu, err := p.CPUPercent(); err == nil {
			usage.CPUPercent += cpu
		}
		if mem, err := p.MemoryInfo(); err == nil && mem != nil {
			usage.MemoryBytes += mem.RSS
		}
		if children, err := p.Children(); err == nil {
			queue = append(queue, children...)
		}
	}

	return usage
}
