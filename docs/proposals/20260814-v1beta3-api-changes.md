# v1beta3 API changes: addons removal and cloud-provider cleanup

**Status:** **Draft**
**Created:** 2026-08-14
**Last updated:** 2026-08-14
**Author:** Artiom Diomin ([@kron4eg](https://github.com/kron4eg))

## Abstract

This proposal documents the breaking API changes applied to the `v1beta3` `KubeOneCluster` schema. It removes the
deprecated addons machinery in favor of Helm releases, drops the Equinix Metal provider and the WeaveNet CNI, moves the
cloud-config fields into their provider-specific specs, fixes the misspelled `CertificateAuthorithyConfig` type, and
introduces a few new capabilities (nftables kube-proxy mode, a typed webhook audit-log mode, and a global
`kubeletConfig` on the cluster level).

## API changes

### Addons removed, Helm releases only

The `Addon`, `AddonRef`, and `Addons` types and the `addons` field of `KubeOneCluster` are removed. KubeOne now
reconciles Helm releases exclusively via the `helmReleases` field (a list of `HelmRelease`). Any functionality that was
previously expressed as a named addon must be converted to a Helm release.

```yaml
# before
addons:
  addons:
    - addon:
        name: example

# after
helmReleases:
  - chart: example
    repoURL: https://charts.example.com
    namespace: default
```

### `CertificateAuthorithyConfig` renamed

The misspelled `CertificateAuthorithyConfig` type is renamed to `CertificateAuthorityConfig`. A deprecated type alias
`CertificateAuthorithyConfig = CertificateAuthorityConfig` is kept for Go API compatibility. The long-deprecated
`caBundle` field is removed; use `certificateAuthority.bundle` instead.

### Cloud-config fields moved into provider specs

The generic `cloudConfig` and `csiConfig` fields are removed from `CloudProviderSpec` and relocated into the
provider-specific specs:

| Provider | Fields |
| --- | --- |
| `aws` | `cloudConfig` |
| `azure` | `cloudConfig` |
| `openstack` | `cloudConfig` |
| `vsphere` | `cloudConfig`, `csiConfig` |

```yaml
# before
cloudProvider:
  cloudConfig: |
    ...
  vsphere: {}

# after
cloudProvider:
  vsphere:
    cloudConfig: |
      ...
```

### Equinix Metal provider removed

The `EquinixMetalSpec` type and the `equinixmetal` field of `CloudProviderSpec` are removed, together with the bundled
`ccm-equinixmetal` addon and its Terraform example. The Equinix Metal provider is no longer supported.

### WeaveNet CNI removed

The `WeaveNetSpec` type, the `weaveNet` field of `CNI`, and the bundled `cni-weavenet` addon are removed. Use `canal` or
`cilium` instead.

### New: nftables kube-proxy mode

`KubeProxyConfig` gains an `nftables` field (`*NFTables`), allowing the kube-proxy mode to be set to nftables. The
`skipInstallation`, `ipvs`, and `iptables` fields are also now optional (`omitempty`).

### New: typed webhook audit-log mode

`WebhookAuditLogConfig.Mode` changes from `string` to a new `WebhookMode` type with the constants `Batch`, `Blocking`,
and `BlockingStrict`.

### Serialization (`omitempty`/`omitzero`) cleanups

Many optional struct fields gain `omitempty,omitzero` tags so they are omitted from the serialized manifest when unset,
including `kubeletConfig`, `containerRuntime`, `clusterNetwork`, `proxy`, `staticWorkers`, `features`, `loggingConfig`,
`tlsCipherSuites`, `certificateAuthority` on the cluster level, and the nested
`kubelet`/`operatingSystem`/`nodeSettings` fields. `ControlPlaneConfig.Hosts` and `NodeSets` are now optional as well.
