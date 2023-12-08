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
	// DatabaseUser - optional username used for barbican DB, defaults to barbican
	DatabaseUser string `json:"databaseUser"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=rabbitmq
	// RabbitMQ instance name
	// Needed to request a transportURL that is created and used in Barbican
	RabbitMqClusterName string `json:"rabbitMqClusterName"`

	// +kubebuilder:validation:Optional
	// Secret containing SimpleCrypto KEK
	SimpleCryptoBackendKEKSecret string `json:"simpleCryptoBackendKEKSecret,omitempty"`

	// +kubebuilder:validation:Optional
	// Secret containing all passwords / keys needed
	Secret string `json:"secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={database: BarbicanDatabasePassword, service: BarbicanPassword}
	// TODO(dmendiza): Maybe we'll add SimpleCrypto key here?
	// PasswordSelectors - Selectors to identify the DB and ServiceUser password from the Secret
	PasswordSelectors PasswordSelector `json:"passwordSelectors"`

	// +kubebuilder:validation:Optional
	// Debug - enable debug for different deploy stages. If an init container is used, it runs and the
	// actual action pod gets started with sleep infinity
	// TODO(dmendiza): Do we need this?
	Debug BarbicanDebug `json:"debug,omitempty"`

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

	// +kubebuilder:validation:Required
	// ContainerImage - Barbican Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`

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

// PasswordSelector to identify the DB and AdminUser password from the Secret
type PasswordSelector struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="BarbicanDatabasePassword"
	// Database - Selector to get the barbican database user password from the Secret
	Database string `json:"database"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="BarbicanPassword"
	// Service - Selector to get the barbican service user password from the Secret
	Service string `json:"service"`
}

// BarbicanDebug indicates whether certain stages of deployment should be paused
type BarbicanDebug struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// dbInitContainer enable debug (waits until /tmp/stop-init-container disappears)
	DBInitContainer bool `json:"dbInitContainer"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// dbSync enable debug
	DBSync bool `json:"dbSync"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// initContainer enable debug (waits until /tmp/stop-init-container disappears)
	InitContainer bool `json:"initContainer"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Service enable debug
	Service bool `json:"service"`
}
