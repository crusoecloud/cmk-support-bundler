package common

// NamespaceConfig defines what data to collect from a namespace.
// This is the single source of truth for all namespace access.
// No other code should hardcode namespace strings.
type NamespaceConfig struct {
	Name         string   // namespace name
	Pods         bool     // include in pod listing scan
	PodPrefixes  []string // collect logs from pods matching these prefixes
	HelmReleases []string // helm release names to collect from this namespace
}

// AdditionalNamespaces defines all namespaces accessed beyond the SlurmCluster namespace.
var AdditionalNamespaces = []NamespaceConfig{
	{
		Name:         "slinky",
		Pods:         true,
		PodPrefixes:  []string{"slurm-operator", "topograph", "node-observer", "node-data-broker"},
		HelmReleases: []string{"slurm-operator", "topograph"},
	},
	{
		Name:         "cert-manager",
		Pods:         true,
		HelmReleases: []string{"cert-manager"},
	},
	{
		Name:        "nvidia-gpu-operator",
		Pods:        true,
	},
	{
		Name:        "nvidia-network-operator",
		Pods:        true,
	},
	{
		Name:        "crusoe-system",
		Pods:        true,
	},
	{
		Name:        "kube-system",
		Pods:        true,
	},
}

// PodListingNamespaces returns namespace names where pod listing is enabled.
func PodListingNamespaces() []string {
	var ns []string
	for _, cfg := range AdditionalNamespaces {
		if cfg.Pods {
			ns = append(ns, cfg.Name)
		}
	}
	return ns
}

// NamespacesForPrefixes returns the cluster namespace plus any additional
// namespaces whose PodPrefixes overlap with the given prefixes.
func NamespacesForPrefixes(clusterNamespace string, prefixes []string) []string {
	namespaces := []string{clusterNamespace}
	for _, cfg := range AdditionalNamespaces {
		if cfg.Name == clusterNamespace {
			continue
		}
		if cfg.hasAnyPrefix(prefixes) {
			namespaces = append(namespaces, cfg.Name)
		}
	}
	return namespaces
}

// AllHelmReleases returns all helm release configs derived from AdditionalNamespaces.
func AllHelmReleases() []HelmReleaseConfig {
	var releases []HelmReleaseConfig
	for _, cfg := range AdditionalNamespaces {
		for _, name := range cfg.HelmReleases {
			releases = append(releases, HelmReleaseConfig{
				Namespace: cfg.Name,
				Name:      name,
			})
		}
	}
	return releases
}

// AllNamespaceNames returns all additional namespace names.
func AllNamespaceNames() []string {
	var ns []string
	for _, cfg := range AdditionalNamespaces {
		ns = append(ns, cfg.Name)
	}
	return ns
}

func (nc NamespaceConfig) hasAnyPrefix(prefixes []string) bool {
	for _, cfgPrefix := range nc.PodPrefixes {
		for _, prefix := range prefixes {
			if cfgPrefix == prefix {
				return true
			}
		}
	}
	return false
}
