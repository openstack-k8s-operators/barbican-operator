/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package functional

import (
	"fmt"

	maps "golang.org/x/exp/maps"

	. "github.com/onsi/gomega" //revive:disable:dot-imports

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barbicanv1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

func CreateBarbicanSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"AdminPassword":            []byte("12345678"),
			"BarbicanPassword":         []byte("12345678"),
			"KeystoneDatabasePassword": []byte("12345678"),
			"BarbicanSimpleCryptoKEK":  []byte("sEFmdFjDUqRM2VemYslV5yGNWjokioJXsg8Nrlc3drU="),
		},
	)
}

func GetDefaultBarbicanSpec() map[string]interface{} {
	return map[string]interface{}{
		"databaseInstance":          "openstack",
		"secret":                    SecretName,
		"simpleCryptoBackendSecret": SecretName,
	}
}

func CreateBarbican(name types.NamespacedName, spec map[string]interface{}) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "barbican.openstack.org/v1beta1",
		"kind":       "Barbican",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return th.CreateUnstructured(raw)
}

func GetBarbican(name types.NamespacedName) *barbicanv1.Barbican {
	instance := &barbicanv1.Barbican{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func CreateBarbicanMessageBusSecret(namespace string, name string) *corev1.Secret {
	s := th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"transport_url": []byte(fmt.Sprintf("rabbit://%s/fake", name)),
		},
	)
	logger.Info("Secret created", "name", name)
	return s
}

func BarbicanConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetBarbican(name)
	return instance.Status.Conditions
}

func BarbicanAPINotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &barbicanv1.BarbicanAPI{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func BarbicanWorkerNotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &barbicanv1.BarbicanWorker{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func BarbicanKeystoneListenerNotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &barbicanv1.BarbicanKeystoneListener{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func BarbicanExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &barbicanv1.Barbican{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeFalse())
	}, timeout, interval).Should(Succeed())
}

func BarbicanAPIConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetBarbicanAPI(name)
	return instance.Status.Conditions
}

func BarbicanAPIExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &barbicanv1.BarbicanAPI{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeFalse())
	}, timeout, interval).Should(Succeed())
}

// GetBarbicanAPI - Returns BarbicanAPI subCR
func GetBarbicanAPI(name types.NamespacedName) *barbicanv1.BarbicanAPI {
	instance := &barbicanv1.BarbicanAPI{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// GetBarbicanKeystoneListener - Returns BarbicanKeystoneListener subCR
func GetBarbicanKeystoneListener(name types.NamespacedName) *barbicanv1.BarbicanKeystoneListener {
	instance := &barbicanv1.BarbicanKeystoneListener{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// GetBarbicanWorker - Returns BarbicanWorker subCR
func GetBarbicanWorker(name types.NamespacedName) *barbicanv1.BarbicanWorker {
	instance := &barbicanv1.BarbicanWorker{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// ========== TLS Stuff ==============
func GetTLSBarbicanSpec() map[string]interface{} {
	return map[string]interface{}{
		"databaseInstance":          "openstack",
		"secret":                    SecretName,
		"simpleCryptoBackendSecret": SecretName,
		"barbicanAPI":               GetTLSBarbicanAPISpec(),
	}
}

func GetTLSBarbicanAPISpec() map[string]interface{} {
	spec := GetDefaultBarbicanAPISpec()
	maps.Copy(spec, map[string]interface{}{
		"tls": map[string]interface{}{
			"api": map[string]interface{}{
				"internal": map[string]interface{}{
					"secretName": InternalCertSecretName,
				},
				"public": map[string]interface{}{
					"secretName": PublicCertSecretName,
				},
			},
			"caBundleSecretName": CABundleSecretName,
		},
	})
	return spec
}

// ========== End of TLS Stuff ============

// ========== PKCS11 Stuff ============
const PKCS11CustomData = `
[p11_crypto_plugin]
plugin_name = PKCS11
library_path = some_library_path
token_labels = some_partition_label
mkek_label = my_mkek_label
hmac_label = my_hmac_label
encryption_mechanism = CKM_AES_GCM
aes_gcm_generate_iv = true
hmac_key_type = CKK_GENERIC_SECRET
hmac_keygen_mechanism = CKM_GENERIC_SECRET_KEY_GEN
hmac_mechanism = CKM_SHA256_HMAC
key_wrap_mechanism = CKM_AES_CBC_PAD
key_wrap_generate_iv = true
always_set_cka_sensitive = true
os_locking_ok = false`

func GetPKCS11BarbicanSpec() map[string]interface{} {
	spec := GetDefaultBarbicanSpec()
	maps.Copy(spec, map[string]interface{}{
		"customServiceConfig":      PKCS11CustomData,
		"enabledSecretStores":      []string{"pkcs11"},
		"globalDefaultSecretStore": "pkcs11",
		"pkcs11": map[string]interface{}{
			"clientDataPath":   PKCS11ClientDataPath,
			"loginSecret":      PKCS11LoginSecret,
			"clientDataSecret": PKCS11ClientDataSecret,
		},
	})
	return spec
}

func GetPKCS11BarbicanAPISpec() map[string]interface{} {
	spec := GetPKCS11BarbicanSpec()
	maps.Copy(spec, GetDefaultBarbicanAPISpec())
	return spec
}

func CreatePKCS11LoginSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"PKCS11Pin": []byte("12345678"),
		},
	)
}

func CreatePKCS11ClientDataSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"Client.cfg": []byte("dummy-data"),
			"CACert.pem": []byte("dummy-data"),
			"Server.pem": []byte("dummy-data"),
			"Client.pem": []byte("dummy-data"),
			"Client.key": []byte("dummy-data"),
		},
	)
}

