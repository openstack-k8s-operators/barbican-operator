/*
Copyright 2025.

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
	"errors"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Barbican webhook", func() {
	It("rejects update to deprecated rabbitMqClusterName field", func() {
		spec := GetDefaultBarbicanSpec()
		spec["rabbitMqClusterName"] = "rabbitmq"

		barbicanName := types.NamespacedName{
			Namespace: namespace,
			Name:      "barbican-webhook-test",
		}

		raw := map[string]any{
			"apiVersion": "barbican.openstack.org/v1beta1",
			"kind":       "Barbican",
			"metadata": map[string]any{
				"name":      barbicanName.Name,
				"namespace": barbicanName.Namespace,
			},
			"spec": spec,
		}

		// Create the Barbican instance
		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			ctx, k8sClient, unstructuredObj, func() error { return nil })
		Expect(err).ShouldNot(HaveOccurred())

		DeferCleanup(func() {
			_ = k8sClient.Delete(ctx, unstructuredObj)
		})

		// Try to update rabbitMqClusterName
		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, barbicanName, unstructuredObj)).Should(Succeed())
			specMap := unstructuredObj.Object["spec"].(map[string]any)
			specMap["rabbitMqClusterName"] = "rabbitmq2"
			err := k8sClient.Update(ctx, unstructuredObj)
			g.Expect(err).Should(HaveOccurred())

			var statusError *k8s_errors.StatusError
			g.Expect(errors.As(err, &statusError)).To(BeTrue())
			g.Expect(statusError.ErrStatus.Details.Kind).To(Equal("Barbican"))
			g.Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("field \"spec.rabbitMqClusterName\" is deprecated"))
			g.Expect(statusError.ErrStatus.Message).To(
				ContainSubstring("use \"spec.messagingBus.cluster\" instead"))
		}, timeout, interval).Should(Succeed())
	})

	It("rejects a kmip secret store without a kmip spec", func() {
		spec := GetDefaultBarbicanSpec()
		spec["enabledSecretStores"] = []string{"kmip"}

		_, err := createBarbicanForWebhook(spec, "barbican-kmip-missing")
		Expect(err).Should(HaveOccurred())

		var statusError *k8s_errors.StatusError
		Expect(errors.As(err, &statusError)).To(BeTrue())
		Expect(statusError.ErrStatus.Details.Kind).To(Equal("Barbican"))
		Expect(statusError.ErrStatus.Message).To(
			ContainSubstring("KMIP is required when kmip is an enabled SecretStore"))
	})

	It("rejects a kmip spec without a clientDataSecret", func() {
		spec := GetDefaultBarbicanSpec()
		spec["enabledSecretStores"] = []string{"kmip"}
		spec["kmip"] = map[string]any{
			"clientDataPath": KMIPClientDataPath,
		}

		_, err := createBarbicanForWebhook(spec, "barbican-kmip-no-secret")
		Expect(err).Should(HaveOccurred())

		var statusError *k8s_errors.StatusError
		Expect(errors.As(err, &statusError)).To(BeTrue())
		Expect(statusError.ErrStatus.Message).To(
			ContainSubstring("clientDataSecret is required when kmip is an enabled SecretStore"))
	})

	It("rejects pkcs11 and kmip sharing a clientDataPath", func() {
		spec := GetDefaultBarbicanSpec()
		spec["enabledSecretStores"] = []string{"pkcs11", "kmip"}
		spec["pkcs11"] = map[string]any{
			"loginSecret":      PKCS11LoginSecret,
			"clientDataSecret": PKCS11ClientDataSecret,
			"clientDataPath":   "/etc/shared-client",
		}
		spec["kmip"] = map[string]any{
			"clientDataSecret": KMIPClientDataSecret,
			"clientDataPath":   "/etc/shared-client",
		}

		_, err := createBarbicanForWebhook(spec, "barbican-clientdata-clash")
		Expect(err).Should(HaveOccurred())

		var statusError *k8s_errors.StatusError
		Expect(errors.As(err, &statusError)).To(BeTrue())
		Expect(statusError.ErrStatus.Message).To(
			ContainSubstring("clientDataPath must differ from spec.pkcs11.clientDataPath"))
	})

	It("defaults the per secret store clientDataPath", func() {
		spec := GetDefaultBarbicanSpec()
		spec["enabledSecretStores"] = []string{"pkcs11", "kmip"}
		spec["pkcs11"] = map[string]any{
			"loginSecret":      PKCS11LoginSecret,
			"clientDataSecret": PKCS11ClientDataSecret,
		}
		spec["kmip"] = map[string]any{
			"clientDataSecret": KMIPClientDataSecret,
		}

		barbicanName, err := createBarbicanForWebhook(spec, "barbican-clientdata-defaults")
		Expect(err).ShouldNot(HaveOccurred())

		// Each store falls back to its own default path, so the two mounts
		// never land on the same point.
		Barbican := GetBarbican(barbicanName)
		Expect(Barbican.Spec.PKCS11.ClientDataPath).To(Equal(barbicanv1beta1.DefaultPKCS11ClientDataPath))
		Expect(Barbican.Spec.KMIP.ClientDataPath).To(Equal(barbicanv1beta1.DefaultKMIPClientDataPath))
	})
})

// createBarbicanForWebhook creates a Barbican from a raw spec so that the
// admission webhooks run against it, and registers it for cleanup.
func createBarbicanForWebhook(spec map[string]any, name string) (types.NamespacedName, error) {
	barbicanName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	unstructuredObj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "barbican.openstack.org/v1beta1",
		"kind":       "Barbican",
		"metadata": map[string]any{
			"name":      barbicanName.Name,
			"namespace": barbicanName.Namespace,
		},
		"spec": spec,
	}}

	_, err := controllerutil.CreateOrPatch(
		ctx, k8sClient, unstructuredObj, func() error { return nil })

	DeferCleanup(func() {
		_ = k8sClient.Delete(ctx, unstructuredObj)
	})

	return barbicanName, err
}
