package barbicanapi

import (
	"slices"

	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	barbican "github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumesAndMounts - return the volumes and mounts for a BarbicanAPI deployment
func GetAPIVolumesAndMounts(instance *barbicanv1beta1.BarbicanAPI) ([]corev1.Volume, []corev1.VolumeMount, error) {

	apiVolumes := []corev1.Volume{
		barbican.GetCustomConfigVolume(instance.Name),
		barbican.GetLogVolume(),
	}

	apiVolumeMounts := []corev1.VolumeMount{
		barbican.GetCustomConfigVolumeMount(),
		barbican.GetKollaConfigVolumeMount(instance.Name),
		barbican.GetLogVolumeMount(),
	}

	// prepend general config volumes and mounts
	apiVolumes = append(barbican.GetVolumes("barbican"), apiVolumes...)
	apiVolumeMounts = append(barbican.GetVolumeMounts(), apiVolumeMounts...)

	// add CA cert if defined
	if instance.Spec.TLS.CaBundleSecretName != "" {
		apiVolumes = append(apiVolumes, instance.Spec.TLS.CreateVolume())
		apiVolumeMounts = append(apiVolumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	for _, endpt := range []service.Endpoint{service.EndpointInternal, service.EndpointPublic} {
		if instance.Spec.TLS.API.Enabled(endpt) {
			var tlsEndptCfg tls.GenericService
			switch endpt {
			case service.EndpointPublic:
				tlsEndptCfg = instance.Spec.TLS.API.Public
			case service.EndpointInternal:
				tlsEndptCfg = instance.Spec.TLS.API.Internal
			}

			svc, err := tlsEndptCfg.ToService()
			if err != nil {
				return nil, nil, err
			}
			apiVolumes = append(apiVolumes, svc.CreateVolume(endpt.String()))
			apiVolumeMounts = append(apiVolumeMounts, svc.CreateVolumeMounts(endpt.String())...)
		}
	}

	// Add PKCS11 volumes
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		apiVolumes = append(apiVolumes, barbican.GetHSMVolumes(*instance.Spec.PKCS11)...)
		apiVolumeMounts = append(apiVolumeMounts, barbican.GetHSMVolumeMounts()...)
	}

	return apiVolumes, apiVolumeMounts, nil
}