// ========== End of PKCS11 Stuff ============

func GetDefaultBarbicanAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"secret":                    SecretName,
		"simpleCryptoBackendSecret": SecretName,
		"replicas":                  1,
		"databaseHostname":          barbicanTest.DatabaseHostname,
		"databaseInstance":          barbicanTest.DatabaseInstance,
		"containerImage":            barbicanTest.ContainerImage,
		"serviceAccount":            barbicanTest.BarbicanSA.Name,
		"transportURLSecret":        barbicanTest.RabbitmqSecretName,
	}
}

func CreateBarbicanAPI(name types.NamespacedName, spec map[string]interface{}) client.Object {
	// we get the parent CR and set ownership to the barbicanAPI CR
	raw := map[string]interface{}{
		"apiVersion": "barbican.openstack.org/v1beta1",
		"kind":       "BarbicanAPI",
		"metadata": map[string]interface{}{
			"annotations": map[string]interface{}{
				"keystoneEndpoint": "true",
			},
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}

	return th.CreateUnstructured(raw)
}

// GetSampleTopologySpec - A sample (and opinionated) Topology Spec used to
// test Service components
func GetSampleTopologySpec() map[string]interface{} {
	// Build the topology Spec
	topologySpec := map[string]interface{}{
		"topologySpreadConstraints": []map[string]interface{}{
			{
				"maxSkew":           1,
				"topologyKey":       corev1.LabelHostname,
				"whenUnsatisfiable": "ScheduleAnyway",
				"labelSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"service": barbicanName.Name,
					},
				},
			},
		},
	}
	return topologySpec
}

// CreateTopology - Creates a Topology CR based on the spec passed as input
func CreateTopology(topology types.NamespacedName, spec map[string]interface{}) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "topology.openstack.org/v1beta1",
		"kind":       "Topology",
		"metadata": map[string]interface{}{
			"name":      topology.Name,
			"namespace": topology.Namespace,
		},
		"spec": spec,
	}
	return th.CreateUnstructured(raw)
}

// GetBarbicanAPISpec -
func GetBarbicanAPISpec(name types.NamespacedName) barbicanv1.BarbicanAPITemplate {
	instance := &barbicanv1.BarbicanAPI{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.BarbicanAPITemplate
}

// GetBarbicanKeystoneListenerSpec -
func GetBarbicanKeystoneListenerSpec(name types.NamespacedName) barbicanv1.BarbicanKeystoneListenerTemplate {
	instance := &barbicanv1.BarbicanKeystoneListener{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.BarbicanKeystoneListenerTemplate
}

// GetBarbicanWorkerSpec -
func GetBarbicanWorkerSpec(name types.NamespacedName) barbicanv1.BarbicanWorkerTemplate {
	instance := &barbicanv1.BarbicanWorker{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.BarbicanWorkerTemplate
}
