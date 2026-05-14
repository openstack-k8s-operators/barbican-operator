package barbicankeystonelistener

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	barbican "github.com/openstack-k8s-operators/barbican-operator/internal/barbican"
	corev1 "k8s.io/api/core/v1"
)

// GetListenerVolumesAndMounts returns the volumes and mounts for a BarbicanKeystoneListener deployment.
// overwriteKeys lists the defaultConfigOverwrite filenames that need SubPath
// mounts into /etc/barbican/ (e.g. policy.yaml).
func GetListenerVolumesAndMounts(instance *barbicanv1beta1.BarbicanKeystoneListener, overwriteKeys []string) ([]corev1.Volume, []corev1.VolumeMount) {
	listenerVolumes := []corev1.Volume{
		barbican.GetCustomConfigVolume(instance.Name),
		barbican.GetLogVolume(),
	}

	listenerVolumeMounts := []corev1.VolumeMount{
		barbican.GetCustomConfigVolumeMount(),
		barbican.GetKollaConfigVolumeMount(instance.Name),
		barbican.GetLogVolumeMount(),
	}
	listenerVolumeMounts = append(listenerVolumeMounts, barbican.GetConfigOverwriteVolumeMounts(overwriteKeys)...)

	// prepend general config volumes and mounts
	listenerVolumes = append(barbican.GetVolumes("barbican"), listenerVolumes...)
	listenerVolumeMounts = append(barbican.GetVolumeMounts(), listenerVolumeMounts...)

	// add the CA bundle
	if instance.Spec.TLS.CaBundleSecretName != "" {
		listenerVolumes = append(listenerVolumes, instance.Spec.TLS.CreateVolume())
		listenerVolumeMounts = append(listenerVolumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	return listenerVolumes, listenerVolumeMounts
}
