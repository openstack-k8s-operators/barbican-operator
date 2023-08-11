package barbican

import (
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	corev1 "k8s.io/api/core/v1"
)

// APIDetails information
type APIDetails struct {
	ContainerImage       string
	DatabaseHost         string
	DatabaseUser         string
	DatabaseName         string
	TransportURL         string
	OSPSecret            string
	DBPasswordSelector   string
	UserPasswordSelector string
	VolumeMounts         []corev1.VolumeMount
}

const (
	// InitContainerCommand -
	InitContainerCommand = "/usr/local/bin/container-scripts/init.sh"
)

// InitContainer - init container for barbican api pods
func InitContainer(init APIDetails) []corev1.Container {
	runAsUser := int64(0)

	args := []string{
		"-c",
		InitContainerCommand,
	}

	envVars := map[string]env.Setter{}
	envVars["DatabaseHost"] = env.SetValue(init.DatabaseHost)
	envVars["DatabaseUser"] = env.SetValue(init.DatabaseUser)
	envVars["DatabaseName"] = env.SetValue(init.DatabaseName)

	envs := []corev1.EnvVar{
		{
			Name: "DatabasePassword",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: init.OSPSecret,
					},
					Key: init.DBPasswordSelector,
				},
			},
		},
		{
			Name: "BarbicanPassword",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: init.OSPSecret,
					},
					Key: init.UserPasswordSelector,
				},
			},
		},
		// TODO
		// {
		// 	Name: "TransportUrl",
		// 	ValueFrom: &corev1.EnvVarSource{
		// 		SecretKeyRef: &corev1.SecretKeySelector{
		// 			LocalObjectReference: corev1.LocalObjectReference{
		// 				Name: init.OSPSecret,
		// 			},
		// 			Key: "TransportUrl",
		// 		},
		// 	},
		// },
	}
	envs = env.MergeEnvs(envs, envVars)
	return []corev1.Container{
		{
			Name:  "init",
			Image: init.ContainerImage,
			SecurityContext: &corev1.SecurityContext{
				RunAsUser: &runAsUser,
			},
			Command: []string{
				"/bin/bash",
			},
			Args:         args,
			Env:          envs,
			VolumeMounts: init.VolumeMounts,
		},
	}
}
