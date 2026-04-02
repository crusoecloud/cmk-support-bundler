package slurm

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectClusterCRs collects the SlurmCluster CR.
func CollectClusterCRs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	gvr := schema.GroupVersionResource{
		Group:    "slurm.crusoe.ai",
		Version:  "v1alpha1",
		Resource: "slurmclusters",
	}

	cluster, err := cc.DynamicClient.Resource(gvr).Namespace(cc.Namespace).Get(ctx, cc.Cluster, metav1.GetOptions{})
	if err != nil {
		report.AddError("Failed to get SlurmCluster %s: %v", cc.Cluster, err)
	} else {
		report.SlurmCluster = cluster
	}
}
