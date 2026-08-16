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

// Package helmchart converts KubeOne addon manifests into Helm charts.
package helmchart

import (
	"encoding/json"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig/v3"
	"github.com/pkg/errors"

	"k8c.io/kubeone/pkg/certificate/cabundle"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// TxtFuncMap returns the template functions available to KubeOne addon
// manifests (sprig plus KubeOne-specific helpers). It is attached to Helm's
// template engine via action.Configuration.CustomTemplateFuncs so that addons
// render identically whether they are applied with kubectl or with Helm.
func TxtFuncMap(overwriteRegistry string) template.FuncMap {
	funcs := sprig.TxtFuncMap()

	funcs["Registry"] = func(registry string) string {
		if overwriteRegistry != "" {
			return overwriteRegistry
		}

		return registry
	}

	funcs["required"] = requiredTemplateFunc
	funcs["caBundleEnvVar"] = caBundleEnvVarTemplateFunc
	funcs["caBundleVolume"] = caBundleVolumeTemplateFunc
	funcs["caBundleVolumeMount"] = caBundleVolumeMountTemplateFunc
	funcs["EquinixMetalSecret"] = equinixMetalSecretTemplateFunc
	funcs["vSphereCSIWebhookConfig"] = vSphereCSIWebhookConfigTemplateFunc

	// CABundle is a cluster-specific function; a stub is registered here so
	// that templates using it can be parsed. Callers are expected to override
	// it with the actual CA bundle.
	funcs["CABundle"] = func() string {
		return ""
	}

	return funcs
}

func requiredTemplateFunc(warn string, input any) (any, error) {
	switch val := input.(type) {
	case nil:
		return val, errors.New(warn)
	case string:
		if val == "" {
			return val, errors.New(warn)
		}
	}

	return input, nil
}

func caBundleEnvVarTemplateFunc() (string, error) {
	buf, err := yaml.Marshal([]corev1.EnvVar{cabundle.EnvVar()})

	return string(buf), err
}

func caBundleVolumeTemplateFunc() (string, error) {
	buf, err := yaml.Marshal([]corev1.Volume{cabundle.Volume()})

	return string(buf), err
}

func caBundleVolumeMountTemplateFunc() (string, error) {
	buf, err := yaml.Marshal([]corev1.VolumeMount{cabundle.VolumeMount()})

	return string(buf), err
}

func equinixMetalSecretTemplateFunc(apiKey, projectID string) (string, error) {
	equinixMetalSecret := struct {
		APIKey    string `json:"apiKey"`
		ProjectID string `json:"projectID"`
	}{
		APIKey:    apiKey,
		ProjectID: projectID,
	}

	buf, err := json.Marshal(equinixMetalSecret)

	return string(buf), err
}

func vSphereCSIWebhookConfigTemplateFunc() (string, error) {
	cfg := vsphereCSIWebhookConfigWrapper{
		WebHookConfig: vsphereCSIWebhookConfig{
			Port:     "8443",
			CertFile: "/run/secrets/tls/cert.pem",
			KeyFile:  "/run/secrets/tls/key.pem",
		},
	}

	var buf strings.Builder
	enc := toml.NewEncoder(&buf)
	enc.Indent = ""
	err := enc.Encode(cfg)

	return buf.String(), err
}

type vsphereCSIWebhookConfig struct {
	Port     string `toml:"port"`
	CertFile string `toml:"cert-file"`
	KeyFile  string `toml:"key-file"`
}

type vsphereCSIWebhookConfigWrapper struct {
	WebHookConfig vsphereCSIWebhookConfig `toml:"WebHookConfig"`
}
