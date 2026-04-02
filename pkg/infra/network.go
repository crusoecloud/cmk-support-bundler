package infra

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectNetworkDiagnostics collects InfiniBand and NCCL info from workers.
func CollectNetworkDiagnostics(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	type command struct {
		field *string
		cmd   string
		label string
	}

	cc.ForEachRunningWorker(report, func(pod corev1.Pod) {
		diag := &common.NetworkDiagnostics{}

		commands := []command{
			{&diag.IBStat, "ibstat", "ibstat not available"},
			{&diag.IBStatus, "ibstatus", "ibstatus not available"},
			{&diag.IBVDevinfo, "ibv_devinfo", "ibv_devinfo not available"},
			{&diag.NCCLEnv, "env | grep -i nccl | sort", "No NCCL environment variables set"},
		}

		for _, item := range commands {
			shellCmd := fmt.Sprintf("%s 2>/dev/null || echo '%s'", item.cmd, item.label)
			if out, err := cc.ExecInPod(ctx, pod.Name, common.DefaultSlurmdContainer, []string{"sh", "-c", shellCmd}); err == nil {
				*item.field = out
			}
		}

		report.NetworkInfo[pod.Name] = diag
	})
}
