package v1beta1

import "github.com/openstack-k8s-operators/lib-common/modules/common/condition"

const (
	BarbicanAPIReadyCondition    condition.Type = "BarbicanAPIReady"
	BarbicanWorkerReadyCondition condition.Type = "BarbicanWorkerReady"
)

const (
	BarbicanAPIReadyInitMessage    = "BarbicanAPI not started"
	BarbicanWorkerReadyInitMessage = "BarbicanWorker not started"
)
