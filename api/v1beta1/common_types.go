package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// BarbicanTemplate defines common Spec elements for all Barbican components
// including the top level CR
type BarbicanTemplate struct {

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=barbican
	// ServiceUser - optional username used for this service to register in keystone
	ServiceUser string `json:"serviceUser"`

	// +kubebuilder:validation:Required
	// MariaDB instance name
	// TODO(dmendiza): Is this comment right?
	// Right now required by the maridb-operator to get the credentials from the instance to create the DB
	// Might not be required in future
	DatabaseInstance string `json:"databaseInstance"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=barbican
	// DatabaseAccount - optional MariaDBAccount CR name used for barbican DB, defaults to barbican
	DatabaseAccount string `json:"databaseAccount"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=rabbitmq
	// RabbitMQ instance name
	// Needed to request a transportURL that is created and used in Barbican
	RabbitMqClusterName string `json:"rabbitMqClusterName"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=osp-secret
	// Secret containing the Key Encryption Key (KEK) used for the Simple Crypto backend
	SimpleCryptoBackendSecret string `json:"simpleCryptoBackendSecret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=osp-secret
	// Secret containing all passwords / keys needed
	Secret string `json:"secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={service: BarbicanPassword, simplecryptokek: BarbicanSimpleCryptoKEK}
	// PasswordSelectors - Selectors to identify the ServiceUser password from the Secret
	PasswordSelectors PasswordSelector `json:"passwordSelectors"`

	// +kubebuilder:validation:Required
	// ServiceAccount - service account name used internally to provide Barbican services the default SA name
	ServiceAccount string `json:"serviceAccount"`

	// +kubebuilder:validation:Optional
	PKCS11 *BarbicanPKCS11Template `json:"pkcs11,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=2
	// +listType:=set
	EnabledSecretStores []SecretStore `json:"enabledSecretStores,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="simple_crypto"
	GlobalDefaultSecretStore SecretStore `json:"globalDefaultSecretStore" yaml:"globalDefaultSecretStore"`
}

// BarbicanComponentTemplate - Variables used by every sub-component of Barbican
// (e.g. API, Worker, Listener)
type BarbicanComponentTemplate struct {

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this component. Setting here overrides
	// any global NodeSelector settings within the Barbican CR.
	NodeSelector *map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Maximum=32
	// +kubebuilder:validation:Minimum=0
	// Replicas of Barbican API to run
	Replicas *int32 `json:"replicas"`

	// +kubebuilder:validation:Optional
	// CustomServiceConfig - customize the service config using this parameter to change service defaults,
	// or overwrite rendered information using raw OpenStack config format. The content gets added to
	// to /etc/<service>/<service>.conf.d directory as a custom config file.
	CustomServiceConfig string `json:"customServiceConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigOverwrite - interface to overwrite default config files like e.g. policy.json.
	// But can also be used to add additional files. Those get added to the service config dir in /etc/<service> .
	// TODO: -> implement
	DefaultConfigOverwrite map[string]string `json:"defaultConfigOverwrite,omitempty"`

	// +kubebuilder:validation:Optional
	// CustomServiceConfigSecrets - customize the service config using this parameter to specify Secrets
	// that contain sensitive service config data. The content of each Secret gets added to the
	// /etc/<service>/<service>.conf.d directory as a custom config file.
	CustomServiceConfigSecrets []string `json:"customServiceConfigSecrets,omitempty"`

	// +kubebuilder:validation:Optional
	// Resources - Compute Resources required by this service (Limits/Requests).
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Optional
	// NetworkAttachments is a list of NetworkAttachment resource names to expose the services to the given network
	NetworkAttachments []string `json:"networkAttachments,omitempty"`

	// +kubebuilder:validation:Optional
	// TopologyRef to apply the Topology defined by the associated CR referenced
	// by name
	TopologyRef *topologyv1.TopoRef `json:"topologyRef,omitempty"`
}

// SecretStore type is used by the EnabledSecretStores variable inside the specification.
// +kubebuilder:validation:Enum=simple_crypto;pkcs11
type SecretStore string

const (
	// SecretStoreSimpleCrypto -
	SecretStoreSimpleCrypto SecretStore = "simple_crypto"

	// SecretStorePKCS11 -
	SecretStorePKCS11 SecretStore = "pkcs11"

	// DefaultPKCS11ClientDataPath is the default path for PKCS11 client data
	DefaultPKCS11ClientDataPath = "/etc/hsm-client"
)

// BarbicanPKCS11Template - Includes common HSM properties
type BarbicanPKCS11Template struct {
        // +kubebuilder:validation:Required
        // OpenShift secret that stores the password to login to the PKCS11 session
        LoginSecret string `json:"loginSecret"`

        // +kubebuilder:validation:Required
        // The OpenShift secret that stores the HSM client data.
        // These will be mounted to /var/lib/config-data/hsm
        ClientDataSecret string `json:"clientDataSecret"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default="/etc/hsm-client"
        // Location to which kolla will copy the data in ClientDataSecret.
        ClientDataPath string `json:"clientDataPath"`
}

// AuthSpec defines authentication parameters
type AuthSpec struct {
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// ApplicationCredentialSecret - Secret containing Application Credential ID and Secret
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`
}

// PasswordSelector to identify the DB and AdminUser password from the Secret
type PasswordSelector struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="BarbicanPassword"
	// Service - Selector to get the barbican service user password from the Secret
	Service string `json:"service"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="SimpleCryptoKEK"
	SimpleCryptoKEK string `json:"simplecryptokek"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="PKCS11Pin"
	PKCS11Pin string `json:"pkcs11pin"`
	// +kubebuilder:validation:Optional
	// +listType:=set
	// Fields containing additional Key Encryption Keys(KEK) used for the Simple Crypto backend
	// It is expected that these fields will exist in the secret referenced in SimpleCryptoBackendSecret
	SimpleCryptoAdditionalKEKs []string `json:"simplecryptoadditionalkeks,omitempty"`
}

// ValidateTopology -
func (instance *BarbicanComponentTemplate) ValidateTopology(
	basePath *field.Path,
	namespace string,
) field.ErrorList {
	var allErrs field.ErrorList
	allErrs = append(allErrs, topologyv1.ValidateTopologyRef(
		instance.TopologyRef,
		*basePath.Child("topologyRef"), namespace)...)
	return allErrs
}
