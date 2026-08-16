/*
Copyright 2025 The KubeOne Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package addons

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"sort"
	"strings"
	"text/template"

	helmchartv2 "helm.sh/helm/v4/pkg/chart/v2"

	embeddedaddons "k8c.io/kubeone/addons"
	"k8c.io/kubeone/pkg/addons/helmchart"
	kubeoneapi "k8c.io/kubeone/pkg/apis/kubeone"
	"k8c.io/kubeone/pkg/fail"
	"k8c.io/kubeone/pkg/state"
)

// HelmFuncMap returns the template functions to attach to Helm's template
// engine (action.Configuration.CustomTemplateFuncs) so that addons render
// identically whether they are applied with kubectl or with Helm.
func HelmFuncMap(s *state.State) template.FuncMap {
	var (
		overwriteRegistry string
		caBundle          string
	)

	if s.Cluster != nil {
		overwriteRegistry = s.Cluster.RegistryConfiguration.ImageRegistry("")
		caBundle = s.Cluster.CertificateAuthority.Bundle
	}

	funcs := helmchart.TxtFuncMap(overwriteRegistry)
	funcs["CABundle"] = func() string {
		return caBundle
	}

	images := &internalImages{
		pauseImage: s.PauseImage,
		resolver:   s.Images.Get,
	}
	funcs["getImage"] = func(name string) (string, error) {
		return images.Get(name)
	}

	return funcs
}

// addonTemplateData merges the given addon params into the template data and
// resolves environment variable references, mirroring the logic used when
// applying an addon with kubectl.
func (a *applier) addonTemplateData(k1cluster *kubeoneapi.KubeOneCluster, addonName string, addonParams map[string]string) (templateData, error) {
	tplDataParams := map[string]string{}
	maps.Copy(tplDataParams, a.TemplateData.Params)
	maps.Copy(tplDataParams, addonParams)

	defaultAddonParams(k1cluster, addonName, tplDataParams)

	for k, v := range tplDataParams {
		if envName, ok := strings.CutPrefix(v, ParamsEnvPrefix); ok {
			if env, ok := os.LookupEnv(envName); ok {
				tplDataParams[k] = env
			} else {
				return templateData{}, fail.RuntimeError{
					Op:  "resolving template data environment variables",
					Err: fmt.Errorf("%q not found", envName),
				}
			}
		}
	}

	tplData := a.TemplateData
	tplData.Params = tplDataParams

	return tplData, nil
}

// Values returns the template data as a Helm values map.
func (d templateData) Values() map[string]any {
	return map[string]any{
		"config":                                   d.Config,
		"params":                                   d.Params,
		"certificates":                             d.Certificates,
		"credentials":                              d.Credentials,
		"customCredentials":                        d.CustomCredentials,
		"credentialsCCM":                           d.CredentialsCCM,
		"credentialsCCMHash":                       d.CredentialsCCMHash,
		"ccmClusterName":                           d.CCMClusterName,
		"calicoIptablesBackend":                    d.CalicoIptablesBackend,
		"deployCSIAddon":                           d.DeployCSIAddon,
		"snapshotterWebhookFailurePolicy":          d.SnapshotterWebhookFailurePolicy,
		"machineControllerCredentialsEnvVars":      d.MachineControllerCredentialsEnvVars,
		"machineControllerCredentialsHash":         d.MachineControllerCredentialsHash,
		"operatingSystemManagerEnabled":            d.OperatingSystemManagerEnabled,
		"operatingSystemManagerCredentialsEnvVars": d.OperatingSystemManagerCredentialsEnvVars,
		"operatingSystemManagerCredentialsHash":    d.OperatingSystemManagerCredentialsHash,
		"registryCredentials":                      d.RegistryCredentials,
		"resources":                                d.Resources,
	}
}

// AddonValues builds the Helm values for the given addon.
func AddonValues(s *state.State, addonName string) (map[string]any, error) {
	applier, err := newAddonsApplier(s)
	if err != nil {
		return nil, err
	}

	addonParams := addonParamsFor(s, addonName)

	tplData, err := applier.addonTemplateData(s.Cluster, addonName, addonParams)
	if err != nil {
		return nil, err
	}

	return tplData.Values(), nil
}

// Chart builds an in-memory Helm chart for the given addon. The addon is
// resolved from the user-provided addons directory first, falling back to the
// embedded addons. Unlike AddonValues, building a chart does not require a live
// cluster.
func Chart(s *state.State, addonName string) (*helmchartv2.Chart, error) {
	fsys, err := addonFS(s, addonName)
	if err != nil {
		return nil, err
	}

	manifests, err := helmchart.ReadManifests(fsys, addonName)
	if err != nil {
		return nil, err
	}

	if len(manifests) == 0 {
		return nil, fail.RuntimeError{
			Op:  fmt.Sprintf("converting %q addon", addonName),
			Err: fmt.Errorf("addon does not exist"),
		}
	}

	return helmchart.BuildChart(addonName, manifests)
}

// addonFS resolves the filesystem for the given addon, preferring the
// user-provided addons directory over the embedded addons.
func addonFS(s *state.State, addonName string) (fs.FS, error) {
	localFS, err := addonsLocalFS(s.Cluster.Addons, s.ManifestFilePath)
	if err != nil {
		return nil, err
	}

	if localFS != nil {
		entries, err := fs.ReadDir(localFS, ".")
		if err != nil {
			return nil, fail.Runtime(err, "reading local addons directory")
		}

		for _, entry := range entries {
			if entry.IsDir() && entry.Name() == addonName {
				return localFS, nil
			}
		}
	}

	return embeddedaddons.FS, nil
}

func addonParamsFor(s *state.State, addonName string) map[string]string {
	if s.Cluster.Addons == nil || !s.Cluster.Addons.Enabled() {
		return nil
	}

	for _, addon := range s.Cluster.Addons.DeclaredAddonsOnly() {
		if addon.Name == addonName {
			return addon.Params
		}
	}

	return nil
}

// AddonAction describes an addon to deploy along with an optional migration
// that must run before it is applied.
type AddonAction struct {
	Name string

	run func() error
}

// Run executes the addon migration, if any.
func (a AddonAction) Run() error {
	if a.run != nil {
		return a.run()
	}

	return nil
}

// DeployableAddons returns the embedded addons to deploy, in order.
func DeployableAddons(s *state.State) []AddonAction {
	actions := collectAddons(s)

	out := make([]AddonAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, AddonAction{Name: action.name, run: action.supportFn})
	}

	return out
}

// DeletableAddons returns the names of embedded addons that must be removed
// because their corresponding feature was disabled.
func DeletableAddons(s *state.State) []string {
	return addonsToDelete(s)
}

// UserAddonsToDeploy returns the names of user-provided addons to deploy.
func UserAddonsToDeploy(s *state.State) ([]string, error) {
	applier, err := newAddonsApplier(s)
	if err != nil {
		return nil, err
	}

	combined := map[string]string{}

	if applier.LocalFS != nil {
		customAddons, err := fs.ReadDir(applier.LocalFS, ".")
		if err != nil {
			return nil, fail.Runtime(err, "reading local addons directory")
		}

		for _, addon := range customAddons {
			if !addon.IsDir() {
				continue
			}

			if _, ok := embeddedAddons[addon.Name()]; ok {
				continue
			}

			combined[addon.Name()] = ""
		}
	}

	for _, addon := range s.Cluster.Addons.DeclaredAddonsOnly() {
		if _, ok := embeddedAddons[addon.Name]; ok {
			continue
		}

		if addon.Delete {
			continue
		}

		combined[addon.Name] = ""
	}

	names := make([]string, 0, len(combined))
	for name := range combined {
		names = append(names, name)
	}

	sort.Strings(names)

	return names, nil
}

// UserAddonsToDelete returns the names of user-provided addons marked for
// deletion.
func UserAddonsToDelete(s *state.State) []string {
	var names []string

	if s.Cluster.Addons == nil || !s.Cluster.Addons.Enabled() {
		return names
	}

	for _, addon := range s.Cluster.Addons.DeclaredAddonsOnly() {
		if addon.Delete {
			names = append(names, addon.Name)
		}
	}

	return names
}

// EmbeddedAddonNames returns the names of all embedded addons.
func EmbeddedAddonNames() []string {
	entries, err := fs.ReadDir(embeddedaddons.FS, ".")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}

	sort.Strings(names)

	return names
}
