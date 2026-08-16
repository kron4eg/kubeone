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

package helmchart

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	embeddedaddons "k8c.io/kubeone/addons"
)

func TestTranslate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "config field",
			in:   `{{ .Config.ClusterNetwork.ServiceSubnet }}`,
			want: `{{.Values.config.ClusterNetwork.ServiceSubnet}}`,
		},
		{
			name: "param field",
			in:   `{{ .Params.IFACE }}`,
			want: `{{.Values.params.IFACE}}`,
		},
		{
			name: "internal images get",
			in:   `{{ .InternalImages.Get "MachineController" }}`,
			want: `{{getImage "MachineController"}}`,
		},
		{
			name: "method call on config",
			in:   `{{ .Config.CloudProvider.CloudProviderName }}`,
			want: `{{.Values.config.CloudProvider.CloudProviderName}}`,
		},
		{
			name: "range over registry credentials",
			in:   `{{ range .RegistryCredentials }}{{ .RegistryName }}{{ end }}`,
			want: `{{range .Values.registryCredentials}}{{.RegistryName}}{{end}}`,
		},
		{
			name: "variable referencing params",
			in:   `{{ $x := .Params }}{{ $x.IFACE }}`,
			want: `{{$x := .Values.params}}{{$x.IFACE}}`,
		},
		{
			name: "resources map",
			in:   `{{ .Resources.MachineControllerWebhookName }}`,
			want: `{{.Values.resources.MachineControllerWebhookName}}`,
		},
		{
			name: "sprig funcs are left untouched",
			in:   `{{ default "fallback" .Params.OPTIONAL }}`,
			want: `{{default "fallback" .Values.params.OPTIONAL}}`,
		},
		{
			name: "registry func is left untouched",
			in:   `{{ Registry "registry.k8s.io" }}/pause:3.9`,
			want: `{{Registry "registry.k8s.io"}}/pause:3.9`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Translate(tt.in)
			if err != nil {
				t.Fatalf("Translate() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Translate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTranslateAllEmbeddedAddons(t *testing.T) {
	entries, err := fs.ReadDir(embeddedaddons.FS, ".")
	if err != nil {
		t.Fatalf("reading embedded addons: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			manifests, err := ReadManifests(embeddedaddons.FS, entry.Name())
			if err != nil {
				t.Fatalf("ReadManifests() error = %v", err)
			}

			for _, manifest := range manifests {
				if _, err := Translate(manifest.Data); err != nil {
					t.Errorf("Translate(%q) error = %v", manifest.Name, err)
				}
			}
		})
	}
}

func TestBuildAndLoadChartRoundTrip(t *testing.T) {
	chart, err := BuildChart("test-addon", []Manifest{
		{
			Name: "a.yaml",
			Data: "kind: ConfigMap\napiVersion: v1\nmetadata:\n  name: {{ .Params.NAME }}\n",
		},
	})
	if err != nil {
		t.Fatalf("BuildChart() error = %v", err)
	}

	dir := t.TempDir()
	err = Write(chart, dir)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	loaded, err := LoadChart(os.DirFS(dir), ".")
	if err != nil {
		t.Fatalf("LoadChart() error = %v", err)
	}

	if loaded.Metadata.Name != "test-addon" {
		t.Fatalf("chart name = %q, want %q", loaded.Metadata.Name, "test-addon")
	}

	if len(loaded.Templates) != 1 {
		t.Fatalf("templates count = %d, want 1", len(loaded.Templates))
	}

	if !strings.Contains(string(loaded.Templates[0].Data), ".Values.params.NAME") {
		t.Fatalf("template was not translated: %s", loaded.Templates[0].Data)
	}
}
