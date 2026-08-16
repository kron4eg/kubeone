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

package localhelm

import (
	"context"
	"errors"
	"os"
	"strings"

	helmaction "helm.sh/helm/v4/pkg/action"
	helmchartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/kube"
	helmrelease "helm.sh/helm/v4/pkg/release"
	"helm.sh/helm/v4/pkg/storage/driver"

	"k8c.io/kubeone/charts"
	"k8c.io/kubeone/pkg/addons"
	"k8c.io/kubeone/pkg/addons/helmchart"
	"k8c.io/kubeone/pkg/fail"
	"k8c.io/kubeone/pkg/kubeconfig"
	"k8c.io/kubeone/pkg/state"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// addonsViaHelmEnv enables deploying addons as Helm releases instead of
	// applying them with kubectl.
	addonsViaHelmEnv = "KUBEONE_ADDONS_VIA_HELM"

	// addonsReleaseNamespace is the namespace used for the Helm releases that
	// back the KubeOne addons.
	addonsReleaseNamespace = "kube-system"
)

// AddonsViaHelm reports whether addons should be deployed as Helm releases.
func AddonsViaHelm() bool {
	v := os.Getenv(addonsViaHelmEnv)

	return v == "true" || v == "1"
}

// DeployAddons deploys the embedded and user-provided addons as Helm releases.
// It is the Helm-based counterpart of addons.Ensure and addons.EnsureUserAddons.
func DeployAddons(st *state.State) error {
	konfigBuf, err := kubeconfig.Download(st)
	if err != nil {
		return err
	}

	tmpKubeConf, err := os.CreateTemp("", "kubeone-addons-kubeconfig-*")
	if err != nil {
		return fail.Runtime(err, "creating temp file for addons kubeconfig")
	}
	defer func() {
		name := tmpKubeConf.Name()
		tmpKubeConf.Close()
		os.Remove(name)
	}()

	n, err := tmpKubeConf.Write(konfigBuf)
	if err != nil {
		return fail.Runtime(err, "writing temp file for addons kubeconfig")
	}
	if n != len(konfigBuf) {
		return fail.NewRuntimeError("incorrect number of bytes written to temp addons kubeconfig", "")
	}

	helmSettings := newHelmSettings(st.Verbose)
	helmCfg, err := newActionConfiguration(helmSettings.Debug)
	if err != nil {
		return err
	}

	helmCfg.CustomTemplateFuncs = addons.HelmFuncMap(st)

	restClientGetter := newRestClientGetter(tmpKubeConf.Name(), addonsReleaseNamespace, st)
	err = helmCfg.Init(restClientGetter, addonsReleaseNamespace, helmStorageDriver)
	if err != nil {
		return fail.Runtime(err, "initializing helm action configuration for addons")
	}

	// Embedded addons, with their migrations.
	for _, action := range addons.DeployableAddons(st) {
		if err = action.Run(); err != nil {
			return err
		}

		if err = deployAddon(st, helmCfg, action.Name); err != nil {
			return err
		}
	}

	// User-provided addons.
	userAddons, err := addons.UserAddonsToDeploy(st)
	if err != nil {
		return err
	}

	for _, name := range userAddons {
		if err = deployAddon(st, helmCfg, name); err != nil {
			return err
		}
	}

	// Remove addons that were disabled or marked for deletion.
	for _, name := range addons.DeletableAddons(st) {
		if err = uninstallAddon(st, helmCfg, name); err != nil {
			return err
		}
	}

	for _, name := range addons.UserAddonsToDelete(st) {
		if err = uninstallAddon(st, helmCfg, name); err != nil {
			return err
		}
	}

	return nil
}

func deployAddon(st *state.State, cfg *helmaction.Configuration, addonName string) error {
	chart, err := helmchart.LoadChart(charts.FS, addonName)
	if err != nil {
		return err
	}

	values, err := addons.AddonValues(st, addonName)
	if err != nil {
		return err
	}

	releaseName := helmReleaseName(addonName)

	histClient := helmaction.NewHistory(cfg)
	histClient.Max = 1
	_, err = histClient.Run(releaseName)

	switch {
	case errors.Is(err, driver.ErrReleaseNotFound):
		st.Logger.Infof("Installing addon %q...", addonName)

		return installAddonRelease(st.Context, cfg, releaseName, chart, values, st)
	case err == nil:
		st.Logger.Infof("Upgrading addon %q...", addonName)

		return upgradeAddonRelease(st.Context, cfg, releaseName, chart, values, st)
	default:
		return fail.Runtime(err, "getting helm release history for addon %q", addonName)
	}
}

func installAddonRelease(ctx context.Context, cfg *helmaction.Configuration, releaseName string, chart *helmchartv2.Chart, values map[string]any, st *state.State) error {
	install := helmaction.NewInstall(cfg)
	install.CreateNamespace = true
	install.Namespace = addonsReleaseNamespace
	install.ReleaseName = releaseName
	install.WaitStrategy = kube.HookOnlyStrategy
	// Adopt resources that were previously applied with kubectl apply.
	install.TakeOwnership = true

	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return fail.Runtime(err, "installing addon release %q", releaseName)
	}

	relAcc, err := helmrelease.NewAccessor(rel)
	if err != nil {
		return fail.Runtime(err, "creating helm release accessor")
	}

	secretObjectKey := ctrlruntimeclient.ObjectKey{
		Name:      makeKey(relAcc.Name(), relAcc.Version()),
		Namespace: addonsReleaseNamespace,
	}

	return addReleaseSecretLabels(ctx, secretObjectKey, st.DynamicClient)
}

func upgradeAddonRelease(ctx context.Context, cfg *helmaction.Configuration, releaseName string, chart *helmchartv2.Chart, values map[string]any, st *state.State) error {
	upgrade := helmaction.NewUpgrade(cfg)
	upgrade.Install = true
	upgrade.ResetThenReuseValues = true
	upgrade.MaxHistory = 5
	upgrade.Namespace = addonsReleaseNamespace
	upgrade.WaitStrategy = kube.HookOnlyStrategy

	rel, err := upgrade.RunWithContext(ctx, releaseName, chart, values)
	if err != nil {
		return fail.Runtime(err, "upgrading addon release %q", releaseName)
	}

	relAcc, err := helmrelease.NewAccessor(rel)
	if err != nil {
		return fail.Runtime(err, "creating helm release accessor")
	}

	secretObjectKey := ctrlruntimeclient.ObjectKey{
		Name:      makeKey(relAcc.Name(), relAcc.Version()),
		Namespace: addonsReleaseNamespace,
	}

	return addReleaseSecretLabels(ctx, secretObjectKey, st.DynamicClient)
}

func uninstallAddon(st *state.State, cfg *helmaction.Configuration, addonName string) error {
	releaseName := helmReleaseName(addonName)

	uninstall := helmaction.NewUninstall(cfg)
	uninstall.WaitStrategy = kube.HookOnlyStrategy

	_, err := uninstall.Run(releaseName)
	if errors.Is(err, driver.ErrReleaseNotFound) {
		return nil
	}
	if err != nil {
		return fail.Runtime(err, "uninstalling addon release %q", releaseName)
	}

	st.Logger.Infof("Uninstalled addon release %q", releaseName)

	return nil
}

// helmReleaseName converts an addon name into a valid Helm release name.
func helmReleaseName(name string) string {
	var b strings.Builder

	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	return strings.Trim(b.String(), "-")
}
