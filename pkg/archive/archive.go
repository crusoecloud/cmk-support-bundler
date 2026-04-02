package archive

import (
	"archive/zip"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gitlab.com/crusoeenergy/island/external/cmk-support-bundler/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// operatorHelmKey and topographHelmKey are derived from the namespace config
// to avoid hardcoding namespace strings in the archive layout.
var operatorHelmKey, topographHelmKey string

func init() {
	for _, rel := range common.AllHelmReleases() {
		key := fmt.Sprintf("%s/%s", rel.Namespace, rel.Name)
		switch rel.Name {
		case "slurm-operator":
			operatorHelmKey = key
		case "topograph":
			topographHelmKey = key
		}
	}
}

// errorMatch represents a single error match found in a file.
type errorMatch struct {
	File    string
	Line    int
	Content string
}

// errorCollector collects error matches across all files.
type errorCollector struct {
	matches []errorMatch
	pattern *regexp.Regexp
}

func newErrorCollector() *errorCollector {
	return &errorCollector{
		matches: []errorMatch{},
		pattern: regexp.MustCompile(`(?i)(error|fatal|failed|failure|panic|exception|critical|cannot|unable to|denied|refused|timeout|timed out)`),
	}
}

func (ec *errorCollector) scan(filename, content string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		if ec.pattern.MatchString(line) {
			ec.matches = append(ec.matches, errorMatch{File: filename, Line: lineNum, Content: line})
		}
	}
}

