package slurm

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectPods collects controller, login, and worker pods.
func CollectPods(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	pods, err := cc.Clientset.CoreV1().Pods(cc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		report.AddError("Failed to list pods: %v", err)
		return
	}

	controllerPrefix := fmt.Sprintf("slurm-%s-controller", cc.Cluster)
	loginPrefix := fmt.Sprintf("slurm-%s-login", cc.Cluster)

	for i := range pods.Items {
		pod := &pods.Items[i]
		name := pod.Name

		if strings.HasPrefix(name, controllerPrefix) {
			report.ControllerPod = pod
			continue
		}

		if strings.HasPrefix(name, loginPrefix) {
			report.LoginPods = append(report.LoginPods, *pod)
			continue
		}

		// Worker pods: {cluster}-{nodeset}-{index}
		if strings.HasPrefix(name, cc.Cluster+"-") {
			report.WorkerPods = append(report.WorkerPods, *pod)
		}
	}
}

// CollectLogs collects logs from controller, login, and worker pods.
func CollectLogs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	// Controller logs
	if report.ControllerPod != nil {
		cc.Log("  - Controller pod: %s", report.ControllerPod.Name)
		report.ControllerContainerLogs = cc.CollectPodContainerLogs(ctx, report.ControllerPod, cc.Namespace, int64(cc.LogLines))
	}

	// Login pod logs
	for i, pod := range report.LoginPods {
		if pod.Status.Phase != corev1.PodRunning {
			cc.Log("  - Login pod %d/%d: %s (skipped, not running)", i+1, len(report.LoginPods), pod.Name)
			continue
		}
		cc.Log("  - Login pod %d/%d: %s", i+1, len(report.LoginPods), pod.Name)
		report.LoginContainerLogs[pod.Name] = cc.CollectPodContainerLogs(ctx, &pod, cc.Namespace, int64(cc.LogLines))
	}

	// Worker pod logs
	totalWorkers := len(report.WorkerPods)
	for i, pod := range report.WorkerPods {
		if pod.Status.Phase != corev1.PodRunning {
			cc.Log("  - Worker pod %d/%d: %s (skipped, not running)", i+1, totalWorkers, pod.Name)
			continue
		}
		cc.Log("  - Worker pod %d/%d: %s", i+1, totalWorkers, pod.Name)
		report.WorkerContainerLogs[pod.Name] = cc.CollectPodContainerLogs(ctx, &pod, cc.Namespace, int64(cc.LogLines))
	}
}
