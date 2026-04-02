package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/archive"
	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/collector"
	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	var output string
	var namespace string
	var logLines int
	var wait bool

	flag.StringVar(&output, "output", "", "Output zip file path (default: debug.zip)")
	flag.StringVar(&namespace, "namespace", "", "Namespace where SlurmCluster is deployed")
	flag.IntVar(&logLines, "log-lines", 200, "Number of log lines per pod")
	flag.BoolVar(&wait, "wait", false, "Wait after writing (for kubectl cp)")
	flag.Parse()

	config := ctrl.GetConfigOrDie()

	cluster, detectedNs, err := detectCluster(namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if namespace == "" {
		namespace = detectedNs
	}
	fmt.Fprintf(os.Stderr, "Detected cluster: %s (namespace: %s)\n", cluster, namespace)

	cc, err := common.NewCollectorContext(config, namespace, cluster, logLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating collector: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "Collecting debug info for cluster %s in namespace %s...\n", cluster, namespace)
	debugReport, err := collector.Collect(ctx, cc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Collection error (continuing with partial data): %v\n", err)
	}

	if output == "" {
		output = "debug.zip"
	}

	if err := archive.WriteZip(output, debugReport); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing zip: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Debug archive written to: %s\n", output)

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for download (send SIGTERM to exit or timeout in 5 minutes)...\n")
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		select {
		case <-sigChan:
			fmt.Fprintf(os.Stderr, "Signal received, exiting.\n")
		case <-time.After(5 * time.Minute):
			fmt.Fprintf(os.Stderr, "Timeout reached, exiting.\n")
		}
	}
}

// detectCluster finds the SlurmCluster. If namespace is provided, searches only
// that namespace. Otherwise searches all namespaces.
func detectCluster(namespace string) (string, string, error) {
	restConfig := ctrl.GetConfigOrDie()

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create dynamic client: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "slurm.crusoe.ai",
		Version:  "v1alpha1",
		Resource: "slurmclusters",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var list *unstructured.UnstructuredList
	if namespace != "" {
		list, err = dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to list SlurmClusters: %w", err)
	}

	switch len(list.Items) {
	case 0:
		if namespace != "" {
			return "", "", fmt.Errorf("no SlurmCluster found in namespace %s", namespace)
		}
		return "", "", fmt.Errorf("no SlurmCluster found in any namespace")
	case 1:
		return list.Items[0].GetName(), list.Items[0].GetNamespace(), nil
	default:
		var found []string
		for _, item := range list.Items {
			found = append(found, fmt.Sprintf("%s/%s", item.GetNamespace(), item.GetName()))
		}
		return "", "", fmt.Errorf("multiple SlurmClusters found: %v", found)
	}
}
