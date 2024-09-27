package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
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

	// +kubebuilder:validation:Optional
	// CustomServiceConfig - customize the service config using this parameter to change service defaults,
	// or overwrite rendered information using raw OpenStack config format. The content gets added to
	// to /etc/<service>/<service>.conf.d directory as custom.conf file.
	CustomServiceConfig string `json:"customServiceConfig,omitempty"`

	// +kubebuilder:validation:Required
	// ServiceAccount - service account name used internally to provide Barbican services the default SA name
	ServiceAccount string `json:"serviceAccount"`
}

// BarbicanComponentTemplate - Variables used by every sub-component of Barbican
// (e.g. API, Worker, Listener)
type BarbicanComponentTemplate struct {

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this component. Setting here overrides
	// any global NodeSelector settings within the Barbican CR.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

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
}

// BarbicanHSMTemplate - Variables used by the operator to interact with an HSM
type BarbicanHSMTemplate struct {
        // +kubebuilder:validation:Required
        // IP address(es) of the HSM(s)
        IPAddress []string `json:"ipAddress"`

        // +kubebuilder:validation:Required
        // The OpenShift secret storing the PKCS#11 HSM's password
        Pin string `json:"pin"`
}

// BarbicanTrustwayTemplate - Variables specific to Eviden's Trustway Proteccio HSMs
type BarbicanTrustwayTemplate struct {
        BarbicanHSMTemplate `json:",inline"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=4
        // +kubebuilder:validation:Maximum=7
        // +kubebuilder:validation:Minimum=0
        // Level of logging, where 0 means "no logging" and 7 means "debug".
        LoggingLevel int `json:"loggingLevel"`

        // +kubebuilder:validation:Required
        // The HSM certificates. The map's key is the HSM's IP address and
        // the value is the OpenShift secret storing the certificate.
        HSMCertificates map[string]string `json:"hsmCertificates"`

        // +kubebuilder:validation:Required
        // The OpenShift secret storing the client certificate plus its key
        ClientCertificate string `json:"clientCertificate"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=0
        // +kubebuilder:validation:Enum=0;2
        // Working mode for the HSM
        Mode int `json:"mode"`
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
}
