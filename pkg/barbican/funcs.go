package barbican

import (
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/affinity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
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

// GetPodAffinity - Returns a corev1.Affinity reference for the specified component.
func GetPodAffinity(componentName string) *corev1.Affinity {
	// If possible two pods of the same component (e.g barbican-worker) should not
	// run on the same worker node. If this is not possible they get still
	// created on the same worker node.
	return affinity.DistributePods(
		common.ComponentSelector,
		[]string{
			componentName,
		},
		corev1.LabelHostname,
	)
}

// BaseSecurityContext - currently used to make sure we don't run cronJob and Log
// Pods as root user, and we drop privileges and Capabilities we don't need
func BaseSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		RunAsUser:                ptr.To(BarbicanUID),
		RunAsGroup:               ptr.To(BarbicanGID),
		RunAsNonRoot:             ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

// HttpdSecurityContext -
func HttpdSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"MKNOD",
			},
		},
		RunAsUser:  ptr.To(BarbicanUID),
		RunAsGroup: ptr.To(BarbicanGID),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

// dbSyncSecurityContext - currently used to make sure we don't run db-sync as
// root user
func dbSyncSecurityContext() *corev1.SecurityContext {

	return &corev1.SecurityContext{
		RunAsUser:  ptr.To(BarbicanUID),
		RunAsGroup: ptr.To(BarbicanGID),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"MKNOD",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}
