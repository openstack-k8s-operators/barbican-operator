package barbicanworker

import (
	"slices"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	barbican "github.com/openstack-k8s-operators/barbican-operator/internal/barbican"
	corev1 "k8s.io/api/core/v1"
)

// GetWorkerVolumesAndMounts returns the volumes and mounts for a BarbicanWorker deployment
func GetWorkerVolumesAndMounts(instance *barbicanv1beta1.BarbicanWorker) ([]corev1.Volume, []corev1.VolumeMount) {
	workerVolumes := []corev1.Volume{
		barbican.GetCustomConfigVolume(instance.Name),
		barbican.GetLogVolume(),
	}

	workerVolumeMounts := []corev1.VolumeMount{
		barbican.GetCustomConfigVolumeMount(),
		barbican.GetKollaConfigVolumeMount(instance.Name),
		barbican.GetLogVolumeMount(),
	}

	// prepend general config volumes and mounts
	workerVolumes = append(barbican.GetVolumes("barbican"), workerVolumes...)
	workerVolumeMounts = append(barbican.GetVolumeMounts(), workerVolumeMounts...)

	// add the CA bundle
	if instance.Spec.TLS.CaBundleSecretName != "" {
		workerVolumes = append(workerVolumes, instance.Spec.TLS.CreateVolume())
		workerVolumeMounts = append(workerVolumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	// Add PKCS11 volumes
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		workerVolumes = append(workerVolumes, barbican.GetHSMVolumes(*instance.Spec.PKCS11)...)
		workerVolumeMounts = append(workerVolumeMounts, barbican.GetHSMVolumeMounts()...)
	}

	return workerVolumes, workerVolumeMounts
}
