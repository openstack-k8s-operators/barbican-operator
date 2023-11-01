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

package main

import (
	"crypto/tls"
	"flag"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/barbican-operator/controllers"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(barbicanv1beta1.AddToScheme(scheme))
	utilruntime.Must(mariadbv1.AddToScheme(scheme))
	utilruntime.Must(keystonev1.AddToScheme(scheme))
	utilruntime.Must(rabbitmqv1.AddToScheme(scheme))
	utilruntime.Must(networkv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var enableHTTP2 bool
	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the metrics and webhook servers.")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8085", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8086", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	disableHTTP2 := func(c *tls.Config) {
		if enableHTTP2 {
			return
		}
		c.NextProtos = []string{"http/1.1"}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "8cc931b9.openstack.org",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		setupLog.Error(err, "")
		os.Exit(1)
	}
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "")
		os.Exit(1)
	}

	if err = (&controllers.BarbicanReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Kclient: kclient,
		Log:     ctrl.Log.WithName("controllers").WithName("Barbican"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Barbican")
		os.Exit(1)
	}
	if err = (&controllers.BarbicanAPIReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Kclient: kclient,
		Log:     ctrl.Log.WithName("controllers").WithName("BarbicanAPI"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BarbicanAPI")
		os.Exit(1)
	}
	if err = (&controllers.BarbicanWorkerReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Kclient: kclient,
		Log:     ctrl.Log.WithName("controllers").WithName("BarbicanWorker"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BarbicanWorker")
		os.Exit(1)
	}

	if err = (&controllers.BarbicanKeystoneListenerReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Kclient: kclient,
		Log:     ctrl.Log.WithName("controllers").WithName("BarbicanKeystoneListener"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BarbicanKeystoneListener")
		os.Exit(1)
	}

	// Acquire environmental defaults and initialize operator defaults with them
	barbicanv1beta1.SetupDefaults()

	// Setup webhooks if requested
	if strings.ToLower(os.Getenv("ENABLE_WEBHOOKS")) != "false" {
		// overriding the default values
		srv := mgr.GetWebhookServer()
		srv.TLSOpts = []func(config *tls.Config){disableHTTP2}

		if err = (&barbicanv1beta1.Barbican{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Barbican")
			os.Exit(1)
		}
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
