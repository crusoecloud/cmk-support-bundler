package infra

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectSystemDiagnostics collects system info from all worker pods.
func CollectSystemDiagnostics(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	cc.ForEachRunningWorker(report, func(pod corev1.Pod) {
		diag := &common.SystemDiagnostics{}

		tasks := []struct {
			field *string
			args  []string
		}{
			{&diag.CPUInfo, []string{"cat", "/proc/cpuinfo"}},
			{&diag.MemInfo, []string{"cat", "/proc/meminfo"}},
			{&diag.Lscpu, []string{"lscpu"}},
			{&diag.Free, []string{"sh", "-c", "free -m 2>/dev/null || echo 'free not available'"}},
			{&diag.DiskFree, []string{"df", "-h"}},
			{&diag.Iostat, []string{"sh", "-c", "iostat -x 1 1 2>/dev/null || echo 'iostat not available'"}},
			{&diag.Ulimit, []string{"sh", "-c", "ulimit -a"}},
			{&diag.IPAddr, []string{"ip", "addr"}},
			{&diag.IPLink, []string{"ip", "link"}},
			{&diag.Hosts, []string{"cat", "/etc/hosts"}},
			{&diag.Mounts, []string{"mount"}},
			{&diag.Uptime, []string{"uptime"}},
			{&diag.Dmesg, []string{"dmesg"}},
			{&diag.LoadAvg, []string{"cat", "/proc/loadavg"}},
			{&diag.SysctlNet, []string{"sh", "-c", "sysctl -a 2>/dev/null | grep -E '^net\\.' | head -200 || echo 'sysctl not available'"}},
		}

		for _, t := range tasks {
			if out, err := cc.ExecInPod(ctx, pod.Name, common.DefaultSlurmdContainer, t.args); err == nil {
				*t.field = out
			}
		}

		report.SystemInfo[pod.Name] = diag
	})
}
