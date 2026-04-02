package slinky

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
)

// CollectCRs collects Slinky Controller, LoginSet, and NodeSet CRs.
func CollectCRs(ctx context.Context, cc *common.CollectorContext, report *common.DebugReport) {
	// Slinky Controller
	controllerGVR := schema.GroupVersionResource{
		Group:    "slinky.slurm.net",
		Version:  "v1beta1",
		Resource: "controllers",
	}

	controllerName := fmt.Sprintf("slurm-%s", cc.Cluster)
	controller, err := cc.DynamicClient.Resource(controllerGVR).Namespace(cc.Namespace).Get(ctx, controllerName, metav1.GetOptions{})
	if err != nil {
		report.AddError("Failed to get Slinky Controller %s: %v", controllerName, err)
	} else {
		report.SlinkyController = controller
	}

	// Slinky LoginSets
	loginSetGVR := schema.GroupVersionResource{
		Group:    "slinky.slurm.net",
		Version:  "v1beta1",
		Resource: "loginsets",
	}

	loginSetList, err := cc.DynamicClient.Resource(loginSetGVR).Namespace(cc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		report.AddError("Failed to list Slinky LoginSets: %v", err)
	} else {
		for _, ls := range loginSetList.Items {
			if strings.Contains(ls.GetName(), cc.Cluster) {
				report.SlinkyLoginSets = append(report.SlinkyLoginSets, ls)
			}
		}
	}

	// Slinky NodeSets
	nodeSetGVR := schema.GroupVersionResource{
		Group:    "slinky.slurm.net",
		Version:  "v1beta1",
		Resource: "nodesets",
	}

	nodeSetList, err := cc.DynamicClient.Resource(nodeSetGVR).Namespace(cc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		report.AddError("Failed to list Slinky NodeSets: %v", err)
	} else {
		for _, ns := range nodeSetList.Items {
			if strings.Contains(ns.GetName(), cc.Cluster) {
				report.SlinkyNodeSets = append(report.SlinkyNodeSets, ns)
			}
		}
	}
}
