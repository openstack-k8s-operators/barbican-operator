package barbican

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// PKCS11PrepJob func
func PKCS11PrepJob(instance *barbicanv1beta1.Barbican, labels map[string]string, annotations map[string]string) *batchv1.Job {
	// The PKCS11 Prep job just needs the main barbican config files, and the files
	// needed to communicate with the relevant HSM. Like the db-sync job it runs
	// barbican-manage, so it uses the selective oslo.config mount (only
	// 00-default.conf and 01-custom.conf) rather than the full config-data
	// directory -- the Apache *.conf files in that Secret are not valid INI and
	// would make barbican-manage fail when oslo.config parses the config-dir.
	// generate_pkcs11_keys.sh also reads p11_crypto_plugin settings via crudini
	// from /etc/barbican/barbican.conf.d/01-custom.conf, which this mount covers.
	pkcs11Volumes := []corev1.Volume{
		GetScriptVolume(instance.Name + "-scripts"),
	}
	osloVolumes, osloMounts := GetOsloConfigVolumes(instance.Name, PKCS11ConfigVolume)
	pkcs11Volumes = append(pkcs11Volumes, osloVolumes...)
	pkcs11Volumes = append(pkcs11Volumes, GetVolumes(instance.Name)...)

	pkcs11Mounts := []corev1.VolumeMount{
		GetScriptVolumeMount(),
	}
	pkcs11Mounts = append(pkcs11Mounts, osloMounts...)
	pkcs11Mounts = append(pkcs11Mounts, GetVolumeMounts()...)

	// add CA cert if defined
	if instance.Spec.BarbicanAPI.TLS.CaBundleSecretName != "" {
		pkcs11Volumes = append(pkcs11Volumes, instance.Spec.BarbicanAPI.TLS.CreateVolume())
		pkcs11Mounts = append(pkcs11Mounts, instance.Spec.BarbicanAPI.TLS.CreateVolumeMounts(nil)...)
	}

	// add the HSM vendor client data. This job only ever runs for the pkcs11
	// secret store, so no other store's client data is mounted here.
	pkcs11Volumes = append(pkcs11Volumes,
		getClientDataVolume(PKCS11ClientDataVolume, instance.Spec.PKCS11.ClientDataSpec))
	pkcs11Mounts = append(pkcs11Mounts,
		getClientDataVolumeMount(PKCS11ClientDataVolume, instance.Spec.PKCS11.ClientDataSpec,
			barbicanv1beta1.DefaultPKCS11ClientDataPath))

	// This job runs as root (not a kolla artifact): the vendor HSM client
	// library setup performed by generate_pkcs11_keys.sh is not verified to
	// work under a non-root UID without real HSM hardware, so root access is
	// kept here as a documented exception -- see docs/remove-kolla-plan.md.
	runAsUser := int64(0)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-pkcs11-prep",
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: ptr.To(false),
					RestartPolicy:                corev1.RestartPolicyOnFailure,
					ServiceAccountName:           instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name: instance.Name + "-pkcs11-prep",
							Command: []string{
								ScriptMountPoint + "/generate_pkcs11_keys.sh",
							},
							Image: instance.Spec.BarbicanAPI.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:          []corev1.EnvVar{},
							VolumeMounts: pkcs11Mounts,
						},
					},
				},
			},
		},
	}

	job.Spec.Template.Spec.Volumes = pkcs11Volumes

	return job
}
