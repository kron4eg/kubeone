# Addons as Helm charts

**Status:** **Draft**
**Created:** 2026-08-15
**Last updated:** 2026-08-15
**Author:** Artiom Diomin ([@kron4eg](https://github.com/kron4eg))

## Abstract

Every KubeOne addon — embedded or user-provided — is currently a directory of YAML manifests rendered with Go
`text/template` and applied with `kubectl apply --prune`. This proposal turns each addon into a small ("micro") Helm
chart and lets KubeOne deploy those charts itself, replacing `kubectl apply --prune`.

The key insight that makes this cheap and reliable is that Helm's template engine *is* Go `text/template`, and
`action.Configuration` exposes a `CustomTemplateFuncs` field. KubeOne already renders addons with `text/template`, so we
can reuse the exact same template functions (sprig plus the KubeOne helpers) by attaching them to Helm, and only
translate the *root context* (`.Config`, `.Params`, … → `.Values.*`).

## Goals

* Generate a standard Helm chart (`Chart.yaml` + `values.yaml` + `templates/`) from every addon.
* Deploy addons with KubeOne itself via Helm, replacing `kubectl apply --prune`.
* Preserve byte-for-byte identical rendering: an addon rendered as a Helm chart must equal the addon rendered and
  applied today.
* Make the conversion available to end users as a CLI subcommand.
* Adopt existing resources on upgrade, so clusters that previously had addons applied with `kubectl` migrate seamlessly
  to Helm ownership.

## Non-goals

* Changing the public `kubeone.k8s.io` API. This is deliberately an internal, opt-in mechanism gated by an environment
  variable; no new manifest fields.
* Translating template *functions* to Helm built-ins. Functions are supplied at runtime via `CustomTemplateFuncs`.
* Full standalone `helm install` portability. A generated chart is best rendered by KubeOne, which provides the
  KubeOne-specific template functions.

## Implementation

### 1. Conversion (`pkg/addons/helmchart`)

A self-contained converter turns an addon directory into a Helm chart:

* `translate.go` parses each manifest with `text/template/parse` and rewrites only the root context fields: `.Config` →
  `.Values.config`, `.Params` → `.Values.params`, `.Certificates`/`.Credentials`/`.CredentialsCCM`/ `.Resources`/etc. →
  their `.Values` counterparts, and `.InternalImages.Get "X"` → `getImage "X"`. All template functions are left
  untouched.
* `funcs.go` holds `TxtFuncMap` (sprig + `Registry`, `required`, `caBundle*`, `EquinixMetalSecret`,
  `vSphereCSIWebhookConfig`, `CABundle`), moved from `pkg/addons/manifest.go` so both the kubectl path and the Helm path
  share it.
* `chart.go` builds/loads/writes a `helm.sh/helm/v4` chart (`BuildChart`, `LoadChart` via `loader.LoadFiles`, `Write`).

The mapping between the `templateData` fields and their `.Values` keys is the single table in `schema.go`;
`templateData.Values()` in `pkg/addons` produces the matching runtime values.

### 2. Pre-generated charts

`hack/gen-addon-charts` regenerates `charts/<addon>/{Chart.yaml,values.yaml, templates/*}` from the embedded addons and
is wired into `hack/update-codegen.sh` (so `verify-codegen` catches drift). The charts are embedded via
`charts/charts.go` (`//go:embed *`). `values.yaml` is intentionally documentation-only: KubeOne injects the full render
context (config, params, certificates, credentials, …) at deploy time, which avoids any `CoalesceValues` type-conflicts
with typed values such as the `*KubeOneCluster` struct.

### 3. Deployment

`localhelm.DeployAddons` (`pkg/localhelm/addons.go`) is the Helm counterpart of `addons.Ensure` +
`addons.EnsureUserAddons`:

* selects addons with the existing `collectAddons` logic plus user addons,
* loads each chart from `charts.FS`,
* builds values with `addons.AddonValues` and attaches `addons.HelmFuncMap` via
  `action.Configuration.CustomTemplateFuncs`,
* runs `helm install`/`helm upgrade` in the `kube-system` release namespace (release name = sanitized addon name), with
  `Install.TakeOwnership` to adopt resources previously applied with `kubectl apply`,
* uninstalls releases for addons that were disabled or marked `delete: true`.

The path is gated by the `KUBEONE_ADDONS_VIA_HELM=true` environment variable and selected via task predicates in
`pkg/tasks/tasks.go`; the kubectl path remains the default.

### 4. CLI

`kubeone addons chart <addon-name> [--output/-o DIR] [--force] [--all]` converts an embedded addon (by name) or a user
addon (by directory path) into a Helm chart, reusing the same converter.

## Known limitations / follow-ups

* Post-render Go mutations (CA-bundle injection and the secrets-store CSI volume injection in `pkg/addons/manifest.go`)
  are not yet applied on the Helm path; they should move to an `Install.PostRenderer`.
* `AddonValues` rebuilds the addon applier (certificates/credentials) per addon; it can be cached.
* `disableTemplating: true` addons need literal `{{`/`}}` escaping in generated templates.
