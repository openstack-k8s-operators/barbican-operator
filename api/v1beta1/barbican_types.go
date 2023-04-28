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

const (
	// DbSyncHash hash
	DbSyncHash = "dbsync"
)

// BarbicanSpec defines the desired state of Barbican
type BarbicanSpec struct {
	BarbicanTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs"`

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this component. Setting here overrides
	// any global NodeSelector settings within the Barbican CR.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// CustomServiceConfig - customize the service config using this parameter to change service defaults,
	// or overwrite rendered information using raw OpenStack config format. The content gets added to
	// to /etc/<service>/<service>.conf.d directory as custom.conf file.
	CustomServiceConfig string `json:"customServiceConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigOverwrite - interface to overwrite default config files like e.g. logging.conf or policy.json.
	// But can also be used to add additional files. Those get added to the service config dir in /etc/<service> .
	// TODO(dmendiza): -> implement
	DefaultConfigOverwrite map[string]string `json:"defaultConfigOverwrite,omitempty"`

	// +kubebuilder:validation:Required
	// BarbicanAPIInternal - Spec definition for the internal and admin API service of this Barbican deployment

	BarbicanAPI BarbicanAPITemplate `json:"barbicanAPI"`

	BarbicanWorker BarbicanWorkerTemplate `json:"barbicanWorker"`
}

// BarbicanStatus defines the observed state of Barbican
type BarbicanStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// API endpoints
	// TODO(dmendiza): This thing is hideous.  Why do we need it?
	APIEndpoints map[string]map[string]string `json:"apiEndpoints,omitempty"`

	// ServiceIDs
	// TODO(dmendiza): This thing is hideous.  Why do we need it?
	ServiceIDs map[string]string `json:"serviceIDs,omitempty"`

	// ReadyCount of Barbican API instances
	BarbicanAPIReadyCount int32 `json:"barbicanAPIReadyCount,omitempty"`

	// ReadyCount of Barbican Worker instances
	BarbicanWorkerReadyCount int32 `json:"barbicanWorkerReadyCount,omitempty"`

	// TransportURLSecret - Secret containing RabbitMQ transportURL
	TransportURLSecret string `json:"transportURLSecret,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Barbican is the Schema for the barbicans API
type Barbican struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarbicanSpec   `json:"spec,omitempty"`
	Status BarbicanStatus `json:"status,omitempty"`
}

// IsReady returns true when both API and Worker are ready
func (instance Barbican) IsReady() bool {
	return instance.Status.Conditions.IsTrue(BarbicanAPIReadyCondition) &&
		instance.Status.Conditions.IsTrue(BarbicanWorkerReadyCondition)
}

//+kubebuilder:object:root=true

// BarbicanList contains a list of Barbican
type BarbicanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Barbican `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Barbican{}, &BarbicanList{})
}
