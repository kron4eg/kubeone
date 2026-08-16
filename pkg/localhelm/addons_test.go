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

import "testing"

func TestHelmReleaseName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "cluster-autoscaler", want: "cluster-autoscaler"},
		{in: "csi-aws-ebs", want: "csi-aws-ebs"},
		{in: "csi-external-snapshotter_v8.2.1", want: "csi-external-snapshotter-v8-2-1"},
		{in: "cni-canal", want: "cni-canal"},
	}

	for _, tt := range tests {
		if got := helmReleaseName(tt.in); got != tt.want {
			t.Errorf("helmReleaseName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
