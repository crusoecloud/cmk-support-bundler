package slurm

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectStatus collects Slurm status commands from the controller pod.
func CollectStatus(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	if report.ControllerPod == nil || report.ControllerPod.Status.Phase != corev1.PodRunning {
		report.AddError("Controller pod not running, skipping Slurm status collection")
		return
	}

	podName := report.ControllerPod.Name
	container := common.DefaultSlurmctldContainer

	// sinfo - node/partition status
	if out, err := cc.ExecInPod(ctx, podName, container, []string{"sinfo", "-a", "-l"}); err == nil {
		report.Sinfo = out
	} else {
		report.AddError("Failed to run sinfo: %v", err)
	}

	// scontrol show config
	if out, err := cc.ExecInPod(ctx, podName, container, []string{"scontrol", "show", "config"}); err == nil {
		report.ScontrolConfig = out
	} else {
		report.AddError("Failed to run scontrol show config: %v", err)
	}

	// scontrol show nodes
	if out, err := cc.ExecInPod(ctx, podName, container, []string{"scontrol", "show", "nodes"}); err == nil {
		report.ScontrolNodes = out
	} else {
		report.AddError("Failed to run scontrol show nodes: %v", err)
	}

	// scontrol show partitions
	if out, err := cc.ExecInPod(ctx, podName, container, []string{"scontrol", "show", "partitions"}); err == nil {
		report.ScontrolPartitions = out
	} else {
		report.AddError("Failed to run scontrol show partitions: %v", err)
	}

	// sdiag - scheduler diagnostics
	if out, err := cc.ExecInPod(ctx, podName, container, []string{"sdiag"}); err == nil {
		report.Sdiag = out
	} else {
		report.AddError("Failed to run sdiag: %v", err)
	}
}
