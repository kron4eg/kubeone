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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	helmchartcommon "helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/chart/loader/archive"
	helmchartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"

	"sigs.k8s.io/yaml"
)

const (
	chartVersion = "0.1.0"
	templatesDir = "templates"
)

// Manifest is a single addon manifest file.
type Manifest struct {
	// Name is the base file name, e.g. "cluster-autoscaler.yaml".
	Name string
	// Data is the raw (untranslated) template content.
	Data string
}

// ReadManifests lists the manifest files of an addon directory. Only
// .yaml, .yml and .json files are considered, mirroring the KubeOne addon
// loader.
func ReadManifests(fsys fs.FS, addonName string) ([]Manifest, error) {
	entries, err := fs.ReadDir(fsys, addonName)
	if err != nil {
		return nil, fmt.Errorf("reading addon directory %q: %w", addonName, err)
	}

	var manifests []Manifest
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".yaml", ".yml", ".json":
		default:
			continue
		}

		data, err := fs.ReadFile(fsys, filepath.Join(addonName, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading addon manifest %q: %w", entry.Name(), err)
		}

		manifests = append(manifests, Manifest{
			Name: entry.Name(),
			Data: string(data),
		})
	}

	return manifests, nil
}

// BuildChart builds an in-memory Helm chart from the given addon manifests.
func BuildChart(addonName string, manifests []Manifest) (*helmchartv2.Chart, error) {
	sort.Slice(manifests, func(i, j int) bool {
		return manifests[i].Name < manifests[j].Name
	})

	chart := &helmchartv2.Chart{
		Metadata: &helmchartv2.Metadata{
			APIVersion: helmchartv2.APIVersionV2,
			Name:       addonName,
			Version:    chartVersion,
			Type:       "application",
		},
		Values: map[string]any{},
	}

	for _, manifest := range manifests {
		translated, err := Translate(manifest.Data)
		if err != nil {
			return nil, fmt.Errorf("translating addon manifest %q: %w", manifest.Name, err)
		}

		chart.Templates = append(chart.Templates, &helmchartcommon.File{
			Name: filepath.Join(templatesDir, manifest.Name),
			Data: []byte(translated),
		})
	}

	return chart, nil
}

// LoadChart loads a pre-generated chart from the given filesystem. The chart
// is expected to live in a directory named addonName containing Chart.yaml,
// values.yaml and a templates/ directory.
func LoadChart(fsys fs.FS, addonName string) (*helmchartv2.Chart, error) {
	var files []*archive.BufferedFile

	err := fs.WalkDir(fsys, addonName, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(addonName, path)
		if err != nil {
			return err
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		files = append(files, &archive.BufferedFile{Name: rel, Data: data})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading chart %q: %w", addonName, err)
	}

	chart, err := loader.LoadFiles(files)
	if err != nil {
		return nil, fmt.Errorf("loading chart %q: %w", addonName, err)
	}

	return chart, nil
}

// Write writes a chart to the given directory on disk.
func Write(chart *helmchartv2.Chart, dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, templatesDir), 0o755); err != nil {
		return fmt.Errorf("creating chart directory: %w", err)
	}

	metadataYAML, err := yaml.Marshal(chart.Metadata)
	if err != nil {
		return fmt.Errorf("marshalling Chart.yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), metadataYAML, 0o644); err != nil {
		return fmt.Errorf("writing Chart.yaml: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(valuesFile()), 0o644); err != nil {
		return fmt.Errorf("writing values.yaml: %w", err)
	}

	for _, template := range chart.Templates {
		path := filepath.Join(dir, template.Name)
		if err := os.WriteFile(path, template.Data, 0o644); err != nil {
			return fmt.Errorf("writing template %q: %w", template.Name, err)
		}
	}

	return nil
}
