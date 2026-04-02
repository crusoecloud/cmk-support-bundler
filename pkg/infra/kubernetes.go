package infra

import (
	"context"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectKubernetesResources collects K8s resources using kubectl.
func CollectKubernetesResources(_ context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	cc.Log("  - Services...")
	report.ServicesDescribe = cc.RunKubectl("get", "services", "-n", cc.Namespace, "-o", "yaml")

	cc.Log("  - Endpoints...")
	report.EndpointsDescribe = cc.RunKubectl("get", "endpoints", "-n", cc.Namespace, "-o", "yaml")

	cc.Log("  - Nodes...")
	report.NodesDescribe = cc.RunKubectl("get", "nodes", "-o", "yaml")

	cc.Log("  - PVCs...")
	report.PVCsDescribe = cc.RunKubectl("get", "pvc", "-n", cc.Namespace, "-o", "yaml")

	cc.Log("  - Events...")
	report.EventsDescribe = cc.RunKubectl("get", "events", "-n", cc.Namespace, "--sort-by=.lastTimestamp")

	cc.Log("  - Cluster-wide pod listing (filtered namespaces)...")
	report.AllPodsListing = cc.CollectPodsFromAllowedNamespaces()

	cc.Log("  - Cluster-wide node listing...")
	report.AllNodesListing = cc.RunKubectl("get", "nodes", "-o", "wide")
}
