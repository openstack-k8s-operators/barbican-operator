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

// BarbicanLunaTemplate - Variables specific to Thales' Luna HSMs
type BarbicanLunaTemplate struct {
        // +kubebuilder:validation:Optional
        // +kubebuilder:default=4
        // +kubebuilder:validation:Maximum=7
        // +kubebuilder:validation:Minimum=0
        // Level of logging, where 0 means "no logging" and 7 means "debug".
        LunaLoggingLevel int `json:"lunaLoggingLevel"`

        // +kubebuilder:validation:Optional
        // The HSM certificates. The map's key is the HSM's IP address and
        // the value is the OpenShift secret storing the certificate.
        LunaHSMCertificates map[string]string `json:"lunaHsmCertificates"`

        // +kubebuilder:validation:Optional
        // The OpenShift secret storing the client certificate plus its key
        LunaClientCertificate string `json:"lunaClientCertificate"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default="/usr/lib/libCryptoki2_64.so"
	// Path to vendor PKCS11 library
	LunaLibraryPath string `json:"lunaLibraryPath"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default="12345678"
	// Token serial number used to identify the token to be used.  Required
	// when the device has multiple tokens with the same label.
	LunaTokenSerialNumber string `json:"lunaTokenSerialNumber"`

        // +kubebuilder:validation:Optional
	// Token label used to identify the token to be used.  Required when
	// token_serial_number is not specified.
	LunaTokenLabel string `json:"lunaTokenLabel"`

        // +kubebuilder:validation:Optional
	// The OpenShift secret containing the password to login to PKCS11 session
	LunaPassword string `json:"lunaPassword"`

        // +kubebuilder:validation:Optional
	// Label to identify master KEK in the HSM (must not be the same as HMAC label)
	LunaMkekLabel string `json:"lunaMkekLabel"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=32
	// Length in bytes of master KEK
	LunaMkekLength int `json:"lunaMkekLength"`

        // +kubebuilder:validation:Optional
	// Label to identify HMAC key in the HSM (must not be the same as MKEK label)
	LunaHmacLabel string `json:"lunaHmacLabel"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// HSM Slot ID that contains the token device to be used
	LunaSlotId int `json:"lunaSlotId"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=true
	// Enable Read/Write session with the HSM?
	LunaRwSession bool `json:"lunaRwSession"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=32
	// Length of Project KEKs to create
	LunaPkekLength int `json:"lunaPkekLength"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=900
	// How long to cache unwrapped Project KEKs
	LunaPkekCacheTtl int `json:"lunaPkekCacheTtl"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=100
	// Max number of items in pkek cache
	LunaPkekCacheLimit int `json:"lunaPkekCacheLimit"`
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
