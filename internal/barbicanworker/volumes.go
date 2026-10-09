package barbicanworker

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	barbican "github.com/openstack-k8s-operators/barbican-operator/internal/barbican"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

// GetWorkerVolumesAndMounts returns the volumes and mounts for a BarbicanWorker deployment.
// overwriteKeys lists the defaultConfigOverwrite filenames that need SubPath
// mounts into /etc/barbican/ (e.g. policy.yaml).
func GetWorkerVolumesAndMounts(instance *barbicanv1beta1.BarbicanWorker, overwriteKeys []string) ([]corev1.Volume, []corev1.VolumeMount) {
	workerVolumes := []corev1.Volume{
		barbican.GetCustomConfigVolume(instance.Name),
		barbican.GetLogVolume(),
	}

	workerVolumeMounts := []corev1.VolumeMount{
		barbican.GetCustomConfigVolumeMount(),
		barbican.GetLogVolumeMount(),
	}
	workerVolumeMounts = append(workerVolumeMounts, barbican.GetConfigOverwriteVolumeMounts(overwriteKeys)...)

	// prepend general config volumes and mounts
	workerVolumes = append(barbican.GetVolumes("barbican"), workerVolumes...)
	workerVolumeMounts = append(barbican.GetVolumeMounts(), workerVolumeMounts...)

	// add the CA bundle
	if instance.Spec.TLS.CaBundleSecretName != "" {
		workerVolumes = append(workerVolumes, instance.Spec.TLS.CreateVolume())
		workerVolumeMounts = append(workerVolumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	// Add the client data volumes of the enabled secret stores
	clientDataVols, clientDataMounts := barbican.GetClientDataVolumes(&instance.Spec.BarbicanTemplate)
	workerVolumes = append(workerVolumes, clientDataVols...)
	workerVolumeMounts = append(workerVolumeMounts, clientDataMounts...)

	// ExtraMounts
	extraVols, extraMounts := barbican.GetExtraVolumes(
		instance.Spec.ExtraMounts,
		[]storage.PropagationType{barbican.BarbicanWorker, barbican.Barbican},
	)
	workerVolumes = append(workerVolumes, extraVols...)
	workerVolumeMounts = append(workerVolumeMounts, extraMounts...)

	return workerVolumes, workerVolumeMounts
}
