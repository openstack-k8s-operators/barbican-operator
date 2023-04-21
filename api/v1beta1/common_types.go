package v1beta1

import "github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"

// BarbicanTemplate defines common Spec elements for all Barbican components
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
	// Secret containing all passwords / keys needed
	Secret string `json:"secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={database: BarbicanDatabasePassword, service: BarbicanServiceUserPassword}
	// TODO(dmendiza): Maybe we'll add SimpleCrypto key here?
	// PasswordSelectors - Selectors to identify the DB and ServiceUser password from the Secret
	PasswordSelectors PasswordSelector `json:"passwordSelectors"`

	// +kubebuilder:validation:Optional
	// Debug - enable debug for different deploy stages. If an init container is used, it runs and the
	// actual action pod gets started with sleep infinity
	// TODO(dmendiza): Do we need this?
	Debug BarbicanDebug `json:"debug,omitempty"`
}

// BarbicanComponentTemplate - Variables used by every component of Barbican
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
	Replicas int32 `json:"replicas"`

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
	// +kubebuilder:default="BarbicanServiceUserPassword"
	// Service - Selector to get the barbican service user password from the Secret
	Service string `json:"service"`

	// +kubebuilder:validation:Optional
	// SimpleCryptoKEK - base64 encoded KEK for SimpleCrypto backend
	SimpleCryptoKEK string `json:"simpleCryptoKEK,omitempty"`
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

// MetalLBConfig to configure the MetalLB loadbalancer service
type MetalLBConfig struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=internal;public
	// Endpoint, OpenStack endpoint this service maps to
	Endpoint endpoint.Endpoint `json:"endpoint"`

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// IPAddressPool expose VIP via MetalLB on the IPAddressPool
	IPAddressPool string `json:"ipAddressPool"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	// SharedIP if true, VIP/VIPs get shared with multiple services
	SharedIP bool `json:"sharedIP"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	// SharedIPKey specifies the sharing key which gets set as the annotation on the LoadBalancer service.
	// Services which share the same VIP must have the same SharedIPKey. Defaults to the IPAddressPool if
	// SharedIP is true, but no SharedIPKey specified.
	SharedIPKey string `json:"sharedIPKey"`

	// +kubebuilder:validation:Optional
	// LoadBalancerIPs, request given IPs from the pool if available. Using a list to allow dual stack (IPv4/IPv6) support
	LoadBalancerIPs []string `json:"loadBalancerIPs,omitempty"`
}
