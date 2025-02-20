package functional

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	barbicanv1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	test "github.com/openstack-k8s-operators/lib-common/modules/test"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/openstack-k8s-operators/barbican-operator/controllers"
	infra_test "github.com/openstack-k8s-operators/infra-operator/apis/test/helpers"
	keystone_test "github.com/openstack-k8s-operators/keystone-operator/api/test/helpers"
	common_test "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	mariadb_test "github.com/openstack-k8s-operators/mariadb-operator/api/test/helpers"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg          *rest.Config
	k8sClient    client.Client
	testEnv      *envtest.Environment
	ctx          context.Context
	cancel       context.CancelFunc
	logger       logr.Logger
	th           *common_test.TestHelper
	keystone     *keystone_test.TestHelper
	mariadb      *mariadb_test.TestHelper
	infra        *infra_test.TestHelper
	namespace    string
	barbicanName types.NamespacedName
	barbicanTest BarbicanTestData
)

const (
	timeout = time.Second * 15

	SecretName = "test-osp-secret"

	interval = time.Millisecond * 200

	// PKCS11 Constants
	PKCS11ClientDataPath   = "/usr/local/luna"
	PKCS11LoginSecret      = "pkcs11-login"
	PKCS11ClientDataSecret = "pkcs11-client-data"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true), func(o *zap.Options) {
		o.Development = true
		o.TimeEncoder = zapcore.ISO8601TimeEncoder
	}))

	ctx, cancel = context.WithCancel(context.TODO())

	keystoneCRDs, err := test.GetCRDDirFromModule(
		"github.com/openstack-k8s-operators/keystone-operator/api", "../../go.mod", "bases")
	Expect(err).ShouldNot(HaveOccurred())
	networkv1CRD, err := test.GetCRDDirFromModule(
		"github.com/k8snetworkplumbingwg/network-attachment-definition-client", "../../go.mod", "artifacts/networks-crd.yaml")
	Expect(err).ShouldNot(HaveOccurred())

	rabbitmqCRDs, err := test.GetCRDDirFromModule(
		"github.com/openstack-k8s-operators/infra-operator/apis", "../../go.mod", "bases")
	Expect(err).ShouldNot(HaveOccurred())

	mariaDBCRDs, err := test.GetCRDDirFromModule(
		"github.com/openstack-k8s-operators/mariadb-operator/api", "../../go.mod", "bases")
	Expect(err).ShouldNot(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			keystoneCRDs,
			rabbitmqCRDs,
			mariaDBCRDs,
		},
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{
				networkv1CRD,
			},
		},

		ErrorIfCRDPathMissing: true,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
			// NOTE(gibi): if localhost is resolved to ::1 (ipv6) then starting
			// the webhook fails as it try to parse the address as ipv4 and
			// failing on the colons in ::1
			LocalServingHost: "127.0.0.1",
		},
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = barbicanv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = keystonev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = mariadbv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = rabbitmqv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = networkv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = topologyv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	logger = ctrl.Log.WithName("---Test---")
	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	th = common_test.NewTestHelper(ctx, k8sClient, timeout, interval, logger)
	Expect(th).NotTo(BeNil())
	keystone = keystone_test.NewTestHelper(ctx, k8sClient, timeout, interval, logger)
	Expect(keystone).NotTo(BeNil())
	mariadb = mariadb_test.NewTestHelper(ctx, k8sClient, timeout, interval, logger)
	Expect(mariadb).NotTo(BeNil())
	infra = infra_test.NewTestHelper(ctx, k8sClient, timeout, interval, logger)
	Expect(infra).NotTo(BeNil())

	// Start the controller-manager if goroutine
	webhookInstallOptions := &testEnv.WebhookInstallOptions
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		WebhookServer: webhook.NewServer(
			webhook.Options{
				Host:    webhookInstallOptions.LocalServingHost,
				Port:    webhookInstallOptions.LocalServingPort,
				CertDir: webhookInstallOptions.LocalServingCertDir,
			}),
		LeaderElection: false,
	})
	Expect(err).ToNot(HaveOccurred())

	kclient, err := kubernetes.NewForConfig(cfg)
	Expect(err).ToNot(HaveOccurred(), "failed to create kclient")

	err = (&controllers.BarbicanReconciler{
		Client:  k8sManager.GetClient(),
		Scheme:  k8sManager.GetScheme(),
		Kclient: kclient,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.BarbicanAPIReconciler{
		Client:  k8sManager.GetClient(),
		Scheme:  k8sManager.GetScheme(),
		Kclient: kclient,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.BarbicanKeystoneListenerReconciler{
		Client:  k8sManager.GetClient(),
		Scheme:  k8sManager.GetScheme(),
		Kclient: kclient,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
	barbicanv1.SetupDefaults()

	err = (&controllers.BarbicanWorkerReconciler{
		Client:  k8sManager.GetClient(),
		Scheme:  k8sManager.GetScheme(),
		Kclient: kclient,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())
	barbicanv1.SetupDefaults()

	err = (&barbicanv1.Barbican{}).SetupWebhookWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// wait for the webhook server to get ready
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true}) // #nosec G402
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = BeforeEach(func() {
	// NOTE(gibi): We need to create a unique namespace for each test run
	// as namespaces cannot be deleted in a locally running envtest. See
	// https://book.kubebuilder.io/reference/envtest.html#namespace-usage-limitation
	namespace = uuid.New().String()
	th.CreateNamespace(namespace)
	// We still request the delete of the Namespace to properly cleanup if
	// we run the test in an existing cluster.
	barbicanName = types.NamespacedName{
		Namespace: namespace,
		Name:      "barbican",
	}

	barbicanTest = GetBarbicanTestData(barbicanName)

	DeferCleanup(th.DeleteNamespace, namespace)
})
