package slinky

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// operatorPrefixes are the pod name prefixes for operator pods.
var operatorPrefixes = []string{"slurm-operator"}

// CollectOperatorLogs collects logs from slurm-operator and slurm-operator-webhook pods.
func CollectOperatorLogs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	namespacesToSearch := common.NamespacesForPrefixes(cc.Namespace, operatorPrefixes)

	for _, ns := range namespacesToSearch {
		pods, err := cc.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if matchesAnyPrefix(pod.Name, operatorPrefixes) {
				key := fmt.Sprintf("%s/%s", ns, pod.Name)
				report.OperatorLogs[key] = cc.CollectPodContainerLogsFlat(ctx, pod, ns, int64(cc.LogLines))
			}
		}
	}
}

func matchesAnyPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
