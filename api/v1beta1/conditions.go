package v1beta1

import "github.com/openstack-k8s-operators/lib-common/modules/common/condition"

const (
	// BarbicanAPIReadyCondition -
	BarbicanAPIReadyCondition condition.Type = "BarbicanAPIReady"

	// BarbicanWorkerReadyCondition -
	BarbicanWorkerReadyCondition condition.Type = "BarbicanWorkerReady"
	// BarbicanRabbitMQTransportURLReadyCondition -
	BarbicanRabbitMQTransportURLReadyCondition condition.Type = "BarbicanRabbitMQTransportURLReady"
)

const (
	// BarbicanAPIReadyInitMessage -
	BarbicanAPIReadyInitMessage = "BarbicanAPI not started"
	// BarbicanAPIReadyErrorMessage -
	BarbicanAPIReadyErrorMessage = "BarbicanAPI error occured %s"
	// BarbicanWorkerReadyInitMessage -
	BarbicanWorkerReadyInitMessage = "BarbicanWorker not started"

	// BarbicanRabbitMQTransportURLReadyRunningMessage -
	BarbicanRabbitMQTransportURLReadyRunningMessage = "BarbicanRabbitMQTransportURL creation in progress"
	// BarbicanRabbitMQTransportURLReadyMessage -
	BarbicanRabbitMQTransportURLReadyMessage = "BarbicanRabbitMQTransportURL successfully created"
	// BarbicanRabbitMQTransportURLReadyErrorMessage -
	BarbicanRabbitMQTransportURLReadyErrorMessage = "BarbicanRabbitMQTransportURL error occured %s"
)
