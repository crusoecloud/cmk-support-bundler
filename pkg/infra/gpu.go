package infra

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectGPUDiagnostics collects GPU info from all worker pods.
func CollectGPUDiagnostics(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	cc.ForEachRunningWorker(report, func(pod corev1.Pod) {
		diag := &common.GPUDiagnostics{}

		tasks := []struct {
			target *string
			cmd    []string
		}{
			{&diag.NvidiaSmiQuery, []string{"sh", "-c", "nvidia-smi -q 2>/dev/null | sed '/Processes/,$d'"}},
			{&diag.NvidiaSmiCSV, []string{"sh", "-c", "nvidia-smi --query-gpu=index,name,uuid,persistence_mode,pstate,temperature.gpu,temperature.memory,power.draw,power.limit,clocks_throttle_reasons.active,ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total,memory.used,memory.total,utilization.gpu,utilization.memory,pcie.link.gen.current,pcie.link.width.current --format=csv 2>/dev/null || echo 'query-gpu not available'"}},
			{&diag.PersistenceMode, []string{"sh", "-c", "nvidia-smi -q 2>/dev/null | grep -A1 'Persistence Mode' | head -20 || echo 'persistence query not available'"}},
			{&diag.GPUTopology, []string{"nvidia-smi", "topo", "-m"}},
			{&diag.NVLinkStatus, []string{"nvidia-smi", "nvlink", "-s"}},
			{&diag.NVLinkCounters, []string{"nvidia-smi", "nvlink", "-c"}},
		}

		for _, t := range tasks {
			if out, err := cc.ExecInPod(ctx, pod.Name, common.DefaultSlurmdContainer, t.cmd); err == nil {
				*t.target = out
			}
		}

		report.GPUInfo[pod.Name] = diag
	})
}
