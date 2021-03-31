/*


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

	kfservingv1 "github.com/kubeflow/kfserving/pkg/apis/serving/v1beta1"
	seldonv1 "github.com/seldonio/seldon-core/operator/apis/machinelearning.seldon.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	servingv1 "fuseml.suse/api/v1"
	v1controller "fuseml.suse/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	// scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.Parse()
	logf.SetLogger(zap.New())
	log := logf.Log.WithName("entrypoint")

	// Get a config to talk to the apiserver
	log.Info("Setting up client for manager")
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "unable to set up client config")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	log.Info("Setting up manager")
	mgr, err := manager.New(cfg, manager.Options{MetricsBindAddress: metricsAddr, Port: 9443})
	if err != nil {
		log.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	log.Info("Setting up FuseML v1 scheme")
	if err := servingv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable to add FuseML v1 to scheme")
		os.Exit(1)
	}

	log.Info("Setting up KFServing v1beta1 scheme")
	if err := kfservingv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable to add KFServing v1beta1 to scheme")
		os.Exit(1)
	}

	log.Info("Setting up SeldonCore v1 scheme")
	if err := seldonv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable to add SeldonCore v1 to scheme")
		os.Exit(1)
	}

	log.Info("Setting up core scheme")
	if err := v1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "unable to add Core APIs to scheme")
		os.Exit(1)
	}

	// Setup Controller
	setupLog.Info("Setting up controller")
	eventBroadcaster := record.NewBroadcaster()
	clientSet, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create clientSet")
		os.Exit(1)
	}
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})
	if err = (&v1controller.InferenceServiceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("v1beta1Controllers").WithName("InferenceService"),
		Scheme: mgr.GetScheme(),
		Recorder: eventBroadcaster.NewRecorder(
			mgr.GetScheme(), v1.EventSource{Component: "v1controller"}),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "v1controller", "InferenceService")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}
