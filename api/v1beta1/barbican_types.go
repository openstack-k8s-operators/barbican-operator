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
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"

)

const (
	// DbSyncHash hash
	DbSyncHash = "dbsync"

	// PKCS11PrepHash hash
	PKCS11PrepHash = "pkcs11prep"

	// Container image fall-back defaults

	// BarbicanAPIContainerImage is the fall-back container image for BarbicanAPI
	BarbicanAPIContainerImage = "quay.io/podified-antelope-centos9/openstack-barbican-api:current-podified"

	// BarbicanWorkerContainerImage is the fall-back container image for BarbicanAPI
	BarbicanWorkerContainerImage = "quay.io/podified-antelope-centos9/openstack-barbican-worker:current-podified"

	// BarbicanKeystoneListenerContainerImage is the fall-back container image for BarbicanAPI
	BarbicanKeystoneListenerContainerImage = "quay.io/podified-antelope-centos9/openstack-barbican-keystone-listener:current-podified"

	// Barbican API timeout
	APITimeout = 90
)

// BarbicanSpec defines the desired state of Barbican
type BarbicanSpec struct {
	BarbicanSpecBase `json:",inline"`

	// +kubebuilder:validation:Required
	// BarbicanAPI - Spec definition for the  API services of this Barbican deployment
	BarbicanAPI BarbicanAPITemplate `json:"barbicanAPI"`

	// +kubebuilder:validation:Required
	// BarbicanWorker - Spec definition for the Worker service of this Barbican deployment
	BarbicanWorker BarbicanWorkerTemplate `json:"barbicanWorker"`

	// +kubebuilder:validation:Required
	// BarbicanKeystoneListener - Spec definition for the KeystoneListener service of this Barbican deployment
	BarbicanKeystoneListener BarbicanKeystoneListenerTemplate `json:"barbicanKeystoneListener"`
}

// BarbicanSpecCore defines the desired state of Barbican, for use with the OpenStackControlplane CR (no containerImages)
type BarbicanSpecCore struct {
	BarbicanSpecBase `json:",inline"`

	// +kubebuilder:validation:Required
	// BarbicanAPI - Spec definition for the  API services of this Barbican deployment
	BarbicanAPI BarbicanAPITemplateCore `json:"barbicanAPI"`

	// +kubebuilder:validation:Required
	// BarbicanWorker - Spec definition for the Worker service of this Barbican deployment
	BarbicanWorker BarbicanWorkerTemplateCore `json:"barbicanWorker"`

	// +kubebuilder:validation:Required
	// BarbicanKeystoneListener - Spec definition for the KeystoneListener service of this Barbican deployment
	BarbicanKeystoneListener BarbicanKeystoneListenerTemplateCore `json:"barbicanKeystoneListener"`
}

// BarbicanSpecBase -
type BarbicanSpecBase struct {
	BarbicanTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs"`

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this component. Setting here overrides
	// any global NodeSelector settings within the Barbican CR.
	NodeSelector *map[string]string `json:"nodeSelector,omitempty"`

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

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=90
	// Barbican API timeout
	APITimeout int `json:"apiTimeout"`

	// +kubebuilder:validation:Optional
	// TopologyRef to apply the Topology defined by the associated CR referenced
	// by name
	TopologyRef *topologyv1.TopoRef `json:"topologyRef,omitempty"`
}

// BarbicanStatus defines the observed state of Barbican
type BarbicanStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// ServiceID
	ServiceID string `json:"serviceID,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// ReadyCount of Barbican API instances
	BarbicanAPIReadyCount int32 `json:"barbicanAPIReadyCount,omitempty"`

	// ReadyCount of Barbican Worker instances
	BarbicanWorkerReadyCount int32 `json:"barbicanWorkerReadyCount,omitempty"`

	// ReadyCount of Barbican KeystoneListener instances
	BarbicanKeystoneListenerReadyCount int32 `json:"barbicanKeystoneListenerReadyCount,omitempty"`

	// TransportURLSecret - Secret containing RabbitMQ transportURL
	TransportURLSecret string `json:"transportURLSecret,omitempty"`

	// Barbican Database Hostname
	DatabaseHostname string `json:"databaseHostname,omitempty"`

	// ObservedGeneration - the most recent generation observed for this
	// service. If the observed generation is less than the spec generation,
	// then the controller has not processed the latest changes injected by
	// the opentack-operator in the top-level CR (e.g. the ContainerImage)
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// Barbican is the Schema for the barbicans API
type Barbican struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BarbicanSpec   `json:"spec,omitempty"`
	Status BarbicanStatus `json:"status,omitempty"`
}

// IsReady - returns true if Barbican is reconciled successfully
func (instance Barbican) IsReady() bool {
	return instance.Status.Conditions.IsTrue(condition.ReadyCondition)
}

//+kubebuilder:object:root=true

// BarbicanList contains a list of Barbican
type BarbicanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Barbican `json:"items"`
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance Barbican) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance Barbican) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance Barbican) RbacResourceName() string {
	return "barbican-" + instance.Name
}

func init() {
	SchemeBuilder.Register(&Barbican{}, &BarbicanList{})
}

// SetupDefaults - initializes any CRD field defaults based on environment variables (the defaulting mechanism itself is implemented via webhooks)
func SetupDefaults() {
	// Acquire environmental defaults and initialize Barbican defaults with them
	barbicanDefaults := BarbicanDefaults{
		APIContainerImageURL:              util.GetEnvVar("RELATED_IMAGE_BARBICAN_API_IMAGE_URL_DEFAULT", BarbicanAPIContainerImage),
		WorkerContainerImageURL:           util.GetEnvVar("RELATED_IMAGE_BARBICAN_WORKER_IMAGE_URL_DEFAULT", BarbicanWorkerContainerImage),
		KeystoneListenerContainerImageURL: util.GetEnvVar("RELATED_IMAGE_BARBICAN_KEYSTONE_LISTENER_IMAGE_URL_DEFAULT", BarbicanKeystoneListenerContainerImage),
		BarbicanAPITimeout:                APITimeout,
	}

	SetupBarbicanDefaults(barbicanDefaults)
}
