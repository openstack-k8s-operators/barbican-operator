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
	CustomServiceConfigFileName = "02-service.conf"
	// CustomServiceConfigSecretsFileName -
	CustomServiceConfigSecretsFileName = "03-secrets.conf"
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
	// LogVolume is the default logVolume name used to mount logs on both
	// BarbicanAPI and the sidecar container
	LogVolume = "logs"
)

// DbsyncPropagation keeps track of the DBSync Service Propagation Type
var DbsyncPropagation = []storage.PropagationType{storage.DBSync}

// BarbicanAPIPropagation is the  definition of the BarbicanAPI propagation group
// It allows the BarbicanAPI pod to mount volumes destined to Barbican related
// ServiceTypes
var BarbicanAPIPropagation = []storage.PropagationType{Barbican, BarbicanAPI}

// BarbicanWorkerPropagation is the  definition of the BarbicanWorker propagation group
// It allows the BarbicanWorker pod to mount volumes destined to Barbican related
// ServiceTypes
var BarbicanWorkerPropagation = []storage.PropagationType{Barbican, BarbicanWorker}

// BarbicanKeystoneListenerPropagation is the  definition of the BarbicanKeystoneListener propagation group
// It allows the BarbicanKeystoneListener pod to mount volumes destined to Barbican related
// ServiceTypes
var BarbicanKeystoneListenerPropagation = []storage.PropagationType{Barbican, BarbicanKeystoneListener}
