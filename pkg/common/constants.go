package common

// DefaultSlurmdContainer is the container name for slurmd in worker pods.
const DefaultSlurmdContainer = "slurmd"

// DefaultSlurmctldContainer is the container name for slurmctld in controller pods.
const DefaultSlurmctldContainer = "slurmctld"

// DefaultMaxLogLines is the default number of log lines to collect from containers.
const DefaultMaxLogLines = int64(5000)
