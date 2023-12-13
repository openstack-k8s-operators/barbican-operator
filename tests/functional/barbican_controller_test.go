package functional_test

import (
	//"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	//rabbit //"k8s.io/utils/ptr"
)

var _ = Describe("Barbican controller", func() {

	var barbicanName types.NamespacedName
	var barbicanTransportURL types.NamespacedName
	var dbSyncJobName types.NamespacedName
	var barbicanConfigMapData types.NamespacedName
	//var bootstrapJobName types.NamespacedName
	//var deploymentName types.NamespacedName

	BeforeEach(func() {

		barbicanName = types.NamespacedName{
			Name:      "barbican",
			Namespace: namespace,
		}
		barbicanTransportURL = types.NamespacedName{
			Name:      "barbican-barbican-transport",
			Namespace: namespace,
		}
		dbSyncJobName = types.NamespacedName{
			Name:      "barbican-db-sync",
			Namespace: namespace,
		}
		barbicanConfigMapData = types.NamespacedName{
			Name:      "barbican-config-data",
			Namespace: namespace,
		}
		/*
			bootstrapJobName = types.NamespacedName{
				Name:      "keystone-bootstrap",
				Namespace: namespace,
			}
			deploymentName = types.NamespacedName{
				Name:      "keystone",
				Namespace: namespace,
			}
		*/

		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())

		/*
			DeferCleanup(
				k8sClient.Delete, ctx, th.CreateSecret(
					types.NamespacedName{Namespace: namespace, Name: SecretName},
					map[string][]byte{},
				))
		*/
	})

	When("A Barbican instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanName, GetDefaultBarbicanSpec()))
		})

		It("should have the Spec fields defaulted", func() {
			Barbican := GetBarbican(barbicanName)
			Expect(Barbican.Spec.ServiceUser).Should(Equal("barbican"))
			Expect(Barbican.Spec.DatabaseInstance).Should(Equal("openstack"))
			Expect(Barbican.Spec.DatabaseUser).Should(Equal("barbican"))
		})

		It("should have the Status fields initialized", func() {
			Barbican := GetBarbican(barbicanName)
			Expect(Barbican.Status.Hash).To(BeEmpty())
			Expect(Barbican.Status.BarbicanAPIReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.BarbicanWorkerReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.BarbicanKeystoneListenerReadyCount).To(Equal(int32(0)))
			Expect(Barbican.Status.TransportURLSecret).To(Equal(""))
			Expect(Barbican.Status.DatabaseHostname).To(Equal(""))
		})

		It("should have input not ready and unknown Conditions initialized", func() {
			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionUnknown,
			)

			for _, cond := range []condition.Type{
				condition.ServiceConfigReadyCondition,
				condition.DBReadyCondition,
				condition.DBSyncReadyCondition,
				//condition.BarbicanAPIReadyCondition,
				//condition.BarbicanWorkerReadyCondition,
				//condition.BarbicanKeystoneListenerReadyCondition,
				//condition.,
				//condition.ExposeServiceReadyCondition,
				//condition.BootstrapReadyCondition,
				//condition.DeploymentReadyCondition,
				condition.NetworkAttachmentsReadyCondition,
				//condition.CronJobReadyCondition,
			} {
				th.ExpectCondition(
					barbicanName,
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
				return GetBarbican(barbicanName).Finalizers
			}, timeout, interval).Should(ContainElement("Barbican"))
		})
		It("should not create a config map", func() {
			Eventually(func() []corev1.ConfigMap {
				return th.ListConfigMaps(barbicanConfigMapData.Name).Items
			}, timeout, interval).Should(BeEmpty())
		})
	})

	When("Barbican DB is created", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanName.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanName, GetDefaultBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateKeystoneAPISecret(namespace, SecretName))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					namespace,
					GetBarbican(barbicanName).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(barbicanTransportURL)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanName.Namespace))
			//DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, "memcached", memcachedSpec))
			/*
				infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
				DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(namespace))
			*/
		})
		It("Should set DBReady Condition and set DatabaseHostname Status when DB is Created", func() {
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanName)
			th.SimulateJobSuccess(dbSyncJobName)
			Barbican := GetBarbican(barbicanName)
			Expect(Barbican.Status.DatabaseHostname).To(Equal("hostname-for-openstack"))
			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.InputReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Should fail if db-sync job fails when DB is Created", func() {
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanName)
			th.SimulateJobFailure(dbSyncJobName)
			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanName,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Does not create BarbicanAPI", func() {
			BarbicanAPINotExists(barbicanName)
		})
		It("Does not create BarbicanWorker", func() {
			BarbicanWorkerNotExists(barbicanName)
		})
		It("Does not create BarbicanKeystoneListener", func() {
			BarbicanKeystoneListenerNotExists(barbicanName)
		})
	})
})