func (ec *errorCollector) summary() string {
	if len(ec.matches) == 0 {
		return "No errors, failures, or fatal messages found.\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d error/warning lines across all files:\n", len(ec.matches)))
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	fileMatches := make(map[string][]errorMatch)
	for _, m := range ec.matches {
		fileMatches[m.File] = append(fileMatches[m.File], m)
	}

	for file, matches := range fileMatches {
		sb.WriteString(fmt.Sprintf("### %s (%d matches)\n", file, len(matches)))
		for _, m := range matches {
			sb.WriteString(fmt.Sprintf("  L%d: %s\n", m.Line, m.Content))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// zipHelper wraps zip.Writer to eliminate boilerplate.
type zipHelper struct {
	*zip.Writer
	ec *errorCollector
}

// write handles empty checks, YAML marshaling, and optional error scanning.
func (z *zipHelper) write(filename string, data interface{}, scan bool) error {
	if data == nil {
		return nil
	}

	var b []byte
	var err error

	switch v := data.(type) {
	case string:
		if v == "" {
			return nil
		}
		b = []byte(v)
	case []byte:
		if len(v) == 0 {
			return nil
		}
		b = v
	default:
		b, err = yaml.Marshal(v)
		if err != nil || string(b) == "null\n" || string(b) == "[]\n" {
			return nil
		}
	}

	writer, err := z.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create zip entry %s: %w", filename, err)
	}
	if _, err = writer.Write(b); err != nil {
		return fmt.Errorf("failed to write zip entry %s: %w", filename, err)
	}
	if scan {
		z.ec.scan(filename, string(b))
	}
	return nil
}

// writeHelm writes Helm release files to the zip archive.
func (z *zipHelper) writeHelm(baseDir string, release *common.HelmRelease) error {
	if release == nil {
		return nil
	}
	_ = z.write(filepath.Join(baseDir, "status.txt"), release.Status, false)
	_ = z.write(filepath.Join(baseDir, "history.txt"), release.History, false)
	_ = z.write(filepath.Join(baseDir, "values.yaml"), release.Values, false)
	return z.write(filepath.Join(baseDir, "manifest.yaml"), release.Manifest, false)
}

// extractObjects plucks raw maps out of Unstructured lists for YAML marshaling.
func extractObjects(items []unstructured.Unstructured) []interface{} {
	if len(items) == 0 {
		return nil
	}
	var out []interface{}
	for _, item := range items {
		out = append(out, item.Object)
	}
	return out
}

// sanitizePod returns a copy of the pod with Env and EnvFrom stripped from all containers.
func sanitizePod(pod *corev1.Pod) corev1.Pod {
	sanitized := pod.DeepCopy()
	for i := range sanitized.Spec.InitContainers {
		sanitized.Spec.InitContainers[i].Env = nil
		sanitized.Spec.InitContainers[i].EnvFrom = nil
	}
	for i := range sanitized.Spec.Containers {
		sanitized.Spec.Containers[i].Env = nil
		sanitized.Spec.Containers[i].EnvFrom = nil
	}
	return *sanitized
}

// writePod writes a sanitized pod to the zip archive.
func (z *zipHelper) writePod(filename string, pod interface{}) error {
	switch p := pod.(type) {
	case *corev1.Pod:
		if p == nil {
			return nil
		}
		sanitized := sanitizePod(p)
		return z.write(filename, sanitized, false)
	case corev1.Pod:
		sanitized := sanitizePod(&p)
		return z.write(filename, sanitized, false)
	default:
		return z.write(filename, pod, false)
	}
}

// sanitizeUnstructured deep-copies an unstructured object map and removes
// the specified fields from the spec section.
func sanitizeUnstructured(obj map[string]interface{}, fieldsToRemove []string) map[string]interface{} {
	// Deep copy via JSON round-trip
	data, err := yaml.Marshal(obj)
	if err != nil {
		return obj
	}
	var copy map[string]interface{}
	if err := yaml.Unmarshal(data, &copy); err != nil {
		return obj
	}

	// Remove fields from spec
	if spec, ok := copy["spec"].(map[string]interface{}); ok {
		for _, field := range fieldsToRemove {
			delete(spec, field)
		}
	}
	return copy
}

// WriteZip writes the debug report to a zip file with component-centric organization.
func WriteZip(outputPath string, dr *common.DebugReport) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	z := &zipHelper{
		Writer: zip.NewWriter(file),
		ec:     newErrorCollector(),
	}
	defer z.Close()

	// Metadata
	_ = z.write("metadata.yaml", map[string]interface{}{
		"timestamp": dr.Timestamp, "clusterName": dr.ClusterName, "namespace": dr.Namespace, "logLines": dr.LogLines,
	}, false)

	// Cluster CRs
	if dr.SlurmCluster != nil {
		obj := sanitizeUnstructured(dr.SlurmCluster.Object, []string{"rootSSHPubKeys"})
		_ = z.write("cluster/slurmcluster.yaml", obj, false)
	}
	if dr.SlinkyController != nil {
		_ = z.write("cluster/slinky-controller.yaml", dr.SlinkyController.Object, false)
	}
	_ = z.write("cluster/slinky-loginsets.yaml", extractObjects(dr.SlinkyLoginSets), false)
	_ = z.write("cluster/slinky-nodesets.yaml", extractObjects(dr.SlinkyNodeSets), false)

	// Operator
	for key, logs := range dr.OperatorLogs {
		_ = z.write(fmt.Sprintf("components/operator/logs/%s.log", strings.ReplaceAll(key, "/", "_")), logs, true)
	}
	_ = z.writeHelm("components/operator/helm", dr.HelmReleases[operatorHelmKey])

	// Topograph
	for _, pod := range dr.TopographPods {
		_ = z.writePod(fmt.Sprintf("components/topograph/pods/%s.yaml", pod.Name), pod)
	}
	for key, logs := range dr.TopographLogs {
		_ = z.write(fmt.Sprintf("components/topograph/logs/%s.log", strings.ReplaceAll(key, "/", "_")), logs, true)
	}
	_ = z.writeHelm("components/topograph/helm", dr.HelmReleases[topographHelmKey])

	// Controller
	_ = z.writePod("components/controller/pod.yaml", dr.ControllerPod)
	for container, logs := range dr.ControllerContainerLogs {
		_ = z.write(fmt.Sprintf("components/controller/logs/%s.log", container), logs, true)
	}
	for filename, content := range dr.MountedConfigs {
		_ = z.write(filepath.Join("components/controller/config", filename), content, false)
	}

	// Slurm status
	baseSlurm := "components/controller/slurm-status"
	_ = z.write(filepath.Join(baseSlurm, "sinfo.txt"), dr.Sinfo, true)
	_ = z.write(filepath.Join(baseSlurm, "sdiag.txt"), dr.Sdiag, false)
	_ = z.write(filepath.Join(baseSlurm, "scontrol-config.txt"), dr.ScontrolConfig, false)
	_ = z.write(filepath.Join(baseSlurm, "scontrol-nodes.txt"), dr.ScontrolNodes, false)
	_ = z.write(filepath.Join(baseSlurm, "scontrol-partitions.txt"), dr.ScontrolPartitions, false)

	// Login
	for _, pod := range dr.LoginPods {
		_ = z.writePod(fmt.Sprintf("components/login/pods/%s.yaml", pod.Name), pod)
	}
	for podName, cLogs := range dr.LoginContainerLogs {
		for cName, logs := range cLogs {
			_ = z.write(fmt.Sprintf("components/login/logs/%s/%s.log", podName, cName), logs, true)
		}
	}

	// Workers
	for _, pod := range dr.WorkerPods {
		_ = z.writePod(fmt.Sprintf("components/workers/pods/%s.yaml", pod.Name), pod)
	}
	for podName, cLogs := range dr.WorkerContainerLogs {
		for cName, logs := range cLogs {
			_ = z.write(fmt.Sprintf("components/workers/logs/%s/%s.log", podName, cName), logs, true)
		}
	}

	// GPU diagnostics
	for worker, gpu := range dr.GPUInfo {
		base := filepath.Join("components/workers/gpu", worker)
		_ = z.write(filepath.Join(base, "nvidia-smi-q.txt"), gpu.NvidiaSmiQuery, false)
		_ = z.write(filepath.Join(base, "nvidia-metrics.csv"), gpu.NvidiaSmiCSV, false)
		_ = z.write(filepath.Join(base, "persistence-mode.txt"), gpu.PersistenceMode, false)
		_ = z.write(filepath.Join(base, "nvidia-topo.txt"), gpu.GPUTopology, false)
		_ = z.write(filepath.Join(base, "nvlink-status.txt"), gpu.NVLinkStatus, false)
		_ = z.write(filepath.Join(base, "nvlink-counters.txt"), gpu.NVLinkCounters, false)
	}

	// Network diagnostics
	for worker, net := range dr.NetworkInfo {
		base := filepath.Join("components/workers/network", worker)
		_ = z.write(filepath.Join(base, "ibstat.txt"), net.IBStat, false)
		_ = z.write(filepath.Join(base, "ibstatus.txt"), net.IBStatus, false)
		_ = z.write(filepath.Join(base, "ibv_devinfo.txt"), net.IBVDevinfo, false)
		_ = z.write(filepath.Join(base, "nccl-env.txt"), net.NCCLEnv, false)
	}

	// System diagnostics
	for worker, sys := range dr.SystemInfo {
		base := filepath.Join("components/workers/system", worker)
		_ = z.write(filepath.Join(base, "cpuinfo.txt"), sys.CPUInfo, false)
		_ = z.write(filepath.Join(base, "meminfo.txt"), sys.MemInfo, false)
		_ = z.write(filepath.Join(base, "lscpu.txt"), sys.Lscpu, false)
		_ = z.write(filepath.Join(base, "free.txt"), sys.Free, false)
		_ = z.write(filepath.Join(base, "df.txt"), sys.DiskFree, false)
		_ = z.write(filepath.Join(base, "iostat.txt"), sys.Iostat, false)
		_ = z.write(filepath.Join(base, "ulimit.txt"), sys.Ulimit, false)
		_ = z.write(filepath.Join(base, "ip-addr.txt"), sys.IPAddr, false)
		_ = z.write(filepath.Join(base, "ip-link.txt"), sys.IPLink, false)
		_ = z.write(filepath.Join(base, "hosts.txt"), sys.Hosts, false)
		_ = z.write(filepath.Join(base, "mounts.txt"), sys.Mounts, false)
		_ = z.write(filepath.Join(base, "uptime.txt"), sys.Uptime, false)
		_ = z.write(filepath.Join(base, "loadavg.txt"), sys.LoadAvg, false)
		_ = z.write(filepath.Join(base, "sysctl-net.txt"), sys.SysctlNet, false)
		_ = z.write(filepath.Join(base, "dmesg.txt"), sys.Dmesg, true)
	}

	// ConfigMaps
	for cmName, cmData := range dr.SlurmConfigMaps {
		base := filepath.Join("configmaps", strings.ReplaceAll(cmName, "/", "_"))
		for key, value := range cmData {
			_ = z.write(filepath.Join(base, strings.ReplaceAll(key, "/", "_")), value, false)
		}
	}

	// Kubernetes resources
	_ = z.write("kubernetes/services.yaml", dr.ServicesDescribe, false)
	_ = z.write("kubernetes/endpoints.yaml", dr.EndpointsDescribe, false)
	_ = z.write("kubernetes/nodes.yaml", dr.NodesDescribe, false)
	_ = z.write("kubernetes/pvcs.yaml", dr.PVCsDescribe, false)
	_ = z.write("kubernetes/all-pods.txt", dr.AllPodsListing, false)
	_ = z.write("kubernetes/all-nodes.txt", dr.AllNodesListing, false)
	_ = z.write("kubernetes/events.txt", dr.EventsDescribe, true)

	// Other Helm releases (not operator or topograph)
	for key, release := range dr.HelmReleases {
		if key != operatorHelmKey && key != topographHelmKey {
			_ = z.writeHelm(filepath.Join("helm", strings.ReplaceAll(key, "/", "_")), release)
		}
	}

	// Collection errors
	if len(dr.Errors) > 0 {
		_ = z.write("collection-errors.txt", strings.Join(dr.Errors, "\n"), false)
	}

	// Error summary
	_ = z.write("error-summary.txt", z.ec.summary(), false)

	return nil
}
