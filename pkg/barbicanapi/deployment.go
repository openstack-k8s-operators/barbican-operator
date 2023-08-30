package barbicanapi

import (
	"fmt"

	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	barbican "github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
)

const (
	// ServiceCommand -
	ServiceCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"
)

func Deployment(
	instance *barbicanv1beta1.BarbicanAPI,
	configHash string,
	labels map[string]string,
	annotations map[string]string,
) *appsv1.Deployment {
	runAsUser := int64(0)
	var config0644AccessMode int32 = 0644
	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["CONFIG_HASH"] = env.SetValue(configHash)
	livenessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      5,
		PeriodSeconds:       3,
		InitialDelaySeconds: 5,
	}
	readinessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		InitialDelaySeconds: 5,
	}
	args := []string{"-c"}
	if instance.Spec.Debug.Service {
		args = append(args, common.DebugCommand)
		livenessProbe.Exec = &corev1.ExecAction{
			Command: []string{
				"/bin/true",
			},
		}
		readinessProbe.Exec = livenessProbe.Exec
	} else {
		args = append(args, ServiceCommand)
		//
		// https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
		//
		livenessProbe.HTTPGet = &corev1.HTTPGetAction{
			Path: "/healthcheck",
			Port: intstr.IntOrString{Type: intstr.Int, IntVal: int32(barbican.BarbicanPublicPort)},
		}
		readinessProbe.HTTPGet = livenessProbe.HTTPGet
	}

	apiVolumes := []corev1.Volume{
		{
			Name: "config-data-custom",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  instance.Name + "-config-data",
				},
			},
		},
	}

	apiVolumes = append(apiVolumes, barbican.GetLogVolume()...)
	apiVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data",
			MountPath: "/var/lib/kolla/config_files/config.json",
			SubPath:   "barbican-api-config.json",
			ReadOnly:  true,
		},
	}
	// Append LogVolume to the apiVolumes: this will be used to stream
	// logging
	apiVolumeMounts = append(apiVolumeMounts, barbican.GetLogVolumeMount()...)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: instance.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: instance.Spec.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name: instance.Name + "-log",
							Command: []string{
								"/bin/bash",
							},
							Args:  []string{"-c", "tail -n+1 -F " + barbican.BarbicanLogPath + instance.Name + ".log"},
							Image: instance.Spec.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:            env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:   barbican.GetLogVolumeMount(),
							Resources:      instance.Spec.Resources,
							ReadinessProbe: readinessProbe,
							LivenessProbe:  livenessProbe,
						},
						{
							Name: barbican.ServiceName + "-api",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env: env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: append(barbican.GetVolumeMounts(
								instance.Spec.CustomServiceConfigSecrets,
								barbican.BarbicanAPIPropagation),
								apiVolumeMounts...,
							),
							Resources:      instance.Spec.Resources,
							ReadinessProbe: readinessProbe,
							LivenessProbe:  livenessProbe,
						},
					},
				},
			},
		},
	}
	deployment.Spec.Template.Spec.Volumes = append(barbican.GetVolumes(
		instance.Name,
		barbican.ServiceName,
		instance.Spec.CustomServiceConfigSecrets,
		barbican.BarbicanAPIPropagation),
		apiVolumes...)

	return deployment
}
