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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8c.io/kubeone/pkg/addons"
	"k8c.io/kubeone/pkg/addons/helmchart"
	"k8c.io/kubeone/pkg/fail"
	"k8c.io/kubeone/pkg/state"
)

type addonsChartOpts struct {
	globalOptions
	OutputDir string `longflag:"output" shortflag:"o"`
	Force     bool   `longflag:"force"`
	All       bool   `longflag:"all"`
}

func addonsChartCmd(rootFlags *pflag.FlagSet) *cobra.Command {
	opts := &addonsChartOpts{}

	cmd := &cobra.Command{
		Use:           "chart [addon-name]",
		Short:         "Convert an addon into a Helm chart",
		SilenceErrors: true,
		Example:       `kubeone -m kubeone.yaml addons chart cluster-autoscaler`,
		Args: func(_ *cobra.Command, args []string) error {
			if !opts.All && len(args) != 1 {
				return errors.New("exactly one addon name is required (or use --all)")
			}

			if opts.All && len(args) > 0 {
				return errors.New("addon name must not be provided together with --all")
			}

			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			gopts, err := persistentGlobalOptions(rootFlags)
			if err != nil {
				return err
			}

			opts.globalOptions = *gopts

			return runAddonsChart(opts, args)
		},
	}

	cmd.Flags().StringVarP(
		&opts.OutputDir,
		longFlagName(opts, "OutputDir"),
		shortFlagName(opts, "OutputDir"),
		"",
		"directory to write the chart to (defaults to ./<addon-name>, or ./addon-charts with --all).",
	)
	cmd.Flags().BoolVar(
		&opts.Force,
		longFlagName(opts, "Force"),
		false,
		"overwrite existing chart directories.",
	)
	cmd.Flags().BoolVar(
		&opts.All,
		longFlagName(opts, "All"),
		false,
		"convert all embedded addons.",
	)

	return cmd
}

func runAddonsChart(opts *addonsChartOpts, args []string) error {
	s, err := opts.BuildState()
	if err != nil {
		return err
	}

	var names []string

	switch {
	case opts.All:
		names = addons.EmbeddedAddonNames()
	default:
		names = []string{args[0]}
	}

	for _, name := range names {
		if err := convertAddon(s, name, opts); err != nil {
			return err
		}
	}

	return nil
}

func convertAddon(s *state.State, addonName string, opts *addonsChartOpts) error {
	chart, err := addons.Chart(s, addonName)
	if err != nil {
		return err
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		if opts.All {
			outputDir = filepath.Join("addon-charts", addonName)
		} else {
			outputDir = addonName
		}
	} else if opts.All {
		outputDir = filepath.Join(outputDir, addonName)
	}

	if !opts.Force {
		if _, err := os.Stat(outputDir); err == nil {
			return fail.RuntimeError{
				Op:  fmt.Sprintf("checking output directory %q", outputDir),
				Err: errors.New("directory already exists, use --force to overwrite"),
			}
		}
	}

	if err := helmchart.Write(chart, outputDir); err != nil {
		return fail.Runtime(err, "writing chart for addon %q", addonName)
	}

	fmt.Printf("Wrote chart for addon %q to %s\n", addonName, outputDir)

	return nil
}
