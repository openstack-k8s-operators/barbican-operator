package barbican

const (
	// ServiceName -
	ServiceName = "barbican"
	// DatabaseName -
	DatabaseName = "barbican"
	// KollaConfigDbSync -
	KollaConfigDbSync = "/var/lib/config-data/merged/barbican-dbsync-config.json"

	// DefaultsConfigFileName -
	DefaultsConfigFileName = "00-default.conf"
	// CustomConfigFileName -
	CustomConfigFileName = "01-custom.conf"
	// CustomServiceConfigFileName -
	CustomServiceConfigFileName = "02-service.conf"
	// CustomServiceConfigSecretsFileName -
	CustomServiceConfigSecretsFileName = "03-secrets.conf"
	// LogVolume is the default logVolume name used to mount logs on both
	// BarbicanAPI and the sidecar container
	LogVolume = "logs"
)
