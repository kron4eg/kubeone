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
	"bytes"
	"context"
	"testing"
	"text/template"

	"k8c.io/kubeone/pkg/addons/helmchart"
	kubeoneapi "k8c.io/kubeone/pkg/apis/kubeone"
	"k8c.io/kubeone/pkg/state"
)

func TestTemplateDataValues(t *testing.T) {
	td := templateData{
		Config:                &kubeoneapi.KubeOneCluster{},
		Params:                map[string]string{"IFACE": "en|eth1"},
		Certificates:          map[string]string{"MetricsServerCert": "cert"},
		Credentials:           map[string]string{"FOO": "bar"},
		CustomCredentials:     map[string]string{"CUSTOM": "value"},
		CredentialsCCM:        map[string]string{"CCM": "value"},
		DeployCSIAddon:        true,
		RegistryCredentials:   []registryCredentialsContainer{{RegistryName: "docker.io"}},
		Resources:             map[string]string{"Webhook": "webhook"},
		CredentialsCCMHash:    "hash",
		CCMClusterName:        "cluster",
		CalicoIptablesBackend: "NFT",
	}

	values := td.Values()

	if values["config"] != td.Config {
		t.Fatalf("values[config] = %v, want %v", values["config"], td.Config)
	}

	params, ok := values["params"].(map[string]string)
	if !ok || params["IFACE"] != "en|eth1" {
		t.Fatalf("values[params] = %v", values["params"])
	}

	if values["deployCSIAddon"] != true {
		t.Fatalf("values[deployCSIAddon] = %v, want true", values["deployCSIAddon"])
	}

	// Ensure every mapped key has a corresponding entry in the values map.
	for _, key := range []string{
		"config", "params", "certificates", "credentials", "customCredentials",
		"credentialsCCM", "credentialsCCMHash", "ccmClusterName",
		"calicoIptablesBackend", "deployCSIAddon", "snapshotterWebhookFailurePolicy",
		"machineControllerCredentialsEnvVars", "machineControllerCredentialsHash",
		"operatingSystemManagerEnabled", "operatingSystemManagerCredentialsEnvVars",
		"operatingSystemManagerCredentialsHash", "registryCredentials", "resources",
	} {
		if _, ok := values[key]; !ok {
			t.Fatalf("values map is missing key %q", key)
		}
	}
}

func TestHelmFuncMap(t *testing.T) {
	s, err := state.New(context.Background())
	if err != nil {
		t.Fatalf("state.New() error = %v", err)
	}

	s.Cluster = &kubeoneapi.KubeOneCluster{
		Versions: kubeoneapi.VersionConfig{
			Kubernetes: "1.34.0",
		},
	}
	s.PauseImage = "registry.k8s.io/pause:3.10"

	funcs := HelmFuncMap(s)

	for _, name := range []string{"Registry", "required", "CABundle", "getImage", "caBundleEnvVar", "default"} {
		if _, ok := funcs[name]; !ok {
			t.Fatalf("func map is missing %q", name)
		}
	}

	getImage, ok := funcs["getImage"].(func(string) (string, error))
	if !ok {
		t.Fatal("getImage has an unexpected type")
	}

	pause, err := getImage("PauseImage")
	if err != nil {
		t.Fatalf("getImage(PauseImage) error = %v", err)
	}
	if pause != s.PauseImage {
		t.Fatalf("getImage(PauseImage) = %q, want %q", pause, s.PauseImage)
	}

	registry, ok := funcs["Registry"].(func(string) string)
	if !ok {
		t.Fatal("Registry has an unexpected type")
	}
	if got := registry("registry.k8s.io"); got != "registry.k8s.io" {
		t.Fatalf("Registry() = %q, want %q", got, "registry.k8s.io")
	}
}

func TestTranslateRoundTrip(t *testing.T) {
	s, err := state.New(context.Background())
	if err != nil {
		t.Fatalf("state.New() error = %v", err)
	}
	s.PauseImage = "registry.k8s.io/pause:3.10"

	td := templateData{
		Config: &kubeoneapi.KubeOneCluster{
			Name: "test-cluster",
			ClusterNetwork: kubeoneapi.ClusterNetworkConfig{
				ServiceSubnet: "10.240.16.0/20",
			},
		},
		Params:         map[string]string{"PARAM": "hello"},
		Certificates:   map[string]string{"MetricsServerCert": "CERT"},
		DeployCSIAddon: true,
		Resources:      map[string]string{"Webhook": "webhook"},
		InternalImages: &internalImages{pauseImage: s.PauseImage, resolver: s.Images.Get},
	}

	s.Cluster = td.Config

	funcs := HelmFuncMap(s)

	src := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Config.Name }}
data:
  serviceSubnet: "{{ .Config.ClusterNetwork.ServiceSubnet }}"
  param: "{{ .Params.PARAM }}"
  image: "{{ .InternalImages.Get "PauseImage" }}"
  registry: "{{ Registry "registry.k8s.io" }}/pause"
  cert: "{{ .Certificates.MetricsServerCert }}"
  resource: "{{ .Resources.Webhook }}"
{{ if .DeployCSIAddon }}  csi: "yes"
{{ end }}`

	translated, err := helmchart.Translate(src)
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}

	originalOut, err := renderTemplate(src, funcs, td)
	if err != nil {
		t.Fatalf("rendering original template: %v", err)
	}

	translatedOut, err := renderTemplate(translated, funcs, map[string]any{"Values": td.Values()})
	if err != nil {
		t.Fatalf("rendering translated template: %v", err)
	}

	if originalOut != translatedOut {
		t.Fatalf("translated template output differs.\noriginal:\n%s\ntranslated:\n%s", originalOut, translatedOut)
	}
}

func renderTemplate(src string, funcs template.FuncMap, data any) (string, error) {
	tpl, err := template.New("test").Funcs(funcs).Parse(src)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
