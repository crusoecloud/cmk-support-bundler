package slinky

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// topographPrefixes are the pod name prefixes for topograph-related pods.
var topographPrefixes = []string{"topograph", "node-observer", "node-data-broker"}

// CollectTopographLogs collects logs from topograph, node-observer, and node-data-broker pods.
func CollectTopographLogs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	namespacesToSearch := common.NamespacesForPrefixes(cc.Namespace, topographPrefixes)

	for _, ns := range namespacesToSearch {
		pods, err := cc.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for i := range pods.Items {
			pod := &pods.Items[i]

			if !matchesAnyPrefix(pod.Name, topographPrefixes) {
				continue
			}

			report.TopographPods = append(report.TopographPods, *pod)

			key := fmt.Sprintf("%s/%s", ns, pod.Name)
			report.TopographLogs[key] = cc.CollectPodContainerLogsFlat(ctx, pod, ns, int64(cc.LogLines))
		}
	}
}
