package barbican

import "github.com/openstack-k8s-operators/lib-common/modules/storage"

const (
	// ServiceName -
	ServiceName = "barbican"
	// ComponentAPI -
	ComponentAPI = "barbican-api"
	// ComponentKeystoneListener -
	ComponentKeystoneListener = "keystone-listener"
	// ComponentWorker -
	ComponentWorker = "barbican-worker"
	// ServiceType -
	ServiceType = "key-manager"

	// DatabaseName - Name of the database used in CREATE DATABASE statement
	DatabaseName = "barbican"

	// DatabaseCRName - Name of the MariaDBDatabase CR
	DatabaseCRName = "barbican"

	// DatabaseUsernamePrefix - used by EnsureMariaDBAccount when a new username
	// is to be generated, e.g. "barbican_e5a4", "barbican_78bc", etc
	DatabaseUsernamePrefix = "barbican"

	// BarbicanPublicPort -
	BarbicanPublicPort int32 = 9311
	// BarbicanInternalPort -
	BarbicanInternalPort int32 = 9311
	// DefaultsConfigFileName -
	DefaultsConfigFileName = "00-default.conf"
	// CustomConfigFileName -
	CustomConfigFileName = "01-custom.conf"
	// CustomServiceConfigFileName -
	CustomServiceConfigFileName = "02-service-custom.conf"
	// CustomServiceConfigSecretsFileName -
	CustomServiceConfigSecretsFileName = "03-secrets-custom.conf"
	// BarbicanAPI defines the barbican-api group
	BarbicanAPI storage.PropagationType = "BarbicanAPI"
	// BarbicanWorker defines the barbican-worker group
	BarbicanWorker storage.PropagationType = "BarbicanWorker"
	// BarbicanKeystoneListener defines the barbican-keystone-listener group
	BarbicanKeystoneListener storage.PropagationType = "BarbicanKeystoneListener"
	// Barbican is the global ServiceType that refers to all the components deployed
	// by the barbican operator
	Barbican storage.PropagationType = "Barbican"
	// BarbicanLogPath is the path used by BarbicanAPI to stream/store its logs
	BarbicanLogPath = "/var/log/barbican/"
	// LogVolume is the default volume name used to mount logs
	LogVolume = "logs"
	// ConfigVolume is the default volume name used to mount service config
	ConfigVolume = "config-data"
	// ConfigMountPoint is the mount point for service config
	ConfigMountPoint = "/var/lib/config-data/default"
	// ScriptVolume is the default volume name used to mount scripts
	ScriptVolume = "scripts"
	// ScriptMountPoint is the mount point for scripts
	ScriptMountPoint = "/usr/local/bin/container-scripts"
	// PKCS11DataVolume is the volume used to mount PKCS11 client Data
	PKCS11ClientDataVolume = "pkcs11-client-data"
	// PKCS11DataVolume is the mount point used for PKCS11 client Data
	PKCS11ClientDataMountPoint = "/var/lib/config-data/hsm"
)
