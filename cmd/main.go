/*
Copyright 2023 The Nephio Authors.

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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	runscheme "sigs.k8s.io/controller-runtime/pkg/scheme"

	actorConfigv1 "actor/api/v1"

	refv1alpha1 "github.com/nephio-project/api/references/v1alpha1"
	workloadv1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	"workload.nephio.org/ran_deployment/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "27226901.workload.nephio.org",
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

	schemeBuilder := &runscheme.Builder{GroupVersion: refv1alpha1.GroupVersion}
	schemeBuilder.Register(&refv1alpha1.Config{}, &refv1alpha1.ConfigList{})
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "Not able to register references/v1alpha1 Config kind")
	}

	schemeBuilder = &runscheme.Builder{GroupVersion: workloadv1alpha1.GroupVersion}
	schemeBuilder.Register(&workloadv1alpha1.NFConfig{}, &workloadv1alpha1.NFConfigList{})
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "Not able to register workload/v1alpha1 NFConfig kind")
	}

	schemeBuilder = &runscheme.Builder{GroupVersion: workloadv1alpha1.GroupVersion}
	schemeBuilder.Register(&workloadv1alpha1.NFDeployment{}, &workloadv1alpha1.NFDeploymentList{})
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "Not able to register workload/v1alpha1 NFDeployment kind")
	}

	schemeBuilder = &runscheme.Builder{GroupVersion: actorConfigv1.GroupVersion}
	schemeBuilder.Register(&actorConfigv1.ActorConfig{}, &actorConfigv1.ActorConfigList{})
	if err := schemeBuilder.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "Not able to register actorConfigv1 ActorConfig kind")
	}

	if err = (&controller.RANDeploymentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RANDeployment")
		os.Exit(1)
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
