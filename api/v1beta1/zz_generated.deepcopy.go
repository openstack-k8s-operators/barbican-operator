//go:build !ignore_autogenerated

/*
Copyright 2023.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *APIOverrideSpec) DeepCopyInto(out *APIOverrideSpec) {
	*out = *in
	if in.Service != nil {
		in, out := &in.Service, &out.Service
		*out = make(map[service.Endpoint]service.RoutedOverrideSpec, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new APIOverrideSpec.
func (in *APIOverrideSpec) DeepCopy() *APIOverrideSpec {
	if in == nil {
		return nil
	}
	out := new(APIOverrideSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Barbican) DeepCopyInto(out *Barbican) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Barbican.
func (in *Barbican) DeepCopy() *Barbican {
	if in == nil {
		return nil
	}
	out := new(Barbican)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Barbican) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPI) DeepCopyInto(out *BarbicanAPI) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPI.
func (in *BarbicanAPI) DeepCopy() *BarbicanAPI {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPI)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanAPI) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPIList) DeepCopyInto(out *BarbicanAPIList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BarbicanAPI, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPIList.
func (in *BarbicanAPIList) DeepCopy() *BarbicanAPIList {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPIList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanAPIList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPISpec) DeepCopyInto(out *BarbicanAPISpec) {
	*out = *in
	in.BarbicanTemplate.DeepCopyInto(&out.BarbicanTemplate)
	in.BarbicanAPITemplate.DeepCopyInto(&out.BarbicanAPITemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPISpec.
func (in *BarbicanAPISpec) DeepCopy() *BarbicanAPISpec {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPISpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPIStatus) DeepCopyInto(out *BarbicanAPIStatus) {
	*out = *in
	if in.Hash != nil {
		in, out := &in.Hash, &out.Hash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.APIEndpoints != nil {
		in, out := &in.APIEndpoints, &out.APIEndpoints
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(condition.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NetworkAttachments != nil {
		in, out := &in.NetworkAttachments, &out.NetworkAttachments
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPIStatus.
func (in *BarbicanAPIStatus) DeepCopy() *BarbicanAPIStatus {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPIStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPITemplate) DeepCopyInto(out *BarbicanAPITemplate) {
	*out = *in
	in.BarbicanAPITemplateCore.DeepCopyInto(&out.BarbicanAPITemplateCore)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPITemplate.
func (in *BarbicanAPITemplate) DeepCopy() *BarbicanAPITemplate {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPITemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanAPITemplateCore) DeepCopyInto(out *BarbicanAPITemplateCore) {
	*out = *in
	in.BarbicanComponentTemplate.DeepCopyInto(&out.BarbicanComponentTemplate)
	in.Override.DeepCopyInto(&out.Override)
	in.TLS.DeepCopyInto(&out.TLS)
	in.HttpdCustomization.DeepCopyInto(&out.HttpdCustomization)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanAPITemplateCore.
func (in *BarbicanAPITemplateCore) DeepCopy() *BarbicanAPITemplateCore {
	if in == nil {
		return nil
	}
	out := new(BarbicanAPITemplateCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanComponentTemplate) DeepCopyInto(out *BarbicanComponentTemplate) {
	*out = *in
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = new(map[string]string)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]string, len(*in))
			for key, val := range *in {
				(*out)[key] = val
			}
		}
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
	if in.DefaultConfigOverwrite != nil {
		in, out := &in.DefaultConfigOverwrite, &out.DefaultConfigOverwrite
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.CustomServiceConfigSecrets != nil {
		in, out := &in.CustomServiceConfigSecrets, &out.CustomServiceConfigSecrets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Resources.DeepCopyInto(&out.Resources)
	if in.NetworkAttachments != nil {
		in, out := &in.NetworkAttachments, &out.NetworkAttachments
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanComponentTemplate.
func (in *BarbicanComponentTemplate) DeepCopy() *BarbicanComponentTemplate {
	if in == nil {
		return nil
	}
	out := new(BarbicanComponentTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanDefaults) DeepCopyInto(out *BarbicanDefaults) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanDefaults.
func (in *BarbicanDefaults) DeepCopy() *BarbicanDefaults {
	if in == nil {
		return nil
	}
	out := new(BarbicanDefaults)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListener) DeepCopyInto(out *BarbicanKeystoneListener) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListener.
func (in *BarbicanKeystoneListener) DeepCopy() *BarbicanKeystoneListener {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListener)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanKeystoneListener) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListenerList) DeepCopyInto(out *BarbicanKeystoneListenerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BarbicanKeystoneListener, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListenerList.
func (in *BarbicanKeystoneListenerList) DeepCopy() *BarbicanKeystoneListenerList {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListenerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanKeystoneListenerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListenerSpec) DeepCopyInto(out *BarbicanKeystoneListenerSpec) {
	*out = *in
	in.BarbicanTemplate.DeepCopyInto(&out.BarbicanTemplate)
	in.BarbicanKeystoneListenerTemplate.DeepCopyInto(&out.BarbicanKeystoneListenerTemplate)
	out.TLS = in.TLS
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListenerSpec.
func (in *BarbicanKeystoneListenerSpec) DeepCopy() *BarbicanKeystoneListenerSpec {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListenerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListenerStatus) DeepCopyInto(out *BarbicanKeystoneListenerStatus) {
	*out = *in
	if in.Hash != nil {
		in, out := &in.Hash, &out.Hash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(condition.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NetworkAttachments != nil {
		in, out := &in.NetworkAttachments, &out.NetworkAttachments
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListenerStatus.
func (in *BarbicanKeystoneListenerStatus) DeepCopy() *BarbicanKeystoneListenerStatus {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListenerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListenerTemplate) DeepCopyInto(out *BarbicanKeystoneListenerTemplate) {
	*out = *in
	in.BarbicanKeystoneListenerTemplateCore.DeepCopyInto(&out.BarbicanKeystoneListenerTemplateCore)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListenerTemplate.
func (in *BarbicanKeystoneListenerTemplate) DeepCopy() *BarbicanKeystoneListenerTemplate {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListenerTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanKeystoneListenerTemplateCore) DeepCopyInto(out *BarbicanKeystoneListenerTemplateCore) {
	*out = *in
	in.BarbicanComponentTemplate.DeepCopyInto(&out.BarbicanComponentTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanKeystoneListenerTemplateCore.
func (in *BarbicanKeystoneListenerTemplateCore) DeepCopy() *BarbicanKeystoneListenerTemplateCore {
	if in == nil {
		return nil
	}
	out := new(BarbicanKeystoneListenerTemplateCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanList) DeepCopyInto(out *BarbicanList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Barbican, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanList.
func (in *BarbicanList) DeepCopy() *BarbicanList {
	if in == nil {
		return nil
	}
	out := new(BarbicanList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanPKCS11Template) DeepCopyInto(out *BarbicanPKCS11Template) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanPKCS11Template.
func (in *BarbicanPKCS11Template) DeepCopy() *BarbicanPKCS11Template {
	if in == nil {
		return nil
	}
	out := new(BarbicanPKCS11Template)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanSpec) DeepCopyInto(out *BarbicanSpec) {
	*out = *in
	in.BarbicanSpecBase.DeepCopyInto(&out.BarbicanSpecBase)
	in.BarbicanAPI.DeepCopyInto(&out.BarbicanAPI)
	in.BarbicanWorker.DeepCopyInto(&out.BarbicanWorker)
	in.BarbicanKeystoneListener.DeepCopyInto(&out.BarbicanKeystoneListener)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanSpec.
func (in *BarbicanSpec) DeepCopy() *BarbicanSpec {
	if in == nil {
		return nil
	}
	out := new(BarbicanSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanSpecBase) DeepCopyInto(out *BarbicanSpecBase) {
	*out = *in
	in.BarbicanTemplate.DeepCopyInto(&out.BarbicanTemplate)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = new(map[string]string)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]string, len(*in))
			for key, val := range *in {
				(*out)[key] = val
			}
		}
	}
	if in.DefaultConfigOverwrite != nil {
		in, out := &in.DefaultConfigOverwrite, &out.DefaultConfigOverwrite
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanSpecBase.
func (in *BarbicanSpecBase) DeepCopy() *BarbicanSpecBase {
	if in == nil {
		return nil
	}
	out := new(BarbicanSpecBase)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanSpecCore) DeepCopyInto(out *BarbicanSpecCore) {
	*out = *in
	in.BarbicanSpecBase.DeepCopyInto(&out.BarbicanSpecBase)
	in.BarbicanAPI.DeepCopyInto(&out.BarbicanAPI)
	in.BarbicanWorker.DeepCopyInto(&out.BarbicanWorker)
	in.BarbicanKeystoneListener.DeepCopyInto(&out.BarbicanKeystoneListener)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanSpecCore.
func (in *BarbicanSpecCore) DeepCopy() *BarbicanSpecCore {
	if in == nil {
		return nil
	}
	out := new(BarbicanSpecCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanStatus) DeepCopyInto(out *BarbicanStatus) {
	*out = *in
	if in.Hash != nil {
		in, out := &in.Hash, &out.Hash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(condition.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanStatus.
func (in *BarbicanStatus) DeepCopy() *BarbicanStatus {
	if in == nil {
		return nil
	}
	out := new(BarbicanStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanTemplate) DeepCopyInto(out *BarbicanTemplate) {
	*out = *in
	out.PasswordSelectors = in.PasswordSelectors
	if in.PKCS11 != nil {
		in, out := &in.PKCS11, &out.PKCS11
		*out = new(BarbicanPKCS11Template)
		**out = **in
	}
	if in.EnabledSecretStores != nil {
		in, out := &in.EnabledSecretStores, &out.EnabledSecretStores
		*out = make([]SecretStore, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanTemplate.
func (in *BarbicanTemplate) DeepCopy() *BarbicanTemplate {
	if in == nil {
		return nil
	}
	out := new(BarbicanTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorker) DeepCopyInto(out *BarbicanWorker) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorker.
func (in *BarbicanWorker) DeepCopy() *BarbicanWorker {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorker)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanWorker) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorkerList) DeepCopyInto(out *BarbicanWorkerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]BarbicanWorker, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorkerList.
func (in *BarbicanWorkerList) DeepCopy() *BarbicanWorkerList {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorkerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BarbicanWorkerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorkerSpec) DeepCopyInto(out *BarbicanWorkerSpec) {
	*out = *in
	in.BarbicanTemplate.DeepCopyInto(&out.BarbicanTemplate)
	in.BarbicanWorkerTemplate.DeepCopyInto(&out.BarbicanWorkerTemplate)
	out.TLS = in.TLS
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorkerSpec.
func (in *BarbicanWorkerSpec) DeepCopy() *BarbicanWorkerSpec {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorkerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorkerStatus) DeepCopyInto(out *BarbicanWorkerStatus) {
	*out = *in
	if in.Hash != nil {
		in, out := &in.Hash, &out.Hash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(condition.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.NetworkAttachments != nil {
		in, out := &in.NetworkAttachments, &out.NetworkAttachments
		*out = make(map[string][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorkerStatus.
func (in *BarbicanWorkerStatus) DeepCopy() *BarbicanWorkerStatus {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorkerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorkerTemplate) DeepCopyInto(out *BarbicanWorkerTemplate) {
	*out = *in
	in.BarbicanWorkerTemplateCore.DeepCopyInto(&out.BarbicanWorkerTemplateCore)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorkerTemplate.
func (in *BarbicanWorkerTemplate) DeepCopy() *BarbicanWorkerTemplate {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorkerTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarbicanWorkerTemplateCore) DeepCopyInto(out *BarbicanWorkerTemplateCore) {
	*out = *in
	in.BarbicanComponentTemplate.DeepCopyInto(&out.BarbicanComponentTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarbicanWorkerTemplateCore.
func (in *BarbicanWorkerTemplateCore) DeepCopy() *BarbicanWorkerTemplateCore {
	if in == nil {
		return nil
	}
	out := new(BarbicanWorkerTemplateCore)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpdCustomization) DeepCopyInto(out *HttpdCustomization) {
	*out = *in
	if in.CustomConfigSecret != nil {
		in, out := &in.CustomConfigSecret, &out.CustomConfigSecret
		*out = new(string)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpdCustomization.
func (in *HttpdCustomization) DeepCopy() *HttpdCustomization {
	if in == nil {
		return nil
	}
	out := new(HttpdCustomization)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PasswordSelector) DeepCopyInto(out *PasswordSelector) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PasswordSelector.
func (in *PasswordSelector) DeepCopy() *PasswordSelector {
	if in == nil {
		return nil
	}
	out := new(PasswordSelector)
	in.DeepCopyInto(out)
	return out
}
