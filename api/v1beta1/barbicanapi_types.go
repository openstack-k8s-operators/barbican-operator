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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BarbicanAPITemplate defines the input parameters for the Barbican API service
type BarbicanAPITemplate struct {
	// Common input parameters for the Barbican API service
	BarbicanComponentTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// ExternalEndpoints, expose a VIP via MetalLB on the pre-created address pool
	ExternalEndpoints []MetalLBConfig `json:"externalEndpoints,omitempty"`
}

// BarbicanAPISpec defines the desired state of BarbicanAPI
type BarbicanAPISpec struct {
	BarbicanTemplate `json:",inline"`

	BarbicanAPITemplate `json:",inline"`
}

// BarbicanAPIStatus defines the observed state of BarbicanAPI
type BarbicanAPIStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BarbicanAPI is the Schema for the barbicanapis API
type BarbicanAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarbicanAPISpec   `json:"spec,omitempty"`
	Status BarbicanAPIStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BarbicanAPIList contains a list of BarbicanAPI
type BarbicanAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BarbicanAPI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BarbicanAPI{}, &BarbicanAPIList{})
}
