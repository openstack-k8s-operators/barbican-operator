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

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	barbicanv1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

func CreateKeystoneAPISecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"AdminPassword":            []byte("12345678"),
			"KeystoneDatabasePassword": []byte("12345678"),
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

func CreateBarbicanSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"BarbicanDatabasePassword": []byte("12345678"),
			"BarbicanPassword":         []byte("12345678"),
			"BarbicanSimpleCryptoKEK":  []byte("sEFmdFjDUqRM2VemYslV5yGNWjokioJXsg8Nrlc3drU="),
		},
	)
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

// ========== TLS Stuff ==============
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

func GetBarbicanAPI(name types.NamespacedName) *barbicanv1.BarbicanAPI {
	instance := &barbicanv1.BarbicanAPI{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

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
