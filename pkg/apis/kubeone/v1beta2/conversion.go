/*
Copyright 2020 The KubeOne Authors.

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

package v1beta2

import (
	kubeoneapi "k8c.io/kubeone/pkg/apis/kubeone"

	conversion "k8s.io/apimachinery/pkg/conversion"
)

// Convert_v1beta2_CloudProviderSpec_To_kubeone_CloudProviderSpec routes CloudConfig/CSIConfig from
// top-level v1beta2 fields into the per-provider structs in the internal type, and folds CABundle.
func Convert_v1beta2_CloudProviderSpec_To_kubeone_CloudProviderSpec(in *CloudProviderSpec, out *kubeoneapi.CloudProviderSpec, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_CloudProviderSpec_To_kubeone_CloudProviderSpec(in, out, scope); err != nil {
		return err
	}

	if len(in.CloudConfig) > 0 {
		switch {
		case out.AWS != nil:
			out.AWS.CloudConfig = in.CloudConfig
		case out.Azure != nil:
			out.Azure.CloudConfig = in.CloudConfig
		case out.Openstack != nil:
			out.Openstack.CloudConfig = in.CloudConfig
		case out.Vsphere != nil:
			out.Vsphere.CloudConfig = in.CloudConfig
		}
	}

	if len(in.CSIConfig) > 0 {
		if out.Vsphere != nil {
			out.Vsphere.CSIConfig = in.CSIConfig
		}
	}

	return nil
}

// Convert_kubeone_AWSSpec_To_v1beta2_AWSSpec extracts nested CloudConfig back to top-level.
func Convert_kubeone_AWSSpec_To_v1beta2_AWSSpec(in *kubeoneapi.AWSSpec, out *AWSSpec, scope conversion.Scope) error {
	if err := autoConvert_kubeone_AWSSpec_To_v1beta2_AWSSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_AWSSpec_To_kubeone_AWSSpec creates internal AWSSpec from v1beta2 (fields are now nested).
func Convert_v1beta2_AWSSpec_To_kubeone_AWSSpec(in *AWSSpec, out *kubeoneapi.AWSSpec, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_AWSSpec_To_kubeone_AWSSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_kubeone_AzureSpec_To_v1beta2_AzureSpec extracts nested CloudConfig back to top-level.
func Convert_kubeone_AzureSpec_To_v1beta2_AzureSpec(in *kubeoneapi.AzureSpec, out *AzureSpec, scope conversion.Scope) error {
	if err := autoConvert_kubeone_AzureSpec_To_v1beta2_AzureSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_AzureSpec_To_kubeone_AzureSpec creates internal AzureSpec from v1beta2 (fields are now nested).
func Convert_v1beta2_AzureSpec_To_kubeone_AzureSpec(in *AzureSpec, out *kubeoneapi.AzureSpec, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_AzureSpec_To_kubeone_AzureSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_kubeone_OpenstackSpec_To_v1beta2_OpenstackSpec extracts nested CloudConfig back to top-level.
func Convert_kubeone_OpenstackSpec_To_v1beta2_OpenstackSpec(in *kubeoneapi.OpenstackSpec, out *OpenstackSpec, scope conversion.Scope) error {
	if err := autoConvert_kubeone_OpenstackSpec_To_v1beta2_OpenstackSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_OpenstackSpec_To_kubeone_OpenstackSpec creates internal OpenstackSpec from v1beta2 (fields are now nested).
func Convert_v1beta2_OpenstackSpec_To_kubeone_OpenstackSpec(in *OpenstackSpec, out *kubeoneapi.OpenstackSpec, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_OpenstackSpec_To_kubeone_OpenstackSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_kubeone_VsphereSpec_To_v1beta2_VsphereSpec extracts nested CloudConfig/CSIConfig back to top-level.
func Convert_kubeone_VsphereSpec_To_v1beta2_VsphereSpec(in *kubeoneapi.VsphereSpec, out *VsphereSpec, scope conversion.Scope) error {
	if err := autoConvert_kubeone_VsphereSpec_To_v1beta2_VsphereSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_VsphereSpec_To_kubeone_VsphereSpec creates internal VsphereSpec from v1beta2 (fields are now nested).
func Convert_v1beta2_VsphereSpec_To_kubeone_VsphereSpec(in *VsphereSpec, out *kubeoneapi.VsphereSpec, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_VsphereSpec_To_kubeone_VsphereSpec(in, out, scope); err != nil {
		return err
	}

	return nil
}

// Convert_v1beta2_CertificateAuthorithyConfig_To_kubeone_CertificateAuthorityConfig bridges the typo rename.
func Convert_v1beta2_CertificateAuthorithyConfig_To_kubeone_CertificateAuthorityConfig(in *CertificateAuthorithyConfig, out *kubeoneapi.CertificateAuthorityConfig, _ conversion.Scope) error {
	out.Bundle = in.Bundle
	out.File = in.File
	out.CertificateValidityPeriod = in.CertificateValidityPeriod

	return nil
}

// Convert_kubeone_CertificateAuthorityConfig_To_v1beta2_CertificateAuthorithyConfig bridges the typo rename (reverse direction).
func Convert_kubeone_CertificateAuthorityConfig_To_v1beta2_CertificateAuthorithyConfig(in *kubeoneapi.CertificateAuthorityConfig, out *CertificateAuthorithyConfig, _ conversion.Scope) error {
	out.Bundle = in.Bundle
	out.File = in.File
	out.CertificateValidityPeriod = in.CertificateValidityPeriod

	return nil
}

func Convert_v1beta2_KubeOneCluster_To_kubeone_KubeOneCluster(in *KubeOneCluster, out *kubeoneapi.KubeOneCluster, scope conversion.Scope) error {
	if err := autoConvert_v1beta2_KubeOneCluster_To_kubeone_KubeOneCluster(in, out, scope); err != nil {
		return err
	}

	if len(in.HelmReleases) > 0 && out.Addons == nil {
		out.Addons = &kubeoneapi.Addons{}
	}

	for _, hr := range in.HelmReleases {
		hr := hr.DeepCopy()
		hrOut := kubeoneapi.HelmRelease{}

		if err := autoConvert_v1beta2_HelmRelease_To_kubeone_HelmRelease(hr, &hrOut, scope); err != nil {
			return err
		}

		out.Addons.Addons = append(out.Addons.Addons, kubeoneapi.AddonRef{HelmRelease: &hrOut})
	}

	return nil
}

func Convert_kubeone_KubeOneCluster_To_v1beta2_KubeOneCluster(in *kubeoneapi.KubeOneCluster, out *KubeOneCluster, scope conversion.Scope) error {
	// AssetsConfiguration has been removed in the v1beta2 API
	return autoConvert_kubeone_KubeOneCluster_To_v1beta2_KubeOneCluster(in, out, scope)
}

func Convert_v1beta2_ContainerRuntimeConfig_To_kubeone_ContainerRuntimeConfig(in *ContainerRuntimeConfig, out *kubeoneapi.ContainerRuntimeConfig, scope conversion.Scope) error {
	return autoConvert_v1beta2_ContainerRuntimeConfig_To_kubeone_ContainerRuntimeConfig(in, out, scope)
}

func Convert_v1beta2_Addon_To_kubeone_AddonRef(in *Addon, out *kubeoneapi.AddonRef, _ conversion.Scope) error {
	out.Addon = &kubeoneapi.Addon{
		Name:              in.Name,
		DisableTemplating: in.DisableTemplating,
		Params:            in.Params,
		Delete:            in.Delete,
	}

	return nil
}

func Convert_kubeone_KubeletConfig_To_v1beta2_KubeletConfig(in *kubeoneapi.KubeletConfig, out *KubeletConfig, s conversion.Scope) error {
	return autoConvert_kubeone_KubeletConfig_To_v1beta2_KubeletConfig(in, out, s)
}

func Convert_kubeone_AddonRef_To_v1beta2_Addon(*kubeoneapi.AddonRef, *Addon, conversion.Scope) error {
	return nil
}

func Convert_v1beta2_Addons_To_kubeone_Addons(in *Addons, out *kubeoneapi.Addons, scope conversion.Scope) error {
	return autoConvert_v1beta2_Addons_To_kubeone_Addons(in, out, scope)
}

func Convert_kubeone_Features_To_v1beta2_Features(in *kubeoneapi.Features, out *Features, s conversion.Scope) error {
	return autoConvert_kubeone_Features_To_v1beta2_Features(in, out, s)
}

func Convert_v1beta2_Features_To_kubeone_Features(in *Features, out *kubeoneapi.Features, s conversion.Scope) error {
	return autoConvert_v1beta2_Features_To_kubeone_Features(in, out, s)
}

func Convert_v1beta2_CiliumSpec_To_kubeone_CiliumSpec(in *CiliumSpec, out *kubeoneapi.CiliumSpec, _ conversion.Scope) error {
	out.KubeProxyReplacement = in.KubeProxyReplacement == KubeProxyReplacementStrict
	out.EnableHubble = in.EnableHubble
	out.EnableL2Announcements = in.EnableL2Announcements
	out.EnableGatewayAPI = in.EnableGatewayAPI
	out.EnableLocalRedirectPolicy = in.EnableLocalRedirectPolicy

	return nil
}

func Convert_kubeone_CiliumSpec_To_v1beta2_CiliumSpec(in *kubeoneapi.CiliumSpec, out *CiliumSpec, _ conversion.Scope) error {
	out.KubeProxyReplacement = KubeProxyReplacementDisabled
	if in.KubeProxyReplacement {
		out.KubeProxyReplacement = KubeProxyReplacementStrict
	}
	out.EnableHubble = in.EnableHubble
	out.EnableL2Announcements = in.EnableL2Announcements
	out.EnableGatewayAPI = in.EnableGatewayAPI
	out.EnableLocalRedirectPolicy = in.EnableLocalRedirectPolicy

	return nil
}

func Convert_kubeone_ContainerRuntimeContainerd_To_v1beta2_ContainerRuntimeContainerd(in *kubeoneapi.ContainerRuntimeContainerd, out *ContainerRuntimeContainerd, s conversion.Scope) error {
	return autoConvert_kubeone_ContainerRuntimeContainerd_To_v1beta2_ContainerRuntimeContainerd(in, out, s)
}

func Convert_v1beta2_ProviderSpec_To_kubeone_ProviderSpec(in *ProviderSpec, out *kubeoneapi.ProviderSpec, s conversion.Scope) error {
	return autoConvert_v1beta2_ProviderSpec_To_kubeone_ProviderSpec(in, out, s)
}

func Convert_v1beta2_CNI_To_kubeone_CNI(in *CNI, out *kubeoneapi.CNI, s conversion.Scope) error {
	return autoConvert_v1beta2_CNI_To_kubeone_CNI(in, out, s)
}

func Convert_kubeone_KubeProxyConfig_To_v1beta2_KubeProxyConfig(in *kubeoneapi.KubeProxyConfig, out *KubeProxyConfig, s conversion.Scope) error {
	return autoConvert_kubeone_KubeProxyConfig_To_v1beta2_KubeProxyConfig(in, out, s)
}
