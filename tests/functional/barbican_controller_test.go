package functional

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//revive:disable-next-line:dot-imports
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	controllers "github.com/openstack-k8s-operators/barbican-operator/controllers"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	mariadb_test "github.com/openstack-k8s-operators/mariadb-operator/api/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
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
			Expect(Barbican.Spec.DatabaseAccount).Should(Equal("barbican"))
			Expect(Barbican.Spec.CustomServiceConfig).Should(Equal(barbicanTest.BaseCustomServiceConfig))
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
			}, timeout, interval).Should(ContainElement("openstack.org/barbican"))
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
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
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
		It("should create config-data and scripts ConfigMaps", func() {
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			conf := cf.Data["my.cnf"]
			Expect(conf).To(
				ContainSubstring("[client]\nssl=0"))
		})
		It("Should fail if db-sync job fails when DB is Created", func() {
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
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
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))

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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
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

		It("checks the 10-barbican_wsgi_main.conf contains the correct TimeOut", func() {
			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			httpdConfData := string(cf.Data["10-barbican_wsgi_main.conf"])
			Expect(httpdConfData).To(
				ContainSubstring("TimeOut 90"),
			)
		})
		It("checks the relevant secrets contain the customServiceConfig", func() {
			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData := string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))

			cf = th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData = string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))

			cf = th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData = string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))

			cf = th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData = string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))
		})
	})
	When("A Barbican with TLS is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetTLSBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetTLSBarbicanAPISpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBTLSDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
			DeferCleanup(k8sClient.Delete, ctx, CreateCustomConfigSecret(
				barbicanTest.Instance.Namespace,
				APICustomConfigSecret1Name,
				barbicanTest.APICustomConfigSecret1Contents),
			)
			DeferCleanup(k8sClient.Delete, ctx, CreateCustomConfigSecret(
				barbicanTest.Instance.Namespace,
				APICustomConfigSecret2Name,
				barbicanTest.APICustomConfigSecret2Contents),
			)
		})

		It("Creates BarbicanAPI", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionTrue,
			)

			BarbicanAPIExists(barbicanTest.Instance)

			d := th.GetDeployment(barbicanTest.BarbicanAPIDeployment)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))

			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(6))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			// Check the default volumes
			th.AssertVolumeExists("config-data", d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists("config-data-custom", d.Spec.Template.Spec.Volumes)

			// cert deployment volumes
			th.AssertVolumeExists(barbicanTest.CABundleSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(barbicanTest.InternalCertSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(barbicanTest.PublicCertSecret.Name, d.Spec.Template.Spec.Volumes)

			// default and cert volumeMounts
			container := d.Spec.Template.Spec.Containers[1]
			th.AssertVolumeMountExists("config-data", "", container.VolumeMounts)
			th.AssertVolumeMountExists("config-data-custom", "", container.VolumeMounts)

			th.AssertVolumeMountExists(barbicanTest.InternalCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.InternalCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.PublicCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.PublicCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(barbicanTest.CABundleSecret.Name, "tls-ca-bundle.pem", container.VolumeMounts)

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
		})

		It("should create config-data and scripts ConfigMaps", func() {
			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			conf := cf.Data["my.cnf"]
			Expect(conf).To(
				ContainSubstring("[client]\nssl-ca=/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem\nssl=1"))
		})

		It("checks the relevant secrets contain the base and API customServiceConfig", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData := string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))

			cf = th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData = string(cf.Data["01-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.BaseCustomServiceConfig))

			customData = string(cf.Data["02-service-custom.conf"])
			Expect(customData).To(Equal(barbicanTest.APICustomServiceConfig))
		})

		It("checks the relevant secrets contain the base and API defaultConfigOverwrite", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			for fname, val := range barbicanTest.BaseDefaultConfigOverwrite {
				customData := string(cf.Data[fname])
				Expect(customData).To(Equal(val))
			}

			cf = th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			for fname, val := range barbicanTest.APIDefaultConfigOverwrite {
				// all the API custom values should be there
				customData := string(cf.Data[fname])
				Expect(customData).To(Equal(val))
			}
			for fname, val := range barbicanTest.BaseDefaultConfigOverwrite {
				_, ok := barbicanTest.APIDefaultConfigOverwrite[fname]
				if ok {
					// we've already checked this value
					continue
				}
				customData := string(cf.Data[fname])
				Expect(customData).To(Equal(val))
			}

		})
		It("checks the relevant secrets contain the API CustomServiceConfigSecrets", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			cf := th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			customData := string(cf.Data["03-secrets-custom.conf"])

			secret1 := th.GetSecret(barbicanTest.APICustomConfigSecret1)
			Expect(secret1).ShouldNot(BeNil())

			secret2 := th.GetSecret(barbicanTest.APICustomConfigSecret2)
			Expect(secret1).ShouldNot(BeNil())

			Expect(customData).To(Equal(string(secret1.Data["secret1"]) + "\n" + string(secret2.Data["secret2"]) + "\n"))
		})
	})

	When("Barbican is created with topologyRef", func() {
		var topologyRef, topologyRefAlt *topologyv1.TopoRef
		BeforeEach(func() {
			// Create Test Topologies based on the entries defined in the
			// BarbicanTopologies array. The spec is generated using GetSampleTopologySpec
			// function, and t.Name is used as label to customize the default
			// selector. By doing this we are able to have well defined and distinct
			// topologies that are applied to BarbicanAPI, BarbicanKeystoneListener
			// and BarbicanWorker.
			for _, t := range barbicanTest.BarbicanTopologies {
				// Build the topology Spec
				topologySpec, _ := GetSampleTopologySpec(t.Name)
				infra.CreateTopology(t, topologySpec)
			}

			// Define the two topology references used in this test: topologyRef
			// represents the global-topology, while topologyRefAlt represents
			// an alternative topologyRef that is used by the "update" envTest
			// to override the global topology referenced here.
			topologyRef = &topologyv1.TopoRef{
				Name:      barbicanTest.BarbicanTopologies[0].Name,
				Namespace: barbicanTest.BarbicanTopologies[0].Namespace,
			}
			topologyRefAlt = &topologyv1.TopoRef{
				Name:      barbicanTest.BarbicanTopologies[1].Name,
				Namespace: barbicanTest.BarbicanTopologies[1].Namespace,
			}
			spec := GetDefaultBarbicanSpec()
			// Update the spec and add a top-level topologyRef that points to
			// gloabal-topology
			spec["topologyRef"] = map[string]interface{}{
				"name": topologyRef.Name,
			}
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, spec))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))

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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
		})
		It("sets topology in CR status", func() {
			// expectedTopology points to topologyRef, the global-topology
			// defined in BeforeEach
			expectedTopology := &topologyv1.TopoRef{
				Name:      topologyRef.Name,
				Namespace: topologyRef.Namespace,
			}
			Eventually(func(g Gomega) {
				// retrieve the created topology in the current namespace
				tp := infra.GetTopology(types.NamespacedName{
					Name:      expectedTopology.Name,
					Namespace: expectedTopology.Namespace,
				})
				// We expect that finalizer has length eq to 3, because the same
				// global-topology is applied at top-level and inherited by all
				// the barbican subCRs
				g.Expect(tp.GetFinalizers()).To(HaveLen(3))
				finalizers := tp.GetFinalizers()
				barbicanAPI := GetBarbicanAPI(barbicanTest.BarbicanAPI)
				g.Expect(barbicanAPI.Status.LastAppliedTopology).ToNot(BeNil())
				// verify the .Status.LastAppliedTopology is updated and points
				// to the referenced topologyRef
				g.Expect(barbicanAPI.Status.LastAppliedTopology).To(Equal(topologyRef))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanapi-%s", barbicanAPI.Name)))

				barbicanWorker := GetBarbicanWorker(barbicanTest.BarbicanWorker)
				g.Expect(barbicanWorker.Status.LastAppliedTopology).ToNot(BeNil())
				// verify the .Status.LastAppliedTopology is updated and points
				// to the referenced topologyRef
				g.Expect(barbicanWorker.Status.LastAppliedTopology).To(Equal(topologyRef))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanworker-%s", barbicanWorker.Name)))

				barbicanKeystoneListener := GetBarbicanKeystoneListener(barbicanTest.BarbicanKeystoneListener)
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).ToNot(BeNil())
				// verify the .Status.LastAppliedTopology is updated and points
				// to the referenced topologyRef
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).To(Equal(topologyRef))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicankeystonelistener-%s", barbicanKeystoneListener.Name)))
			}, timeout, interval).Should(Succeed())
		})

		It("sets topology in the resulting deployments", func() {
			Eventually(func(g Gomega) {
				// retrieve the topologySpec associated to the currently referenced
				// topology
				_, expectedTopologySpecObj := GetSampleTopologySpec(topologyRef.Name)
				// verify that TopologySpreadConstraints is not nil (it proves that the Topology has been injected in the PodSpec)
				// and that is matches with what has been referenced at top-level and inherited by the current service. The same
				// comment applies to API, Worker and KeystoneListener.
				// NOTE: this tests also the logic defined in the deployment definition, where Affinity is not set by default when
				// a Topology is referenced. By checking that .Spec.Affinity is nil, we prove that we get the expected result
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.Affinity).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(Equal(expectedTopologySpecObj))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanWorkerDeployment).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanWorkerDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(Equal(expectedTopologySpecObj))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanWorkerDeployment).Spec.Template.Spec.Affinity).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanKeystoneListenerDeployment).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanKeystoneListenerDeployment).Spec.Template.Spec.Affinity).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanKeystoneListenerDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(Equal(expectedTopologySpecObj))
			}, timeout, interval).Should(Succeed())
		})
		It("updates topology when the reference changes", func() {
			// expectedTopology points to topologyRefAlt, the alternative topology
			// that is defined in BeforeEach and is suppposed to override the existing
			// referenced topology. This test ensures that we are able to properly
			// perform day2 operations and manage the Topology lifecycle.
			expectedTopology := &topologyv1.TopoRef{
				Name:      topologyRefAlt.Name,
				Namespace: topologyRefAlt.Namespace,
			}
			var finalizers []string
			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanTest.Instance)
				// Update the top-level TopologyRef and point to the alternative
				// Topology that will be propagated to the underlying components
				barbican.Spec.TopologyRef.Name = topologyRefAlt.Name
				g.Expect(k8sClient.Update(ctx, barbican)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				tp := infra.GetTopology(types.NamespacedName{
					Name:      expectedTopology.Name,
					Namespace: expectedTopology.Namespace,
				})
				finalizers = tp.GetFinalizers()
				// We expect that finalizer has length eq to 3, because the same
				// alternative topology is applied at top-level and inherited by
				// all the barbican subCRs
				g.Expect(finalizers).To(HaveLen(3))
				keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)
				barbicanAPI := GetBarbicanAPI(barbicanTest.BarbicanAPI)
				g.Expect(barbicanAPI.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanAPI.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied to the alternative topology
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanapi-%s", barbicanAPI.Name)))
				barbicanWorker := GetBarbicanWorker(barbicanTest.BarbicanWorker)
				g.Expect(barbicanWorker.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanWorker.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied to the alternative topology
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanworker-%s", barbicanWorker.Name)))
				barbicanKeystoneListener := GetBarbicanKeystoneListener(barbicanTest.BarbicanKeystoneListener)
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied to the alternative topology
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicankeystonelistener-%s", barbicanKeystoneListener.Name)))
			}, timeout, interval).Should(Succeed())
		})

		It("verifies the previous topology reference has no finalizers anymore", func() {
			Eventually(func(g Gomega) {
				// Get the previous (global) topology and verify that the finalizers
				// have been removed
				tp := infra.GetTopology(types.NamespacedName{
					Name:      topologyRef.Name,
					Namespace: topologyRef.Namespace,
				})
				g.Expect(tp.GetFinalizers()).To(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("overrides topology when the reference changes", func() {
			// Each sub-component defines its own topologyRef: with this test
			// we prove that they do not inherit the top-level topology anymore,
			// but they honor the reference that has been passed
			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanTest.Instance)
				//Patch BarbicanAPI Spec
				newAPI := GetBarbicanAPISpec(barbicanTest.BarbicanAPI)
				// update barbicanAPI.spec.topologyRef to point to api-topology
				newAPI.TopologyRef.Name = barbicanTest.BarbicanTopologies[1].Name
				barbican.Spec.BarbicanAPI = newAPI
				//Patch BarbicanKeystoneListener Spec
				newKl := GetBarbicanKeystoneListenerSpec(barbicanTest.BarbicanKeystoneListener)
				// update barbicanKeystoneListener.spec.topologyRef to point to
				// klistener-topology
				newKl.TopologyRef.Name = barbicanTest.BarbicanTopologies[2].Name
				barbican.Spec.BarbicanKeystoneListener = newKl
				//Patch BarbicanWorker Spec
				newWorker := GetBarbicanWorkerSpec(barbicanTest.BarbicanWorker)
				// update worker.spec.TopologyRef to point to worker-topology
				newWorker.TopologyRef.Name = barbicanTest.BarbicanTopologies[3].Name
				barbican.Spec.BarbicanWorker = newWorker
				g.Expect(k8sClient.Update(ctx, barbican)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// For each subcomponent, as we did in the previous test, verify
			// that the topologyRef passed at the sub-level has been applied,
			// no inheritance happened
			Eventually(func(g Gomega) {
				expectedTopology := &topologyv1.TopoRef{
					Name:      barbicanTest.BarbicanTopologies[1].Name,
					Namespace: barbicanTest.BarbicanTopologies[1].Namespace,
				}
				// retrieve the referenced topology using the k8s client
				tp := infra.GetTopology(types.NamespacedName{
					Name:      expectedTopology.Name,
					Namespace: expectedTopology.Namespace,
				})
				finalizers := tp.GetFinalizers()
				// api-topology has BarbicanAPI finalizer only, hence we expect
				// length to be 1
				g.Expect(finalizers).To(HaveLen(1))
				barbicanAPI := GetBarbicanAPI(barbicanTest.BarbicanAPI)
				g.Expect(barbicanAPI.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanAPI.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanapi-%s", barbicanAPI.Name)))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				expectedTopology := &topologyv1.TopoRef{
					Name:      barbicanTest.BarbicanTopologies[2].Name,
					Namespace: barbicanTest.BarbicanTopologies[2].Namespace,
				}
				tp := infra.GetTopology(types.NamespacedName{
					Name:      expectedTopology.Name,
					Namespace: expectedTopology.Namespace,
				})
				finalizers := tp.GetFinalizers()
				// klistener-topology has BarbicanKeystoneListener finalizer
				// only, hence we expect length to be 1
				g.Expect(finalizers).To(HaveLen(1))
				barbicanKeystoneListener := GetBarbicanKeystoneListener(barbicanTest.BarbicanKeystoneListener)
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicankeystonelistener-%s", barbicanKeystoneListener.Name)))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				expectedTopology := &topologyv1.TopoRef{
					Name:      barbicanTest.BarbicanTopologies[3].Name,
					Namespace: barbicanTest.BarbicanTopologies[3].Namespace,
				}
				tp := infra.GetTopology(types.NamespacedName{
					Name:      expectedTopology.Name,
					Namespace: expectedTopology.Namespace,
				})
				finalizers := tp.GetFinalizers()
				// worker-topology has BarbicanWorker finalizer
				// only, hence we expect length to be 1
				g.Expect(finalizers).To(HaveLen(1))
				barbicanWorker := GetBarbicanWorker(barbicanTest.BarbicanWorker)
				g.Expect(barbicanWorker.Status.LastAppliedTopology).ToNot(BeNil())
				g.Expect(barbicanWorker.Status.LastAppliedTopology).To(Equal(expectedTopology))
				// verify the finalizer has been applied
				g.Expect(finalizers).To(ContainElement(
					fmt.Sprintf("openstack.org/barbicanworker-%s", barbicanWorker.Name)))
			}, timeout, interval).Should(Succeed())
		})
		It("removes topologyRef from the spec", func() {
			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanTest.Instance)
				// Remove the TopologyRef from the existing Barbican .Spec
				barbican.Spec.TopologyRef = nil
				g.Expect(k8sClient.Update(ctx, barbican)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				// When TopologyRef is removed, LastAppliedTopology topology
				// status field is nil, and not present anymore in the yaml
				// definition
				barbicanAPI := GetBarbicanAPI(barbicanTest.BarbicanAPI)
				g.Expect(barbicanAPI.Status.LastAppliedTopology).Should(BeNil())
				barbicanWorker := GetBarbicanWorker(barbicanTest.BarbicanWorker)
				g.Expect(barbicanWorker.Status.LastAppliedTopology).Should(BeNil())
				barbicanKeystoneListener := GetBarbicanKeystoneListener(barbicanTest.BarbicanKeystoneListener)
				g.Expect(barbicanKeystoneListener.Status.LastAppliedTopology).Should(BeNil())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				// When TopologyRef is removed, LastAppliedTopology topology
				// status field is nil, and the resulting deployment has no
				// TopologySpreadConstraints. In this case the default AntiAffinity
				// using the function provided by lib-common is applied
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.Affinity).ToNot(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanWorkerDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanWorkerDeployment).Spec.Template.Spec.Affinity).ToNot(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanKeystoneListenerDeployment).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanKeystoneListenerDeployment).Spec.Template.Spec.Affinity).ToNot(BeNil())
			}, timeout, interval).Should(Succeed())

			// Verify that all the finalizers in all the existing Topologies
			// have been removed
			Eventually(func(g Gomega) {
				for _, topology := range barbicanTest.BarbicanTopologies {
					// Get the current topology and verify there are no finalizers
					tp := infra.GetTopology(types.NamespacedName{
						Name:      topology.Name,
						Namespace: topology.Namespace,
					})
					g.Expect(tp.GetFinalizers()).To(BeEmpty())
				}
			}, timeout, interval).Should(Succeed())
		})
	})

	When("A Barbican with nodeSelector is created", func() {
		BeforeEach(func() {
			spec := GetDefaultBarbicanSpec()
			spec["nodeSelector"] = map[string]interface{}{
				"foo": "bar",
			}
			spec["barbicanAPI"] = GetDefaultBarbicanAPISpec()
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, spec))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))

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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
		})

		It("sets nodeSelector in resource specs", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())
		})

		It("updates nodeSelector in resource specs when changed", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				newNodeSelector := map[string]string{
					"foo2": "bar2",
				}
				barbican.Spec.NodeSelector = &newNodeSelector
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when cleared", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				emptyNodeSelector := map[string]string{}
				barbican.Spec.NodeSelector = &emptyNodeSelector
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when nilled", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				barbican.Spec.NodeSelector = nil
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				apiNodeSelector := map[string]string{
					"foo": "api",
				}
				barbican.Spec.BarbicanAPI.NodeSelector = &apiNodeSelector
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "api"}))
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override to empty", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				emptyNodeSelector := map[string]string{}
				barbican.Spec.BarbicanAPI.NodeSelector = &emptyNodeSelector
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPIDeployment).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})

	When("A Barbican with PKCS11 plugin is created - Luna model", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreatePKCS11LoginSecret(barbicanTest.Instance.Namespace, PKCS11LoginSecret))
			DeferCleanup(k8sClient.Delete, ctx, CreatePKCS11ClientDataSecret(barbicanTest.Instance.Namespace, PKCS11ClientDataSecret, "luna"))

			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetPKCS11BarbicanSpec("luna")))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			infra.SimulateTransportURLReady(barbicanTest.BarbicanTransportURL)
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace))
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetPKCS11BarbicanAPISpec("luna")))
			th.SimulateJobSuccess(barbicanTest.BarbicanPKCS11Prep)
		})

		It("Creates BarbicanAPI", func() {
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionTrue,
			)

			BarbicanAPIExists(barbicanTest.Instance)

			d := th.GetDeployment(barbicanTest.BarbicanAPIDeployment)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))

			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(4))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			container := d.Spec.Template.Spec.Containers[1]

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))

			// Checking the PKCS11 Client Data container
			Expect(container.Name).To(Equal(barbican.ComponentAPI))
			foundMount := false
			indexMount := 0
			for index, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == barbican.PKCS11ClientDataVolume {
					foundMount = true
					indexMount = index
					break
				}
			}
			Expect(foundMount).To(BeTrue())
			Expect(container.VolumeMounts[indexMount].MountPath).To(Equal(barbican.PKCS11ClientDataMountPoint))
		})

		It("Verifies the Barbican PKCS11 struct is in good shape", func() {
			Barbican := GetBarbican(barbicanTest.Instance)
			Expect(Barbican.Spec.EnabledSecretStores).Should(Equal([]barbicanv1beta1.SecretStore{"pkcs11"}))
			Expect(Barbican.Spec.GlobalDefaultSecretStore).Should(Equal(barbicanv1beta1.SecretStore("pkcs11")))

			pkcs11 := Barbican.Spec.PKCS11
			Expect(pkcs11.LoginSecret).Should(Equal(PKCS11LoginSecret))
			Expect(pkcs11.ClientDataSecret).Should(Equal(PKCS11ClientDataSecret))
			Expect(pkcs11.ClientDataPath).Should(Equal(PKCS11ClientDataPath["luna"]))
		})

		It("Verifies the BarbicanAPI PKCS11 struct is in good shape", func() {
			BarbicanAPI := GetBarbicanAPI(barbicanTest.Instance)
			Expect(BarbicanAPI.Spec.EnabledSecretStores).Should(Equal([]barbicanv1beta1.SecretStore{"pkcs11"}))
			Expect(BarbicanAPI.Spec.GlobalDefaultSecretStore).Should(Equal(barbicanv1beta1.SecretStore("pkcs11")))

			pkcs11 := BarbicanAPI.Spec.PKCS11
			Expect(pkcs11.LoginSecret).Should(Equal(PKCS11LoginSecret))
			Expect(pkcs11.ClientDataSecret).Should(Equal(PKCS11ClientDataSecret))
			Expect(pkcs11.ClientDataPath).Should(Equal(PKCS11ClientDataPath["luna"]))
		})

		It("Checks if the two relevant secrets have the right contents", func() {
			// TODO(alee) Eliminate this test?  Not sure if it tests anything other than setup
			hsmSecret := th.GetSecret(barbicanTest.BarbicanPKCS11LoginSecret)
			Expect(hsmSecret).ShouldNot(BeNil())
			confPKCS11 := hsmSecret.Data["PKCS11Pin"]
			Expect(confPKCS11).To(
				ContainSubstring("12345678"))

			clientDataSecret := th.GetSecret(barbicanTest.BarbicanPKCS11ClientDataSecret)
			Expect(clientDataSecret).ShouldNot(BeNil())
			confClient := clientDataSecret.Data["Client.cfg"]
			Expect(confClient).To(
				ContainSubstring("dummy-data"))
			confCA := clientDataSecret.Data["CACert.pem"]
			Expect(confCA).To(
				ContainSubstring("dummy-data"))
			confServer := clientDataSecret.Data["Server.pem"]
			Expect(confServer).To(
				ContainSubstring("dummy-data"))

			confClient = clientDataSecret.Data["Client.pem"]
			Expect(confClient).To(
				ContainSubstring("dummy-data"))
			confKey := clientDataSecret.Data["Client.key"]
			Expect(confKey).To(
				ContainSubstring("dummy-data"))
		})

		It("Verifies if 00-default.conf, barbican-api-config.json and 01-custom.conf have the right contents for Barbican.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["00-default.conf"]
			Expect(conf).To(
				ContainSubstring("stores_lookup_suffix = pkcs11"))
			Expect(conf).To(
				ContainSubstring("[secretstore:pkcs11]\nsecret_store_plugin = store_crypto\ncrypto_plugin = p11_crypto\nglobal_default = true"))
			Expect(conf).To(
				ContainSubstring("[p11_crypto_plugin]\nlogin = 12345678"))

			conf = confSecret.Data["01-custom.conf"]
			Expect(conf).To(
				ContainSubstring(PKCS11CustomData))

			conf = confSecret.Data["barbican-api-config.json"]
			Expect(conf).To(
				ContainSubstring("\"source\": \"/var/lib/config-data/hsm\""))
			Expect(conf).To(
				ContainSubstring("\"dest\": \"/usr/local/luna\""))
		})

		It("Verifies if 00-default.conf and 01-custom.conf have the right contents for BarbicanAPI.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["00-default.conf"]
			Expect(conf).To(
				ContainSubstring("stores_lookup_suffix = pkcs11"))
			Expect(conf).To(
				ContainSubstring("[secretstore:pkcs11]\nsecret_store_plugin = store_crypto\ncrypto_plugin = p11_crypto\nglobal_default = true"))
			Expect(conf).To(
				ContainSubstring("[p11_crypto_plugin]\nlogin = 12345678"))

			conf = confSecret.Data["01-custom.conf"]
			Expect(conf).To(
				ContainSubstring(PKCS11CustomData))
		})

		It("Verifies if barbican-api-config.json has the right contents for BarbicanAPI.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["barbican-api-config.json"]
			Expect(conf).To(
				ContainSubstring("\"source\": \"/var/lib/config-data/hsm\""))
			Expect(conf).To(
				ContainSubstring("\"dest\": \"/usr/local/luna\""))
		})

		It("Checks if the PKCS11PreJob successfully executed", func() {
			BarbicanExists(barbicanTest.Instance)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				controllers.PKCS11PrepReadyCondition,
				corev1.ConditionTrue,
			)

			// Checking if the volume mount name and mount path match the specified values.
			var elemClient, elemScript, elemConfig = 0, 0, 0
			for index, mount := range th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts {
				if mount.Name == barbican.PKCS11ClientDataVolume {
					elemClient = index
				} else if mount.Name == barbican.ScriptVolume {
					elemScript = index
				} else if mount.Name == barbican.ConfigVolume && mount.SubPath == "" {
					elemConfig = index
				}
			}

			volume := th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemClient].Name
			mountPath := th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemClient].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.PKCS11ClientDataVolume))
				g.Expect(mountPath).To(Equal(barbican.PKCS11ClientDataMountPoint))
			}, timeout, interval).Should(Succeed())

			volume = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemScript].Name
			mountPath = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemScript].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.ScriptVolume))
				g.Expect(mountPath).To(Equal(barbican.ScriptMountPoint))
			}, timeout, interval).Should(Succeed())

			volume = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemConfig].Name
			mountPath = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemConfig].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.ConfigVolume))
				g.Expect(mountPath).To(Equal(barbican.ConfigMountPoint))
			}, timeout, interval).Should(Succeed())
		})
	})

	// Proteccio
	When("A Barbican with PKCS11 plugin is created - Proteccio model", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreatePKCS11LoginSecret(barbicanTest.Instance.Namespace, PKCS11LoginSecret))
			DeferCleanup(k8sClient.Delete, ctx, CreatePKCS11ClientDataSecret(barbicanTest.Instance.Namespace, PKCS11ClientDataSecret, "proteccio"))

			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetPKCS11BarbicanSpec("proteccio")))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			infra.SimulateTransportURLReady(barbicanTest.BarbicanTransportURL)
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace))
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetPKCS11BarbicanAPISpec("proteccio")))
			th.SimulateJobSuccess(barbicanTest.BarbicanPKCS11Prep)
		})

		It("Creates BarbicanAPI", func() {
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionTrue,
			)

			BarbicanAPIExists(barbicanTest.Instance)

			d := th.GetDeployment(barbicanTest.BarbicanAPIDeployment)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))

			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(4))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			container := d.Spec.Template.Spec.Containers[1]

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))

			// Checking the PKCS11 Client Data container
			Expect(container.Name).To(Equal(barbican.ComponentAPI))
			foundMount := false
			indexMount := 0
			for index, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == barbican.PKCS11ClientDataVolume {
					foundMount = true
					indexMount = index
					break
				}
			}
			Expect(foundMount).To(BeTrue())
			Expect(container.VolumeMounts[indexMount].MountPath).To(Equal(barbican.PKCS11ClientDataMountPoint))
		})

		It("Verifies the Barbican PKCS11 struct is in good shape", func() {
			Barbican := GetBarbican(barbicanTest.Instance)
			Expect(Barbican.Spec.EnabledSecretStores).Should(Equal([]barbicanv1beta1.SecretStore{"pkcs11"}))
			Expect(Barbican.Spec.GlobalDefaultSecretStore).Should(Equal(barbicanv1beta1.SecretStore("pkcs11")))

			pkcs11 := Barbican.Spec.PKCS11
			Expect(pkcs11.LoginSecret).Should(Equal(PKCS11LoginSecret))
			Expect(pkcs11.ClientDataSecret).Should(Equal(PKCS11ClientDataSecret))
			Expect(pkcs11.ClientDataPath).Should(Equal(PKCS11ClientDataPath["proteccio"]))
		})

		It("Verifies the BarbicanAPI PKCS11 struct is in good shape", func() {
			BarbicanAPI := GetBarbicanAPI(barbicanTest.Instance)
			Expect(BarbicanAPI.Spec.EnabledSecretStores).Should(Equal([]barbicanv1beta1.SecretStore{"pkcs11"}))
			Expect(BarbicanAPI.Spec.GlobalDefaultSecretStore).Should(Equal(barbicanv1beta1.SecretStore("pkcs11")))

			pkcs11 := BarbicanAPI.Spec.PKCS11
			Expect(pkcs11.LoginSecret).Should(Equal(PKCS11LoginSecret))
			Expect(pkcs11.ClientDataSecret).Should(Equal(PKCS11ClientDataSecret))
			Expect(pkcs11.ClientDataPath).Should(Equal(PKCS11ClientDataPath["proteccio"]))
		})

		It("Checks if the two relevant secrets have the right contents", func() {
			hsmSecret := th.GetSecret(barbicanTest.BarbicanPKCS11LoginSecret)
			Expect(hsmSecret).ShouldNot(BeNil())
			confPKCS11 := hsmSecret.Data["PKCS11Pin"]
			Expect(confPKCS11).To(
				ContainSubstring("12345678"))

			clientDataSecret := th.GetSecret(barbicanTest.BarbicanPKCS11ClientDataSecret)
			Expect(clientDataSecret).ShouldNot(BeNil())
			confClient := clientDataSecret.Data["proteccio.rc"]
			Expect(confClient).To(
				ContainSubstring("dummy-data"))

			confServer := clientDataSecret.Data["Server.CRT"]
			Expect(confServer).To(
				ContainSubstring("dummy-data"))

			confClient = clientDataSecret.Data["Client.crt"]
			Expect(confClient).To(
				ContainSubstring("dummy-data"))

			confKey := clientDataSecret.Data["Client.key"]
			Expect(confKey).To(
				ContainSubstring("dummy-data"))
		})

		It("Verifies if 00-default.conf, barbican-api-config.json and 01-custom.conf have the right contents for Barbican.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["00-default.conf"]
			Expect(conf).To(
				ContainSubstring("stores_lookup_suffix = pkcs11"))
			Expect(conf).To(
				ContainSubstring("[secretstore:pkcs11]\nsecret_store_plugin = store_crypto\ncrypto_plugin = p11_crypto\nglobal_default = true"))
			Expect(conf).To(
				ContainSubstring("[p11_crypto_plugin]\nlogin = 12345678"))

			conf = confSecret.Data["01-custom.conf"]
			Expect(conf).To(
				ContainSubstring(PKCS11CustomData))

			conf = confSecret.Data["barbican-api-config.json"]
			Expect(conf).To(
				ContainSubstring("\"source\": \"/var/lib/config-data/hsm\""))
			Expect(conf).To(
				ContainSubstring("\"dest\": \"/etc/proteccio\""))
		})

		It("Verifies if 00-default.conf and 01-custom.conf have the right contents for BarbicanAPI.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["00-default.conf"]
			Expect(conf).To(
				ContainSubstring("stores_lookup_suffix = pkcs11"))
			Expect(conf).To(
				ContainSubstring("[secretstore:pkcs11]\nsecret_store_plugin = store_crypto\ncrypto_plugin = p11_crypto\nglobal_default = true"))
			Expect(conf).To(
				ContainSubstring("[p11_crypto_plugin]\nlogin = 12345678"))

			conf = confSecret.Data["01-custom.conf"]
			Expect(conf).To(
				ContainSubstring(PKCS11CustomData))
		})

		It("Verifies if barbican-api-config.json has the right contents for BarbicanAPI.", func() {
			confSecret := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(confSecret).ShouldNot(BeNil())

			conf := confSecret.Data["barbican-api-config.json"]
			Expect(conf).To(
				ContainSubstring("\"source\": \"/var/lib/config-data/hsm\""))
			Expect(conf).To(
				ContainSubstring("\"dest\": \"/etc/proteccio\""))
		})

		It("Checks if the PKCS11PreJob successfully executed", func() {
			BarbicanExists(barbicanTest.Instance)

			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				controllers.PKCS11PrepReadyCondition,
				corev1.ConditionTrue,
			)

			// Checking if the volume mount name and mount path match the specified values.
			var elemClient, elemScript, elemConfig = 0, 0, 0
			for index, mount := range th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts {
				if mount.Name == barbican.PKCS11ClientDataVolume {
					elemClient = index
				} else if mount.Name == barbican.ScriptVolume {
					elemScript = index
				} else if mount.Name == barbican.ConfigVolume && mount.SubPath == "" {
					elemConfig = index
				}
			}

			volume := th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemClient].Name
			mountPath := th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemClient].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.PKCS11ClientDataVolume))
				g.Expect(mountPath).To(Equal(barbican.PKCS11ClientDataMountPoint))
			}, timeout, interval).Should(Succeed())

			volume = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemScript].Name
			mountPath = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemScript].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.ScriptVolume))
				g.Expect(mountPath).To(Equal(barbican.ScriptMountPoint))
			}, timeout, interval).Should(Succeed())

			volume = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemConfig].Name
			mountPath = th.GetJob(barbicanTest.BarbicanPKCS11Prep).Spec.Template.Spec.Containers[0].VolumeMounts[elemConfig].MountPath

			Eventually(func(g Gomega) {
				g.Expect(volume).To(Equal(barbican.ConfigVolume))
				g.Expect(mountPath).To(Equal(barbican.ConfigMountPoint))
			}, timeout, interval).Should(Succeed())
		})
	})

	When("Deployment rollout is progressing", func() {
		BeforeEach(func() {
			spec := GetDefaultBarbicanSpec()
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, "rabbitmq-secret"))
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, spec))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))

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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)
			// API, Worker and KeystoneListener deployment in progress
			th.SimulateDeploymentProgressing(barbicanTest.BarbicanAPIDeployment)
			th.SimulateDeploymentProgressing(barbicanTest.BarbicanWorkerDeployment)
			th.SimulateDeploymentProgressing(barbicanTest.BarbicanKeystoneListenerDeployment)
		})

		It("shows the BarbicanAPI deployment progressing in DeploymentReadyCondition", func() {
			// BarbicanAPI - deployment progressing
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("shows the BarbicanKeystoneListener deployment progressing in DeploymentReadyCondition", func() {
			// BarbicanKeystoneListener - deployment progressing
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("shows the BarbicanWorker deployment progressing in DeploymentReadyCondition", func() {
			// BarbicanWorker - deployment progressing
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("still shows the BarbicanAPI deployment progressing in DeploymentReadyCondition when rollout hits ProgressDeadlineExceeded", func() {
			th.SimulateDeploymentProgressDeadlineExceeded(barbicanTest.BarbicanAPIDeployment)
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("still shows the BarbicanKeystoneListener deployment progressing in DeploymentReadyCondition when rollout hits ProgressDeadlineExceeded", func() {
			th.SimulateDeploymentProgressDeadlineExceeded(barbicanTest.BarbicanKeystoneListenerDeployment)
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("still shows the BarbicanWorker deployment progressing in DeploymentReadyCondition when rollout hits ProgressDeadlineExceeded", func() {
			th.SimulateDeploymentProgressDeadlineExceeded(barbicanTest.BarbicanWorker)
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("BarbicanAPI reaches Ready when deployment rollout finished", func() {
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanAPIDeployment)
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionTrue,
			)

			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("BarbicanKeystoneListener reaches Ready when deployment rollout finished", func() {
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanKeystoneListenerDeployment)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionTrue,
			)

			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("BarbicanWorker reaches Ready when deployment rollout finished", func() {
			th.ExpectConditionWithDetails(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionFalse,
				condition.RequestedReason,
				condition.DeploymentReadyRunningMessage,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanWorker)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.DeploymentReadyCondition,
				corev1.ConditionTrue,
			)

			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)

			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("Barbican overall condition reaches ready when all deployments succeeded", func() {
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)
			// overall Barbican condition false
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionFalse,
			)

			// set all deployments to finished
			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanAPI)
			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanKeystoneListener)
			th.SimulateDeploymentReplicaReady(barbicanTest.BarbicanWorker)
			th.ExpectCondition(
				barbicanTest.BarbicanAPI,
				ConditionGetterFunc(BarbicanAPIConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanKeystoneListener,
				ConditionGetterFunc(BarbicanKeystoneListenerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				barbicanTest.BarbicanWorker,
				ConditionGetterFunc(BarbicanWorkerConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
			// overall Barbican condition true
			th.ExpectCondition(
				barbicanTest.Barbican,
				ConditionGetterFunc(BarbicanConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionTrue,
			)
		})
	})

	// Run MariaDBAccount suite tests.  these are pre-packaged ginkgo tests
	// that exercise standard account create / update patterns that should be
	// common to all controllers that ensure MariaDBAccount CRs.
	mariadbSuite := &mariadb_test.MariaDBTestHarness{
		PopulateHarness: func(harness *mariadb_test.MariaDBTestHarness) {
			harness.Setup(
				"Barbican",
				barbicanTest.Instance.Namespace,
				barbicanTest.Instance.Name,
				"openstack.org/barbican",
				mariadb, timeout, interval,
			)
		},

		// Generate a fully running service given an accountName
		// needs to make it all the way to the end where the mariadb finalizers
		// are removed from unused accounts since that's part of what we are testing
		SetupCR: func(accountName types.NamespacedName) {

			spec := GetDefaultBarbicanSpec()
			spec["databaseAccount"] = accountName.Name

			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, spec))

			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetTLSBarbicanAPISpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanSecret(barbicanTest.Instance.Namespace, SecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(accountName)
			mariadb.SimulateMariaDBTLSDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)

			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(barbicanTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(barbicanTest.PublicCertSecret))
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)

		},
		// Change the account name in the service to a new name
		UpdateAccount: func(newAccountName types.NamespacedName) {

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				barbican.Spec.DatabaseAccount = newAccountName.Name
				g.Expect(th.K8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

		},
		// delete the CR instance to exercise finalizer removal
		DeleteCR: func() {
			th.DeleteInstance(GetBarbican(barbicanName))
		},
	}

	mariadbSuite.RunBasicSuite()

	mariadbSuite.RunURLAssertSuite(func(_ types.NamespacedName, username string, password string) {
		Eventually(func(g Gomega) {
			secretDataMap := th.GetSecret(barbicanTest.BarbicanConfigSecret)

			conf := secretDataMap.Data["00-default.conf"]

			g.Expect(string(conf)).Should(
				ContainSubstring(fmt.Sprintf("sql_connection = mysql+pymysql://%s:%s@hostname-for-openstack.%s.svc/%s?read_default_file=/etc/my.cnf",
					username, password, namespace, barbican.DatabaseName)))
			g.Expect(string(conf)).Should(
				ContainSubstring(fmt.Sprintf("connection = mysql+pymysql://%s:%s@hostname-for-openstack.%s.svc/%s?read_default_file=/etc/my.cnf",
					username, password, namespace, barbican.DatabaseName)))

		}).Should(Succeed())

	})

})

var _ = Describe("Barbican Webhook", func() {

	BeforeEach(func() {
		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects with wrong BarbicanAPI service override endpoint type", func() {
		spec := GetDefaultBarbicanSpec()
		apiSpec := GetDefaultBarbicanAPISpec()
		apiSpec["override"] = map[string]interface{}{
			"service": map[string]interface{}{
				"internal": map[string]interface{}{},
				"wrooong":  map[string]interface{}{},
			},
		}
		spec["barbicanAPI"] = apiSpec

		raw := map[string]interface{}{
			"apiVersion": "barbican.openstack.org/v1beta1",
			"kind":       "Barbican",
			"metadata": map[string]interface{}{
				"name":      barbicanTest.Instance.Name,
				"namespace": barbicanTest.Instance.Namespace,
			},
			"spec": spec,
		}

		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(
			ContainSubstring(
				"invalid: spec.barbicanAPI.override.service[wrooong]: " +
					"Invalid value: \"wrooong\": invalid endpoint type: wrooong"),
		)
	})
	DescribeTable("rejects wrong topology for",
		func(serviceNameFunc func() (string, string)) {

			component, errorPath := serviceNameFunc()
			expectedErrorMessage := fmt.Sprintf("spec.%s.namespace: Invalid value: \"namespace\": Customizing namespace field is not supported", errorPath)

			spec := GetDefaultBarbicanAPISpec()

			// API, Worker and KeystoneListener
			if component != "top-level" {
				spec[component] = map[string]interface{}{
					"topologyRef": map[string]interface{}{
						"name":      "bar",
						"namespace": "foo",
					},
				}
				// top-level topologyRef
			} else {
				spec["topologyRef"] = map[string]interface{}{
					"name":      "bar",
					"namespace": "foo",
				}
			}
			// Build the barbican CR
			raw := map[string]interface{}{
				"apiVersion": "barbican.openstack.org/v1beta1",
				"kind":       "Barbican",
				"metadata": map[string]interface{}{
					"name":      barbicanTest.Instance.Name,
					"namespace": barbicanTest.Instance.Namespace,
				},
				"spec": spec,
			}
			unstructuredObj := &unstructured.Unstructured{Object: raw}
			_, err := controllerutil.CreateOrPatch(
				th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(
				ContainSubstring(expectedErrorMessage))
		},
		Entry("top-level topologyRef", func() (string, string) {
			return "top-level", "topologyRef"
		}),
		Entry("barbicanAPI topologyRef", func() (string, string) {
			component := "barbicanAPI"
			return component, fmt.Sprintf("%s.topologyRef", component)
		}),
		Entry("barbicanKeystoneListener topologyRef", func() (string, string) {
			component := "barbicanKeystoneListener"
			return component, fmt.Sprintf("%s.topologyRef", component)
		}),
		Entry("barbicanWorker topologyRef", func() (string, string) {
			component := "barbicanWorker"
			return component, fmt.Sprintf("%s.topologyRef", component)
		}),
	)
})
