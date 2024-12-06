package barbican

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"

	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// P11PrepCommand -
	P11PrepCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"
	P11PrepConfig  = "p11-prep-config-data"
)

// P11PrepJob func
func P11PrepJob(instance *barbicanv1beta1.Barbican, labels map[string]string, annotations map[string]string) *batchv1.Job {
	secretNames := []string{}

	// The P11 Prep job just needs the main barbican config files, and the files
	// needed to communicate with the relevant HSM.
	p11Volumes := []corev1.Volume{
		GetScriptVolume(instance.Name + "-scripts"),
	}
	p11Volumes = append(p11Volumes, GetVolumes(instance.Name, secretNames)...)

	p11Mounts := []corev1.VolumeMount{
		GetKollaConfigVolumeMount(instance.Name + "-p11-prep"),
		GetScriptVolumeMount(),
	}
	p11Mounts = append(p11Mounts, GetVolumeMounts(secretNames)...)

	// add CA cert if defined
	if instance.Spec.BarbicanAPI.TLS.CaBundleSecretName != "" {
		p11Volumes = append(p11Volumes, instance.Spec.BarbicanAPI.TLS.CreateVolume())
		p11Mounts = append(p11Mounts, instance.Spec.BarbicanAPI.TLS.CreateVolumeMounts(nil)...)
	}

	// add any HSM volumes
	p11Volumes = append(p11Volumes, GetHSMVolumes(*instance.Spec.PKCS11)...)
	p11Mounts = append(p11Mounts, GetHSMVolumeMounts(*instance.Spec.PKCS11)...)

	// add luna specific config files

	args := []string{"-c", P11PrepCommand}

	runAsUser := int64(0)
	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("TRUE")

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-p11-prep",
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name: instance.Name + "-p11-prep",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.BarbicanAPI.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: p11Mounts,
						},
					},
				},
			},
		},
	}

	job.Spec.Template.Spec.Volumes = p11Volumes

	return job
}
