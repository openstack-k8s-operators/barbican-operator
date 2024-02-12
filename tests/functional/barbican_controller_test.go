package functional

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Barbican controller", func() {
	When("A Barbican instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetDefaultBarbicanSpec()))
		})

		It("should have the Spec fields defaulted", func() {
			Barbican := GetBarbican(barbicanTest.Instance)
			Expect(Barbican.Spec.ServiceUser).Should(Equal("barbican"))
			Expect(Barbican.Spec.DatabaseInstance).Should(Equal("openstack"))
			Expect(Barbican.Spec.DatabaseUser).Should(Equal("barbican"))
		})

		It("should have the Status fields initialized", func() {
			Barbican := GetBarbican(barbicanTest.Instance)
			Expect(Barbican.Status.Hash).To(BeEmpty())
			Expect(Barbican.Status.BarbicanAPIReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.BarbicanWorkerReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.BarbicanKeystoneListenerReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.TransportURLSecret).To(Equal(""))
			Expect(Barbican.Status.DatabaseHostname).To(Equal(""))
		})

		It("should have input not ready and unknown Conditions initialized", func() {
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionUnknown,
			)

			for _, cond := range []condition.Type{
				condition.ServiceConfigReadyCondition,
				condition.DBReadyCondition,
				condition.DBSyncReadyCondition,
				condition.NetworkAttachmentsReadyCondition,
			} {
				th.ExpectCondition(
					barbicanTest.Instance,
					ConditionGetterFunc(BarbicanConditionGetter),
					cond,
					corev1.ConditionUnknown,
				)
			}
		})
		It("should have a finalizer", func() {
			// the reconciler loop adds the finalizer so we have to wait for
			// it to run
			Eventually(func() []string {
				return GetBarbican(barbicanTest.Instance).Finalizers
			}, timeout, interval).Should(ContainElement("Barbican"))
		})
		It("should not create a config map", func() {
			Eventually(func() []corev1.ConfigMap {
				return th.ListConfigMaps(barbicanTest.BarbicanConfigMapData.Name).Items
			}, timeout, interval).Should(BeEmpty())
		})
	})

	When("Barbican DB is created", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetDefaultBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateKeystoneAPISecret(barbicanTest.Instance.Namespace, SecretName))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					barbicanTest.Instance.Namespace,
					GetBarbican(barbicanTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(barbicanTest.BarbicanTransportURL)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace))
		})
		It("Should set DBReady Condition and set DatabaseHostname Status when DB is Created", func() {
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.Instance)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.Instance)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
			Barbican := GetBarbican(barbicanTest.Instance)
			Expect(Barbican.Status.DatabaseHostname).To(Equal(fmt.Sprintf("hostname-for-openstack.%s.svc", namespace)))
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Should fail if db-sync job fails when DB is Created", func() {
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.Instance)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.Instance)
			th.SimulateJobFailure(barbicanTest.BarbicanDBSync)
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Does not create BarbicanAPI", func() {
			BarbicanAPINotExists(barbicanTest.Instance)
		})
		It("Does not create BarbicanWorker", func() {
			BarbicanWorkerNotExists(barbicanTest.Instance)
		})
		It("Does not create BarbicanKeystoneListener", func() {
			BarbicanKeystoneListenerNotExists(barbicanTest.Instance)
		})
	})

	When("DB sync is completed", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetDefaultBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateKeystoneAPISecret(barbicanTest.Instance.Namespace, SecretName))

			DeferCleanup(
				k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, "test-osp-secret-barbican"))

			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					barbicanTest.Instance.Namespace,
					GetBarbican(barbicanTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(barbicanTest.BarbicanTransportURL)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace))
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.Instance)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.Instance)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
		})

		It("should have db sync ready condition", func() {
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition,
				corev1.ConditionTrue,
			)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})
	When("A Barbican with TLS is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetTLSBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetTLSBarbicanAPISpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateKeystoneAPISecret(barbicanTest.Instance.Namespace, SecretName))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					barbicanTest.Instance.Namespace,
					GetBarbican(barbicanTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(barbicanTest.BarbicanTransportURL)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace))
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.Instance)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.Instance)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
		})

		It("Creates BarbicanAPI", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			BarbicanAPIExists(barbicanTest.Instance)

			d := th.GetDeployment(barbicanTest.BarbicanAPI)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))

			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(6))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			// cert deployment volumes
			th.AssertVolumeExists(barbicanTest.CABundleSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(barbicanTest.InternalCertSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(barbicanTest.PublicCertSecret.Name, d.Spec.Template.Spec.Volumes)

			// cert volumeMounts
			container := d.Spec.Template.Spec.Containers[1]
			th.AssertVolumeMountExists(barbicanTest.InternalCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.InternalCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.PublicCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.PublicCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.CABundleSecret.Name, "tls-ca-bundle.pem", container.VolumeMounts)

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
		})
	})
})
