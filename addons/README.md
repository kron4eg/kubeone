# Addons

This directory contains various, commonly-used KubeOne Addons. For more details about
how to use the KubeOne Addons, consider the [KubeOne documentation][addons-docs].

## Addons as Helm charts

Every addon can be converted into a standalone Helm chart with the
`addons chart` subcommand:

```bash
# Convert an embedded addon
kubeone -m kubeone.yaml addons chart cluster-autoscaler

# Convert every embedded addon
kubeone -m kubeone.yaml addons chart --all --output ./charts

# Convert a user-defined addon directory
kubeone -m kubeone.yaml addons chart ./my-addons/my-monitoring
```

Each generated chart contains a `Chart.yaml`, a documentation-only `values.yaml`
and a `templates/` directory. KubeOne supplies the template functions and the
runtime values (`config`, `params`, `certificates`, `credentials`, …), so charts
are best rendered by KubeOne itself.

To deploy addons as Helm releases instead of applying them with
`kubectl apply --prune`, set the `KUBEONE_ADDONS_VIA_HELM=true` environment
variable when running `apply`/`upgrade`. See the
[design proposal][helm-charts-proposal] for details.

## Available Addons

- [Cluster backups (with Restic)][backups-addon]
- [Cluster Autoscaler][cluster-autoscaler]
- [Default StorageClass][default-storage-class]
- [Unattended OS Upgrades][unattended-upgrades]

[addons-docs]: https://docs.kubermatic.com/kubeone/main/guides/addons/
[backups-addon]: ./backups-restic/README.md
[cluster-autoscaler]: ./cluster-autoscaler/README.md
[default-storage-class]: ./default-storage-class
[unattended-upgrades]: ./unattended-upgrades/README.md
[helm-charts-proposal]: ../docs/proposals/20260815-addons-as-helm-charts.md

