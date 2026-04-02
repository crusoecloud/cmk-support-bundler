package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// CollectorContext holds the shared clients and config needed by all collectors.
type CollectorContext struct {
	Clientset     *kubernetes.Clientset
	DynamicClient dynamic.Interface
	RestConfig    *rest.Config
	Namespace     string
	Cluster       string
	LogLines      int
}

// NewCollectorContext creates a new CollectorContext.
func NewCollectorContext(config *rest.Config, namespace, cluster string, logLines int) (*CollectorContext, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &CollectorContext{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		RestConfig:    config,
		Namespace:     namespace,
		Cluster:       cluster,
		LogLines:      logLines,
	}, nil
}

// Log prints a progress message to stderr.
func (cc *CollectorContext) Log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// ExecInPod executes a command in a pod's container and returns the output.
func (cc *CollectorContext) ExecInPod(ctx context.Context, podName, container string, command []string) (string, error) {
	req := cc.Clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(cc.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(cc.RestConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n--- stderr ---\n"
		}
		output += stderr.String()
	}

	if err != nil {
		return output, fmt.Errorf("exec failed: %w (output: %s)", err, output)
	}

	return output, nil
}

// RunKubectl runs a kubectl command and returns the output.
func (cc *CollectorContext) RunKubectl(args ...string) string {
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error running kubectl %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}

// RunHelm runs a helm command and returns the output.
func (cc *CollectorContext) RunHelm(args ...string) string {
	cmd := exec.Command("helm", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error running helm %v: %v\n%s", args, err, string(out))
	}
	return string(out)
}

// ForEachRunningWorker iterates over running worker pods with progress logging.
func (cc *CollectorContext) ForEachRunningWorker(report *DebugReport, fn func(pod corev1.Pod)) {
	totalWorkers := len(report.WorkerPods)
	for i, pod := range report.WorkerPods {
		if pod.Status.Phase != corev1.PodRunning {
			cc.Log("  - Worker %d/%d: %s (skipped, not running)", i+1, totalWorkers, pod.Name)
			continue
		}
		cc.Log("  - Worker %d/%d: %s", i+1, totalWorkers, pod.Name)
		fn(pod)
	}
}

// CollectPodContainerLogs collects logs from all containers in a pod,
// including sidecar init containers (restartPolicy=Always).
func (cc *CollectorContext) CollectPodContainerLogs(ctx context.Context, pod *corev1.Pod, ns string, maxLogLines int64) map[string]string {
	result := make(map[string]string)

	for _, container := range pod.Spec.InitContainers {
		if container.RestartPolicy == nil || *container.RestartPolicy != corev1.ContainerRestartPolicyAlways {
			continue
		}
		logs, err := cc.Clientset.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: container.Name,
			TailLines: &maxLogLines,
		}).Do(ctx).Raw()
		if err != nil {
			result[container.Name] = fmt.Sprintf("Error: %v", err)
		} else {
			result[container.Name] = string(logs)
		}
	}

	for _, container := range pod.Spec.Containers {
		logs, err := cc.Clientset.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
			Container: container.Name,
			TailLines: &maxLogLines,
		}).Do(ctx).Raw()
		if err != nil {
			result[container.Name] = fmt.Sprintf("Error: %v", err)
		} else {
			result[container.Name] = string(logs)
		}
	}
	return result
}

// CollectPodContainerLogsFlat collects logs from all containers and returns
// them as a single concatenated string.
func (cc *CollectorContext) CollectPodContainerLogsFlat(ctx context.Context, pod *corev1.Pod, ns string, maxLogLines int64) string {
	containerLogs := cc.CollectPodContainerLogs(ctx, pod, ns, maxLogLines)
	var result string
	for name, logs := range containerLogs {
		result += fmt.Sprintf("=== Container %s ===\n%s\n\n", name, logs)
	}
	return result
}

// GetPodScanNamespaces returns all namespaces for the pod listing scan,
// including the cluster namespace and all additional namespaces with Pods enabled.
func (cc *CollectorContext) GetPodScanNamespaces() []string {
	podNs := PodListingNamespaces()
	namespaces := make([]string, 0, len(podNs)+1)
	namespaces = append(namespaces, podNs...)

	clusterNsPresent := false
	for _, ns := range namespaces {
		if ns == cc.Namespace {
			clusterNsPresent = true
			break
		}
	}
	if !clusterNsPresent {
		namespaces = append(namespaces, cc.Namespace)
	}
	return namespaces
}

// CollectPodsFromAllowedNamespaces collects pods from allowed namespaces only.
func (cc *CollectorContext) CollectPodsFromAllowedNamespaces() string {
	var result strings.Builder
	result.WriteString("NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE\n")

	for _, ns := range cc.GetPodScanNamespaces() {
		out := cc.RunKubectl("get", "pods", "-n", ns, "-o", "wide", "--no-headers")
		if out != "" && !strings.Contains(out, "No resources found") {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					result.WriteString(ns + "\t" + line + "\n")
				}
			}
		}
	}
	return result.String()
}
