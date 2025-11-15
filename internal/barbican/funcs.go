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

// GetLogSecurityContext - do not run Log containers as root user and drop
// all privileges and Capabilities
func GetLogSecurityContext() *corev1.SecurityContext {
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
	}
}

// GetBaseSecurityContext - security Context for an httpd that runs barbican-api
func GetBaseSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"MKNOD",
			},
		},
		RunAsUser:  ptr.To(BarbicanUID),
		RunAsGroup: ptr.To(BarbicanGID),
	}
}

// GetServiceSecurityContext - It defined and returns a securityContext for a
// barbican service (keystone-listener and worker)
func GetServiceSecurityContext(privileged bool) *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(true),
		RunAsUser:                ptr.To(BarbicanUID),
		RunAsGroup:               ptr.To(BarbicanGID),
		Privileged:               &privileged,
	}
}
