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
	"github.com/openstack-k8s-operators/barbican-operator/controllers"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
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
			cf := th.GetSecret(barbicanTest.BarbicanAPIConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			httpdConfData := string(cf.Data["10-barbican_wsgi_main.conf"])
			Expect(httpdConfData).To(
				ContainSubstring("TimeOut 90"),
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBTLSDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
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

		It("should create config-data and scripts ConfigMaps", func() {
			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			conf := cf.Data["my.cnf"]
			Expect(conf).To(
				ContainSubstring("[client]\nssl-ca=/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem\nssl=1"))
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
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)
			th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
		})

		It("sets nodeSelector in resource specs", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())
		})

		It("updates nodeSelector in resource specs when changed", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
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
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when cleared", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
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
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when nilled", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				barbican := GetBarbican(barbicanName)
				barbican.Spec.NodeSelector = nil
				g.Expect(k8sClient.Update(ctx, barbican)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(barbicanTest.BarbicanDBSync)
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
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
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "api"}))
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override to empty", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetJob(barbicanTest.BarbicanDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
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
				g.Expect(th.GetDeployment(barbicanTest.BarbicanAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})

	When("A Barbican with HSM is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateBarbican(barbicanTest.Instance, GetHSMBarbicanSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateBarbicanMessageBusSecret(barbicanTest.Instance.Namespace, barbicanTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateBarbicanAPI(barbicanTest.Instance, GetHSMBarbicanAPISpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateKeystoneAPISecret(barbicanTest.Instance.Namespace, SecretName))
			// keystoneAPI := keystone.CreateKeystoneAPI(barbicanTest.Instance.Namespace)
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
			// DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPI)
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

			d := th.GetDeployment(barbicanTest.BarbicanAPI)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))

			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(3))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			container := d.Spec.Template.Spec.Containers[1]

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTP))
		})

		It("Should have the right configuration contents", func() {
			keystone.SimulateKeystoneEndpointReady(barbicanTest.BarbicanKeystoneEndpoint)
			mariadb.SimulateMariaDBAccountCompleted(barbicanTest.BarbicanDatabaseAccount)
			mariadb.SimulateMariaDBDatabaseCompleted(barbicanTest.BarbicanDatabaseName)

			cf := th.GetSecret(barbicanTest.BarbicanConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			confChrystoki := cf.Data["Chrystoki.conf"]
			Expect(confChrystoki).To(
				ContainSubstring("Luna = {\n  DefaultTimeOut = 500000;\n  PEDTimeout1 = 100000;\n  PEDTimeout2 = 200000;"))
			confDefault := cf.Data["00-default.conf"]
			Expect(confDefault).To(
				ContainSubstring("[secretstore:pkcs11]"))
		})

		It("Should have the relevant conditions in the right state", func() {
			th.ExpectCondition(
				barbicanTest.Instance,
				ConditionGetterFunc(BarbicanConditionGetter),
				controllers.P11PrepReadyCondition,
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
})
