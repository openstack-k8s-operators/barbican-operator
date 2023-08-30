package barbican

import "github.com/openstack-k8s-operators/lib-common/modules/storage"

const (
	// ServiceName -
	ServiceName = "barbican"
	// DatabaseName -
	DatabaseName = "barbican"
	// BarbicanPublicPort -
	BarbicanPublicPort int32 = 9311
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
