package slinky

import (
	"context"
	"fmt"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectHelmReleases collects status of Helm releases defined in AdditionalNamespaces.
func CollectHelmReleases(_ context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	for _, rel := range common.AllHelmReleases() {
		release := &common.HelmRelease{
			Name:      rel.Name,
			Namespace: rel.Namespace,
		}

		release.Status = cc.RunHelm("status", rel.Name, "-n", rel.Namespace)
		release.History = cc.RunHelm("history", rel.Name, "-n", rel.Namespace)
		release.Values = cc.RunHelm("get", "values", rel.Name, "-n", rel.Namespace, "-a")
		release.Manifest = cc.RunHelm("get", "manifest", rel.Name, "-n", rel.Namespace)

		key := fmt.Sprintf("%s/%s", rel.Namespace, rel.Name)
		report.HelmReleases[key] = release
	}
}
