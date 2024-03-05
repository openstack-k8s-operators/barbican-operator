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

package v1beta1

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BarbicanKeystoneListenerTemplate defines common Spec elements for the KeystoneListener process
type BarbicanKeystoneListenerTemplate struct {
	BarbicanKeystoneListenerTemplateCore `json:",inline"`

	// +kubebuilder:validation:Required
	// ContainerImage - Barbican Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`
}

// BarbicanKeystoneListenerTemplate defines common Spec elements for the KeystoneListener process
type BarbicanKeystoneListenerTemplateCore struct {
	BarbicanComponentTemplate `json:",inline"`

	// TODO(dmendiza): Do we need a setting for number of keystone listener processes
	// or is replica scaling good enough?
}

// BarbicanKeystoneListenerSpec defines the desired state of BarbicanKeystoneListener
type BarbicanKeystoneListenerSpec struct {
	BarbicanTemplate `json:",inline"`

	BarbicanKeystoneListenerTemplate `json:",inline"`
	DatabaseHostname                 string `json:"databaseHostname"`

	TransportURLSecret string `json:"transportURLSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// TLS - Parameters related to the TLS
	TLS tls.Ca `json:"tls,omitempty"`
}

// BarbicanKeystoneListenerStatus defines the observed state of BarbicanKeystoneListener
type BarbicanKeystoneListenerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// ReadyCount of barbican API instances
	ReadyCount int32 `json:"readyCount,omitempty"`

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`

	// Barbican Database Hostname
	DatabaseHostname string `json:"databaseHostname,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BarbicanKeystoneListener is the Schema for the barbicankeystonelistener API
type BarbicanKeystoneListener struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarbicanKeystoneListenerSpec   `json:"spec,omitempty"`
	Status BarbicanKeystoneListenerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BarbicanKeystoneListenerList contains a list of BarbicanKeystoneListener
type BarbicanKeystoneListenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BarbicanKeystoneListener `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BarbicanKeystoneListener{}, &BarbicanKeystoneListenerList{})
}
