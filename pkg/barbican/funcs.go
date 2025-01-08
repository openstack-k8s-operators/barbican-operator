package barbican

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOwningBarbicanName - Given a BarbicanAPI, BarbicanKeystoneListener or BarbicanWorker
// object, returning the parent Barbican object that created it (if any)
func GetOwningBarbicanName(instance client.Object) string {
	for _, ownerRef := range instance.GetOwnerReferences() {
		if ownerRef.Kind == "Barbican" {
			return ownerRef.Name
		}
	}

	return ""
}
