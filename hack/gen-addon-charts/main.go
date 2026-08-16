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

// gen-addon-charts regenerates the pre-built Helm charts under charts/ from
// the embedded addon manifests. It is invoked by hack/update-codegen.sh.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	embeddedaddons "k8c.io/kubeone/addons"
	"k8c.io/kubeone/pkg/addons/helmchart"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	const chartsDir = "charts"

	entries, err := fs.ReadDir(os.DirFS("."), chartsDir)
	if err != nil {
		return fmt.Errorf("reading %q: %w", chartsDir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if err := os.RemoveAll(filepath.Join(chartsDir, entry.Name())); err != nil {
			return fmt.Errorf("removing %q: %w", entry.Name(), err)
		}
	}

	addonEntries, err := fs.ReadDir(embeddedaddons.FS, ".")
	if err != nil {
		return fmt.Errorf("reading embedded addons: %w", err)
	}

	var names []string
	for _, entry := range addonEntries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		manifests, err := helmchart.ReadManifests(embeddedaddons.FS, name)
		if err != nil {
			return err
		}

		if len(manifests) == 0 {
			continue
		}

		chart, err := helmchart.BuildChart(name, manifests)
		if err != nil {
			return err
		}

		if err := helmchart.Write(chart, filepath.Join(chartsDir, name)); err != nil {
			return err
		}

		fmt.Printf("generated chart for addon %q\n", name)
	}

	return nil
}
