# Barbican HTTPD Configuration Overrides

The barbican-operator provides mechanisms to customize the Apache HTTPD server
configuration through the use of custom configuration files. This feature
leverages the
[ExtraMounts](https://github.com/openstack-k8s-operators/dev-docs/blob/main/extra_mounts.md)
functionality to mount custom HTTPD configuration files into the Barbican
deployment.

## How It Works

1. **Custom Configuration Files**: Create HTTPD configuration files with your
   custom settings
2. **ConfigMap**: Create ConfigMaps from files containing the overrides
3. **OpenStackControlPlane Patch**: Patch the control plane to mount the
   generated ConfigMap into Barbican containers. The HTTPD configuration
   automatically includes files mounted to `/etc/httpd/conf_custom/*.conf`


### Step 1: Create Custom HTTPD Configuration

Create your custom HTTPD configuration file(s). Any file with a `.conf`
extension placed in `/etc/httpd/conf_custom/` will be automatically included
by the VirtualHost configuration. We recommend naming files with an
`httpd_custom_` prefix for clarity (e.g. `httpd_custom_ssl.conf`).

Example (`httpd_custom_ssl.conf`):
```apache
# TLS 1.2 + 1.3 (compatible default; see note below for PQC-strict mode)
SSLProtocol -all +TLSv1.2 +TLSv1.3

# TLS 1.2 — strong classical AEAD ciphers with ECDHE
SSLCipherSuite ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305

# TLS 1.3 — 256-bit symmetric suites (quantum-resistant at symmetric level)
SSLCipherSuite TLSv1.3 TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
```

> **PQC Note:** For environments with post-quantum cryptography requirements,
> restrict to TLS 1.3 only (`SSLProtocol -all +TLSv1.3`). TLS 1.3 is the
> minimum version that supports hybrid PQC key exchange such as X25519MLKEM768.

### Step 2. Create a ConfigMap

Create a Kubernetes `ConfigMap` containing your custom configuration files:

```bash
oc create configmap httpd-overrides --from-file=httpd_custom_ssl.conf
```

It is possible to add multiple configuration files containing dedicated
configuration directives:

```bash
oc create configmap httpd-overrides \
  --from-file=httpd_custom_ssl.conf \
  --from-file=httpd_custom_timeout.conf
```

### Step 3: Configure ExtraMounts in the OpenStackControlPlane

Update your `OpenStackControlPlane` resource to include the custom HTTPD
configuration files using `extraMounts`. The simplest approach is to mount
the entire ConfigMap to the target `/etc/httpd/conf_custom` mount point:

```yaml
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  barbican:
    enabled: true
    template:
      extraMounts:
      - extraVol:
        - extraVolType: httpd-overrides
          propagation:
          - BarbicanAPI
          mounts:
          - mountPath: /etc/httpd/conf_custom
            name: httpd-overrides
            readOnly: true
          volumes:
          - configMap:
              name: httpd-overrides
            name: httpd-overrides
```

The `propagation` field controls which Barbican sub-components receive the
mount. Valid values are `BarbicanAPI`, `BarbicanWorker`,
`BarbicanKeystoneListener`, and `Barbican` (all components).

## Common Use Cases

- **PQC Readiness**: Enforce TLS 1.3 to enable post-quantum hybrid key exchange when the underlying OpenSSL supports it (e.g., X25519MLKEM768 via oqs-provider)
- **SSL/TLS Hardening**: Override SSLProtocol and SSLCipherSuite directives
- **Timeout Adjustments**: Modify request timeout values for specific environments
- **Security Headers**: Add custom security headers or configurations
- **Logging**: Customize Apache logging configuration
- **Performance Tuning**: Adjust worker processes, connection limits, etc.

## Verification

After deploying your custom `HTTPD` configuration, you can verify that the
settings have been properly applied:

### 1. Find the Barbican Pod

First, identify the running Barbican pod:

```bash
$ oc get pods -l component=barbican-api
```

### 2. Verify Configuration Loading

Connect to the Barbican Pod and check that your custom configuration has been
loaded:

```bash
# Replace <barbican-pod-name> with the actual pod name from step 1
oc rsh -c barbican-api <barbican-pod-name>
# Inside the pod, dump the HTTPD configuration and check for your custom settings
httpd -D DUMP_CONFIG | grep -i sslprotocol
```

For the `httpd_custom_ssl.conf` example, you should see output similar to:

```
SSLProtocol -all +TLSv1.2 +TLSv1.3
```

### 3. Additional Verification Commands

You can also verify other aspects of the configuration:

```bash
# Check all loaded configuration files
$ httpd -D DUMP_INCLUDES
```
