package common

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DebugReport contains all collected debug information.
type DebugReport struct {
	Timestamp   time.Time
	ClusterName string
	Namespace   string
	LogLines    int

	// Slurm CRs
	SlurmCluster *unstructured.Unstructured

	// Slinky CRs
	SlinkyController *unstructured.Unstructured
	SlinkyLoginSets  []unstructured.Unstructured
	SlinkyNodeSets   []unstructured.Unstructured

	// Pods
	ControllerPod *corev1.Pod
	LoginPods     []corev1.Pod
	WorkerPods    []corev1.Pod

	// ConfigMaps (full data: name -> key -> value)
	SlurmConfigMaps map[string]map[string]string

	// Mounted configs from pods (filename -> content)
	MountedConfigs map[string]string

	// Slurm status commands
	Sinfo                string
	ScontrolConfig       string
	ScontrolNodes        string
	ScontrolPartitions string
	Sdiag              string

	// System info (per worker)
	SystemInfo map[string]*SystemDiagnostics

	// K8s resources
	ServicesDescribe     string
	EndpointsDescribe    string
	NodesDescribe string
	PVCsDescribe  string
	EventsDescribe       string
	AllPodsListing string
	AllNodesListing      string

	// Container logs
	ControllerContainerLogs map[string]string
	LoginContainerLogs      map[string]map[string]string
	WorkerContainerLogs     map[string]map[string]string

	// Operator logs
	OperatorLogs map[string]string

	// Topograph
	TopographPods []corev1.Pod
	TopographLogs map[string]string

	// Helm releases
	HelmReleases map[string]*HelmRelease

	// GPU diagnostics (per worker)
	GPUInfo map[string]*GPUDiagnostics

	// Network diagnostics (per worker)
	NetworkInfo map[string]*NetworkDiagnostics

	// Errors during collection
	Errors []string
}

// AddError adds an error to the report.
func (r *DebugReport) AddError(format string, args ...interface{}) {
	r.Errors = append(r.Errors, fmt.Sprintf(format, args...))
}

// NewDebugReport creates a new initialized DebugReport.
func NewDebugReport(cluster, namespace string, logLines int) *DebugReport {
	return &DebugReport{
		Timestamp:               time.Now().UTC(),
		ClusterName:             cluster,
		Namespace:               namespace,
		LogLines:                logLines,
		SlurmConfigMaps:         make(map[string]map[string]string),
		MountedConfigs:          make(map[string]string),
		ControllerContainerLogs: make(map[string]string),
		LoginContainerLogs:      make(map[string]map[string]string),
		WorkerContainerLogs:     make(map[string]map[string]string),
		OperatorLogs:            make(map[string]string),
		TopographLogs:           make(map[string]string),
		HelmReleases:            make(map[string]*HelmRelease),
		GPUInfo:                 make(map[string]*GPUDiagnostics),
		NetworkInfo:             make(map[string]*NetworkDiagnostics),
		SystemInfo:              make(map[string]*SystemDiagnostics),
		Errors:                  []string{},
	}
}

// GPUDiagnostics contains GPU-related debug info from a worker pod.
type GPUDiagnostics struct {
	NvidiaSmiQuery string
	NvidiaSmiCSV   string
	GPUTopology     string
	NVLinkStatus    string
	NVLinkCounters  string
	PersistenceMode string
}

// NetworkDiagnostics contains network-related debug info from a worker pod.
type NetworkDiagnostics struct {
	IBStat     string
	IBStatus   string
	IBVDevinfo string
	NCCLEnv    string
}

// SystemDiagnostics contains system-level debug info from a worker pod.
type SystemDiagnostics struct {
	CPUInfo   string
	MemInfo   string
	Lscpu     string
	Free      string
	DiskFree  string
	Iostat    string
	Ulimit    string
	IPAddr    string
	IPLink    string
	Hosts     string
	Mounts    string
	Uptime    string
	LoadAvg   string
	Dmesg     string
	SysctlNet string
}

// HelmRelease contains status information for a Helm release.
type HelmRelease struct {
	Name      string
	Namespace string
	Status    string
	History   string
	Values    string
	Manifest  string
}

// HelmReleaseConfig defines a Helm release to collect status for.
type HelmReleaseConfig struct {
	Namespace string
	Name      string
}
