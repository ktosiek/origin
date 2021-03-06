// +build !ignore_autogenerated

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package api

import (
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	conversion "k8s.io/kubernetes/pkg/conversion"
)

func init() {
	if err := api.Scheme.AddGeneratedDeepCopyFuncs(
		DeepCopy_api_ClusterResourceQuota,
		DeepCopy_api_ClusterResourceQuotaList,
		DeepCopy_api_ClusterResourceQuotaSpec,
		DeepCopy_api_ClusterResourceQuotaStatus,
		DeepCopy_api_ResourceQuotasStatusByNamespace,
	); err != nil {
		// if one of the deep copy functions is malformed, detect it immediately.
		panic(err)
	}
}

func DeepCopy_api_ClusterResourceQuota(in ClusterResourceQuota, out *ClusterResourceQuota, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api.DeepCopy_api_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if err := DeepCopy_api_ClusterResourceQuotaSpec(in.Spec, &out.Spec, c); err != nil {
		return err
	}
	if err := DeepCopy_api_ClusterResourceQuotaStatus(in.Status, &out.Status, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_api_ClusterResourceQuotaList(in ClusterResourceQuotaList, out *ClusterResourceQuotaList, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := unversioned.DeepCopy_unversioned_ListMeta(in.ListMeta, &out.ListMeta, c); err != nil {
		return err
	}
	if in.Items != nil {
		in, out := in.Items, &out.Items
		*out = make([]ClusterResourceQuota, len(in))
		for i := range in {
			if err := DeepCopy_api_ClusterResourceQuota(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func DeepCopy_api_ClusterResourceQuotaSpec(in ClusterResourceQuotaSpec, out *ClusterResourceQuotaSpec, c *conversion.Cloner) error {
	if in.Selector != nil {
		in, out := in.Selector, &out.Selector
		*out = new(unversioned.LabelSelector)
		if err := unversioned.DeepCopy_unversioned_LabelSelector(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Selector = nil
	}
	if err := api.DeepCopy_api_ResourceQuotaSpec(in.Quota, &out.Quota, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_api_ClusterResourceQuotaStatus(in ClusterResourceQuotaStatus, out *ClusterResourceQuotaStatus, c *conversion.Cloner) error {
	if err := api.DeepCopy_api_ResourceQuotaStatus(in.Total, &out.Total, c); err != nil {
		return err
	}
	if err := DeepCopy_api_ResourceQuotasStatusByNamespace(in.Namespaces, &out.Namespaces, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_api_ResourceQuotasStatusByNamespace(in ResourceQuotasStatusByNamespace, out *ResourceQuotasStatusByNamespace, c *conversion.Cloner) error {
	if newVal, err := c.DeepCopy(in.orderedMap); err != nil {
		return err
	} else {
		out.orderedMap = newVal.(orderedMap)
	}
	return nil
}
