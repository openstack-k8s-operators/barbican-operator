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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BarbicanWorkerTemplate defines common Spec elements for the Worker process
type BarbicanWorkerTemplate struct {
	BarbicanComponentTemplate `json:",inline"`

	// TODO(dmendiza): Do we need a setting for number of worker processes
	// or is replica scaling good enough?
}

// BarbicanWorkerSpec defines the desired state of BarbicanWorker
type BarbicanWorkerSpec struct {
	BarbicanTemplate `json:",inline"`

	BarbicanWorkerTemplate `json:",inline"`
	DatabaseHostname       string `json:"databaseHostname"`

	TransportURLSecret string `json:"transportURLSecret,omitempty"`
}

// BarbicanWorkerStatus defines the observed state of BarbicanWorker
type BarbicanWorkerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// ReadyCount of barbican API instances
	ReadyCount int32 `json:"readyCount,omitempty"`

	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// API endpoint
	//APIEndpoints map[string]string `json:"apiEndpoint,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`

	// Barbican Database Hostname
	DatabaseHostname string `json:"databaseHostname,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BarbicanWorker is the Schema for the barbicanworkers API
type BarbicanWorker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarbicanWorkerSpec   `json:"spec,omitempty"`
	Status BarbicanWorkerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BarbicanWorkerList contains a list of BarbicanWorker
type BarbicanWorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BarbicanWorker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BarbicanWorker{}, &BarbicanWorkerList{})
}
