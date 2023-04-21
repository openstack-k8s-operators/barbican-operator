package v1beta1

import "github.com/openstack-k8s-operators/lib-common/modules/common/condition"

const (
	BarbicanAPIReadyCondition    condition.Type = "BarbicanAPIReady"
	BarbicanWorkerReadyCondition condition.Type = "BarbicanWorkerReady"

	BarbicanRabbitMQTransportURLReadyCondition condition.Type = "BarbicanRabbitMQTransportURLReady"
)

const (
	BarbicanAPIReadyInitMessage    = "BarbicanAPI not started"
	BarbicanWorkerReadyInitMessage = "BarbicanWorker not started"

	BarbicanRabbitMQTransportURLReadyRunningMessage = "BarbicanRabbitMQTransportURL creation in progress"
	BarbicanRabbitMQTransportURLReadyMessage        = "BarbicanRabbitMQTransportURL successfully created"
	BarbicanRabbitMQTransportURLReadyErrorMessage   = "BarbicanRabbitMQTransportURL error occured %s"
)
