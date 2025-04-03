# Volumes and Secrets Mounted by Barbican Pods

In general, configuration is mounted in the barbican pods from secrets.
This doc briefly describes which secrets are mounted and where.

## Relevant Spec Fields

- Barbican.Spec.PKCS11.ClientDataSecret []string
- Barbican.Spec.BarbicanAPI.CustomServiceConfig (string)
- Barbican.Spec.BarbicanAPI.DefaultConfigOverwrite map[string]string
- Barbican.Spec.BarbicanAPI.CustomServiceConfigSecrets []string
- Barbican.Spec.BarbicanWorker.CustomServiceConfig (string)
- Barbican.Spec.BarbicanWorker.DefaultConfigOverwrite map[string]string
- Barbican.Spec.BarbicanWorker.CustomServiceConfigSecrets []string
- Barbican.Spec.BarbicanKeystoneListener.CustomServiceConfig (string)
- Barbican.Spec.BarbicanKeystoneListener.DefaultConfigOverwrite map[string]string
- Barbican.Spec.BarbicanKeystoneListener.CustomServiceConfigSecrets []string

## Secrets

### secret: barbican-config-data
- contains the files produced from templates in templates/barbican/config
- mounted to /var/lib/config-data/default
- some of these files are then copied via kolla to other locations
- contains kolla config JSON files for all the components
- contains 01-default.conf which is an oslo.config file that contains global service
  defaults.  This file is copied by the controller to the barbican-<component>-config-data
  secret, from whence it is mounted to /etc/barbican.conf.d
- contains 01-custom.conf which is are the contents of Barbican.Spec.CustomServiceConfig
  This is global custom service config (eg. pkcs11 settings in barbican.conf)
  This file is copied y the controller to the barbican-<component>-config-data secret,
  from whence it is mounted to /etc/barbican.conf.d
- contains the files which are the contents of Barbican.Spec.DefaultConfigOverwrite
  These are global config overrides (so things like policy.json for instance)
  As above, these file are copied by the controller to the barbican-<component>-config-data
  secret, from whence they are mounted to /etc/barbican.conf.d

### secret: barbican-scripts
- contents of files produced by templates in templates/barbican/scripts
- Right now, this is only mounted in the pkcs11-prep pod which uses generate_pkcs11_keys.sh.
- mounted at /usr/local/bin/container-scripts

### secret: barbican-api-config-data
- contents of files produced by templates in templates/barbicanapi/config
- mounted at /etc/barbican/barbican.conf.d
- contains files copied by the operator from the barbican-config-data secret (see above)
  These are global config settings.
- contains 01-service-defaults.conf which has service specific defaults (like the log name)
- contains 02-service-custom.conf which has the contents of Barbican.Spec.BarbicanAPI.CustomServiceConfig
  This is barbicanapi service specific barbican.conf config.
- contains 03-secrets-custom.conf which contains the contents of the secrets listed
  in Barbican.Spec.BarbicanAPI.CustomServiceConfigSecrets.  This is described further below.
- contains the files which are the contents of Barbican.Spec.BarbicanAPI.DefaultConfigOverwrite
  These are service specific overrides (so things like policy.json for instance).  If the same
  file is defined in both Barbican.Spec.BarbicanAPI.DefaultConfigOverwrite and Barbican.Spec.DefaultConfigOverwrite,
  the contents defined in Barbican.Spec.BarbicanAPI.DefaultConfigOverwrite will be used.
  There is no merging of data.

### secret: barbican-worker-config-data
- same as above for barbican-api-config-data just for barbican-worker.

### secret: barbican-keystone-listener-config-data
- same as above for barbican-api-config-data just for barbican-keystone-listener.

### secret: Barbican.Spec.PKCS11.ClientDataSecret
- contains pkcs11 secret material
- mounted to /var/lib/config-data/hsm
- copied by kolla to Barbican.Spec.PKCS11.ClientDataPath

### secrets: Barbican.Spec.BarbicanAPI.CustomServiceConfigSecrets
- This is a list of secrets that contain oslo.config style config snippets.
- The idea here is that the config snippets contain secret parameters (like passwords)
  inside and so should not be included in the spec in DefaultConfigOverwrite
- The contents are read from the secret and are concatenated into a single file -
  03-secrets-custom.conf which is added to the secret barbican-api-config-data,
  and thereby mounted to /etc/barbican.conf.d.

### secrets: Barbican.Spec.BarbicanWorker.CustomServiceConfigSecrets
- same as above for Barbican.Spec.BarbicanAPI.CustomServiceConfigSecrets just for the
  barbican-worker pod.

### secrets: Barbican.Spec.BarbicanKeystoneListener.CustomServiceConfigSecrets
- same as above for Barbican.Spec.BarbicanAPI.CustomServiceConfigSecrets just for the
  barbican-keystone-listener pod.
