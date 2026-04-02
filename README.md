# CMK Support Bundler - Slurm Cluster Diagnostics

Collects diagnostic information from your CMK Slurm cluster and packages it into a zip archive for troubleshooting.

## Usage

```bash
curl -sL https://raw.githubusercontent.com/crusoecloud/cmk-support-bundler/main/collect.sh | bash
```

The script will prompt you for the namespace where your SlurmCluster is deployed, then handle everything automatically. You can also pass the namespace directly:

```bash
curl -sL https://raw.githubusercontent.com/crusoecloud/cmk-support-bundler/main/collect.sh | bash -s -- <namespace>
```

## What is collected

| Category | Details |
|----------|---------|
| Slurm CRs | SlurmCluster, Slinky Controller/LoginSets/NodeSets |
| Slurm status | sinfo, scontrol (config, nodes, partitions), sdiag |
| Configuration | Slurm-related ConfigMaps and mounted `/etc/slurm/*` files |
| Logs | Container logs for controller, login, and worker pods; operator and topograph logs |
| GPU | nvidia-smi hardware queries, GPU topology, NVLink status, Xid errors |
| Network | InfiniBand status (ibstat, ibstatus, ibv_devinfo), NCCL environment |
| System | CPU, memory, disk, mounts, network interfaces, dmesg, sysctl |
| Kubernetes | Services, endpoints, nodes, PVCs, events, pod listings |
| Helm | Status, history, values, and manifests for operator Helm releases |

## What is NOT collected

- Secret contents or names
- Customer workloads or job data (squeue, job queues, process lists)
- Environment variables from pods
- Deployments, StatefulSets, DaemonSets
- GPU process listings (nvidia-smi process table is stripped)
