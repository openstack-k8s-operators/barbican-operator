package barbican

import (
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var (
	configMode int32 = 0640
	scriptMode int32 = 0740
)

// GetVolumes - service volumes
func GetVolumes(name string) []corev1.Volume {
	var config0644AccessMode int32 = 0644

	return []corev1.Volume{
		{
			Name: ConfigVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  name + "-config-data",
				},
			},
		},
	}
}

// GetVolumeMounts - general VolumeMounts
func GetVolumeMounts() []corev1.VolumeMount {

	return []corev1.VolumeMount{
		{
			Name:      ConfigVolume,
			MountPath: ConfigMountPoint,
			ReadOnly:  true,
		},
		{
			Name:      ConfigVolume,
			MountPath: "/etc/my.cnf",
			SubPath:   "my.cnf",
			ReadOnly:  true,
		},
	}
}

// GetLogVolumeMount - Returns the VolumeMount used for logging purposes
func GetLogVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      LogVolume,
		MountPath: "/var/log/barbican",
		ReadOnly:  false,
	}
}

// GetLogVolume - Returns the Volume used for logging purposes
func GetLogVolume() corev1.Volume {
	return corev1.Volume{
		Name: LogVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{Medium: ""},
		},
	}
}

// GetScriptVolumeMount - Returns the VolumeMount for scripts
func GetScriptVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      ScriptVolume,
		MountPath: ScriptMountPoint,
		ReadOnly:  true,
	}
}

// GetScriptVolume - Return the Volume for scripts
func GetScriptVolume(secretName string) corev1.Volume {
	return corev1.Volume{
		Name: ScriptVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &scriptMode,
				SecretName:  secretName,
			},
		},
	}
}

// GetKollaConfigVolumeMount - Returns the VolumeMount for the kolla config file
func GetKollaConfigVolumeMount(serviceName string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      ConfigVolume,
		MountPath: "/var/lib/kolla/config_files/config.json",
		SubPath:   serviceName + "-config.json",
		ReadOnly:  true,
	}
}

// GetHSMVolume - Returns Volumes for HSM secrets
func GetHSMVolumes(pkcs11 barbicanv1beta1.BarbicanPKCS11Template) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: PKCS11ClientDataVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &configMode,
					SecretName:  pkcs11.ClientDataSecret,
				},
			},
		},
	}
}

// GetHSMVolumeMount - Returns Volume Mounts for HSM secrets
func GetHSMVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      PKCS11ClientDataVolume,
			MountPath: PKCS11ClientDataMountPoint,
			ReadOnly:  true,
		},
	}
}

// GetCustomConfigVolume - service custom config volume
func GetCustomConfigVolume(name string) corev1.Volume {
	var config0644AccessMode int32 = 0644

	return corev1.Volume{
		Name: CustomConfigVolume,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				DefaultMode: &config0644AccessMode,
				SecretName:  name + "-config-data",
			},
		},
	}
}

// GetCustomConfigVolumeMount - service custom config volume mount
func GetCustomConfigVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      CustomConfigVolume,
		MountPath: CustomConfigMountPoint,
		ReadOnly:  true,
	}
}

// GetDBSyncVolumes - dbsync volumes
// Unlike the individual Barbican services, the DbSyncJob doesn't need a
// secret that contains all of the config snippets required by every
// service, The two snippet files that it does need (DefaultsConfigFileName
// and CustomConfigFileName) can be extracted from the top-level barbican
// config-data secret.
func GetDBSyncVolumes(name string) ([]corev1.Volume, []corev1.VolumeMount) {
	var config0644AccessMode int32 = 0644
	dbSyncVolumes := []corev1.Volume{
		{
			Name: "db-sync-config-data",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  name + "-config-data",
					Items: []corev1.KeyToPath{
						{
							Key:  DefaultsConfigFileName,
							Path: DefaultsConfigFileName,
						},
						{
							Key:  CustomConfigFileName,
							Path: CustomConfigFileName,
						},
					},
				},
			},
		},
	}

	dbSyncVolumeMount := []corev1.VolumeMount{
		{
			Name:      "db-sync-config-data",
			MountPath: "/etc/barbican/barbican.conf.d",
			ReadOnly:  true,
		},
	}
	return dbSyncVolumes, dbSyncVolumeMount
}
