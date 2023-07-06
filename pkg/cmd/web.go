/*
Copyright 2023 The KubeOne Authors.

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
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type webOpts struct {
	globalOptions
	ListenAddr string `longflag:"listen"`
}

func webCmd(rootFlags *pflag.FlagSet) *cobra.Command {
	opts := &webOpts{}

	cmd := &cobra.Command{
		Use:           "web",
		Short:         "launch a webserver with kubeone dashboard",
		Long:          heredoc.Doc(``),
		SilenceErrors: true,
		Example:       `kubeone web -m mycluster.yaml -t terraformoutput.json`,
		RunE: func(_ *cobra.Command, args []string) error {
			gopts, err := persistentGlobalOptions(rootFlags)
			if err != nil {
				return err
			}
			opts.globalOptions = *gopts

			return runWeb(opts)
		},
	}

	cmd.Flags().StringVar(&opts.ListenAddr, longFlagName(opts, "ListenAddr"), "127.0.0.1:8008", "Web UI bind address")

	return cmd
}

func runWeb(opts *webOpts) error {
	return nil
}
