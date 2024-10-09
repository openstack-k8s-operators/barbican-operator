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
}

// +kubebuilder:validation:Enum=simple_crypto;pkcs11
// This SecretStore type is used by the EnabledSecretStores variable inside the specification.
type SecretStore string

// BarbicanPKCS11Template - Includes all common HSM properties
type BarbicanPKCS11Template struct {
        // +kubebuilder:validation:Required
        // +kubebuilder:validation:Items:Enum=luna
        // A string containing the HSM type (currently supported: "luna").
        Type string `json:"type"`

        // +kubebuilder:validation:Required
	// Path to vendor's PKCS11 library
	LibraryPath string `json:"libraryPath"`

        // +kubebuilder:validation:Optional
        // Token serial number used to identify the token to be used.
	// One of TokenSerialNumber, TokenLabels or SlotId must
	// be defined.  TokenSerialNumber takes priority over
	// TokenLabels and SlotId
	TokenSerialNumber string `json:"tokenSerialNumber,omitempty"`

        // +kubebuilder:validation:Optional
	// Token labels used to identify the token to be used.
	// One of TokenSerialNumber, TokenLabels or SlotId must
	// be specified. TokenLabels takes priority over SlotId.
	// This can be a comma separated string of labels
	TokenLabels string `json:"tokenLabels,omitempty"`

        // +kubebuilder:validation:Optional
	// One of TokenSerialNumber, TokenLabels or SlotId must
	// be defined.  SlotId is used if none of the others is defined
        SlotId string `json:"slotId,omitempty"`

        // +kubebuilder:validation:Required
	// Label to identify master KEK in the HSM (must not be the same as HMAC label)
	MKEKLabel string `json:"MKEKLabel"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=32
	// Length in bytes of master KEK
	MKEKLength int `json:"MKEKLength"`

        // +kubebuilder:validation:Required
        // Label to identify HMAC key in the HSM (must not be the same as MKEK label)
        HMACLabel string `json:"HMACLabel"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=CKK_GENERIC_SECRET
	// HMAC Key Type
	HMACKeyType string `json:"HMACKeyType"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=CKM_GENERIC_SECRET_KEY_GEN
	// HMAC Keygen Mechanism
	HMACKeygenMechanism string `json:"HMACKeygenMechanism"`

        // +kubebuilder:validation:Optional
	// +kubebuilder:default=CKM_SHA256_HMAC
	// HMAC Mechanism. This replaces hsm_keywrap_mechanism
	HMACMechanism string `json:"HMACMechanism"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=4
        // +kubebuilder:validation:Maximum=7
        // +kubebuilder:validation:Minimum=0
        // Level of logging, where 0 means "no logging" and 7 means "debug".
        LoggingLevel int `json:"loggingLevel"`

	// +kubebuilder:validation:Required
	// The HSM's IPv4 address (X.Y.Z.K)
	ServerAddress string `json:"serverAddress"`

	// +kubebuilder:validation:Optional
	// The IP address of the client connecting to the HSM (X.Y.Z.K)
	ClientAddress string `json:"clientAddress,omitempty"`

        // +kubebuilder:validation:Required
        // OpenShift secret that stores the password to login to the PKCS11 session
        LoginSecret string `json:"loginSecret"`

        // +kubebuilder:validation:Optional
        // The OpenShift secret that stores the HSM certificates.
        CertificatesSecret string `json:"certificatesSecret,omitempty"`

        // +kubebuilder:validation:Optional
        // The mounting point where the certificates will be copied to (e.g., /usr/local/luna/config/certs).
	CertificatesMountPoint string `json:"certificatesMountPoint,omitempty"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=CKM_AES_GCM
        // Secret encryption mechanism
	EncryptionMechanism string `json:"encryptionMechanism"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=CKM_AES_KEY_WRAP_KWP
        // Key wrap mechanism
	KeyWrapMechanism string `json:"keyWrapMechanism"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=true
        // Generate IVs for the key wrap mechanism
	KeyWrapGenerateIV bool `json:"keyWrapGenerateIV"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=true
        // Generate IVs for CKM_AES_GCM mechanism
	AESGCMGenerateIV bool `json:"AESGCMGenerateIV"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=true
        // Always set cka_sensitive
	AlwaysSetCKASensitive bool `json:"alwaysSetCKASensitive"`

        // +kubebuilder:validation:Optional
        // +kubebuilder:default=false
        // Set os_locking_ok
	OSLockingOK  bool `json:"OSLockingOK"`
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
