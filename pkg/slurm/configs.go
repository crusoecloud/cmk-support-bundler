package slurm

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectConfigMaps collects Slurm/Slinky-related ConfigMaps in the SlurmCluster namespace.
func CollectConfigMaps(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	allCMs, err := cc.Clientset.CoreV1().ConfigMaps(cc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		report.AddError("Failed to list ConfigMaps: %v", err)
		return
	}

	for _, cm := range allCMs.Items {
		if !isSlurmRelatedConfigMap(cm.Name, cc.Cluster) {
			continue
		}
		if len(cm.Data) > 0 {
			report.SlurmConfigMaps[cm.Name] = make(map[string]string)
			for key, value := range cm.Data {
				report.SlurmConfigMaps[cm.Name][key] = value
			}
		}
	}
}

// slurmConfigMapPatterns are substrings that identify Slurm-related ConfigMaps.
var slurmConfigMapPatterns = []string{"slurm", "topology", "gres", "munge"}

// isSlurmRelatedConfigMap returns true if the ConfigMap name matches Slurm-related patterns.
func isSlurmRelatedConfigMap(name, cluster string) bool {
	if strings.Contains(name, cluster) {
		return true
	}
	for _, pattern := range slurmConfigMapPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}

// CollectMountedConfigs collects all config files from /etc/slurm/*.
func CollectMountedConfigs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	if report.ControllerPod == nil || report.ControllerPod.Status.Phase != corev1.PodRunning {
		report.AddError("Controller pod not running, skipping mounted config collection")
		return
	}

	podName := report.ControllerPod.Name
	container := common.DefaultSlurmctldContainer

	configFiles := []string{
		"slurm.conf",
		"gres.conf",
		"topology.conf",
		"cgroup.conf",
		"plugstack.conf",
		"prolog.sh",
		"epilog.sh",
		"prolog_slurmctld.sh",
		"epilog_slurmctld.sh",
	}

	for _, filename := range configFiles {
		if out, err := cc.ExecInPod(ctx, podName, container, []string{
			"sh", "-c", "cat /etc/slurm/" + filename + " 2>/dev/null || true",
		}); err == nil && strings.TrimSpace(out) != "" {
			report.MountedConfigs[filename] = out
		}
	}

	// List all files in /etc/slurm to catch any we missed (excluding key files)
	if out, err := cc.ExecInPod(ctx, podName, container, []string{
		"sh", "-c", "ls -la /etc/slurm/ 2>/dev/null | grep -v '\\.key' || echo 'directory not accessible'",
	}); err == nil {
		report.MountedConfigs["_directory_listing.txt"] = out
	}
}
