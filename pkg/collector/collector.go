package collector

import (
	"context"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/infra"
	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/slinky"
	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/slurm"
)

// Collect gathers all debug information.
func Collect(ctx context.Context, cc *common.CollectorContext) (*common.DebugReport, error) {
	report := common.NewDebugReport(cc.Cluster, cc.Namespace, cc.LogLines)

	cc.Log("[1/13] Collecting SlurmCluster CRs...")
	slurm.CollectClusterCRs(ctx, cc, report)

	cc.Log("[2/13] Collecting Slinky CRs (Controller, LoginSets, NodeSets)...")
	slinky.CollectCRs(ctx, cc, report)

	cc.Log("[3/13] Collecting pod information...")
	slurm.CollectPods(ctx, cc, report)

	cc.Log("[4/13] Collecting ConfigMaps...")
	slurm.CollectConfigMaps(ctx, cc, report)

	cc.Log("[5/13] Collecting mounted Slurm configs from pods...")
	slurm.CollectMountedConfigs(ctx, cc, report)

	cc.Log("[6/13] Collecting Slurm status (sinfo, scontrol, sdiag)...")
	slurm.CollectStatus(ctx, cc, report)

	cc.Log("[7/13] Collecting container and daemon logs...")
	slurm.CollectLogs(ctx, cc, report)

	cc.Log("[8/13] Collecting operator logs...")
	slinky.CollectOperatorLogs(ctx, cc, report)

	cc.Log("[9/13] Collecting topograph pods and logs...")
	slinky.CollectTopographLogs(ctx, cc, report)

	cc.Log("[10/13] Collecting GPU diagnostics (nvidia-smi, nvlink)...")
	infra.CollectGPUDiagnostics(ctx, cc, report)

	cc.Log("[11/13] Collecting network diagnostics (ibstat, infiniband)...")
	infra.CollectNetworkDiagnostics(ctx, cc, report)

	cc.Log("[12/13] Collecting system diagnostics (cpu, memory, disk)...")
	infra.CollectSystemDiagnostics(ctx, cc, report)

	cc.Log("[13/13] Collecting Kubernetes resources and Helm releases...")
	infra.CollectKubernetesResources(ctx, cc, report)
	slinky.CollectHelmReleases(ctx, cc, report)

	cc.Log("Collection complete!")
	return report, nil
}
