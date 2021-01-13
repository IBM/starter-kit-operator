/*
Copyright 2021.

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
	"context"
	"flag"
	"os"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	consolev1client "github.com/openshift/client-go/console/clientset/versioned/typed/console/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	coreappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/api/v1alpha1"
	"github.com/ibm/starter-kit-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(buildv1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
	utilruntime.Must(imagev1.AddToScheme(scheme))
	utilruntime.Must(routev1.AddToScheme(scheme))
	utilruntime.Must(consolev1.AddToScheme(scheme))

	utilruntime.Must(devxv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
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

	clientCfg, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	namespace := clientCfg.Contexts[clientCfg.CurrentContext].Namespace
	if namespace == "" {
		namespace = "default"
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f1c5ece8.example.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.StarterKitReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("StarterKit"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StarterKit")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// ========================================================================
	if disableUI, ok := os.LookupEnv("DISABLE_SKIT_OPERATOR_UI"); ok {
		if disableUI == "true" {
			setupLog.Info("The UI for the Starter Kit Operator will not be installed")
		}
	} else {
		setupLog.Info("Installing UI resources")
		coreclient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
		routev1client := routev1client.NewForConfigOrDie(mgr.GetConfig())
		consolev1client := consolev1client.NewForConfigOrDie(mgr.GetConfig())
		// Set operator deployment instance as the owner and controller of all resources so that they get deleted when the operator is uninstalled
		operatorDeployment := &coreappsv1.Deployment{}
		operatorDeployment, err = coreclient.AppsV1().Deployments(namespace).Get(context.TODO(), "starter-kit-operator", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			setupLog.Error(err, "Could not find Operator Deployment")
		}

		var uiImageAccount, uiImageVersion string
		if imgAccountVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_ACCOUNT"); ok {
			setupLog.Info("Using UI image account " + imgAccountVar)
			uiImageAccount = imgAccountVar
		} else {
			uiImageAccount = controllers.DefaultUIImageAccount
		}
		if imgVersionVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_VERSION"); ok {
			setupLog.Info("Using UI image version " + imgVersionVar)
			uiImageVersion = imgVersionVar
		} else {
			uiImageVersion = controllers.DefaultUIImageVersion
		}
		setupLog.Info("Using UI image account " + uiImageAccount + ", version " + uiImageVersion)

		// deployment
		foundDeployment := &coreappsv1.Deployment{}
		foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Get(context.TODO(), controllers.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			setupLog.Info("Creating a new Deployment for the UI", "Namespace", namespace, "Name", controllers.UIName)
			uiDeployment := controllers.NewDeploymentForUI(namespace, uiImageAccount, uiImageVersion)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiDeployment, mgr.GetScheme()); err != nil {
				setupLog.Error(err, "Error setting Operator Deployment as owner of UI Deployment")
			}
			foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Create(context.TODO(), uiDeployment, metav1.CreateOptions{})
			if err != nil {
				setupLog.Error(err, "Error creating new Deployment for the UI")
			}

			// Deployment created successfully
			setupLog.Info("Deployment for UI created successfully")
		} else if err != nil {
			setupLog.Error(err, "Error fetching Deployment for the UI")
		} else {
			// Deployment already exists - don't requeue
			setupLog.Info("Skip reconcile: Deployment for the UI already exists", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
		}

		// service
		foundService := &corev1.Service{}
		foundService, err = coreclient.CoreV1().Services(namespace).Get(context.TODO(), controllers.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			setupLog.Info("Creating a new Service for the UI", "Namespace", namespace, "Name", controllers.UIName)
			uiService := controllers.NewServiceForUI(namespace)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiService, mgr.GetScheme()); err != nil {
				setupLog.Error(err, "Error setting Operator Deployment as owner of UI Service")
			}
			foundService, err = coreclient.CoreV1().Services(namespace).Create(context.TODO(), uiService, metav1.CreateOptions{})
			if err != nil {
				setupLog.Error(err, "Error creating Service for the UI")
			}

			// Service created successfully
			setupLog.Info("Service for the UI created successfully")
		} else if err != nil {
			setupLog.Error(err, "Error fetching Service for the UI")
		} else {
			// Service already exists - don't requeue
			setupLog.Info("Skip reconcile: Service for the UI already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
		}

		// route for UI
		foundRoute := &routev1.Route{}
		foundRoute, err = routev1client.Routes(namespace).Get(context.TODO(), controllers.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			setupLog.Info("Creating a new Route for the UI", "Namespace", namespace, "Name", controllers.UIName)
			uiRoute := controllers.NewRouteForUI(namespace)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiRoute, mgr.GetScheme()); err != nil {
				setupLog.Error(err, "Error setting Operator Deployment as owner of UI Route")
			}
			foundRoute, err = routev1client.Routes(namespace).Create(context.TODO(), uiRoute, metav1.CreateOptions{})
			if err != nil {
				setupLog.Error(err, "Error creating Route for the UI")
			}

			// Route created successfully
			setupLog.Info("Route for the UI created successfully")
		} else if err != nil {
			setupLog.Error(err, "Error fetching Route for the UI")
		} else {
			// Route already exists - don't requeue
			setupLog.Info("Skip reconcile: Route for the UI already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
		}

		// console link for UI
		foundConsoleLink := &consolev1.ConsoleLink{}
		foundConsoleLink, err = consolev1client.ConsoleLinks().Get(context.TODO(), controllers.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			setupLog.Info("Creating a new ConsoleLink for the UI", "Namespace", namespace, "Name", controllers.UIName)
			consoleLink := controllers.NewConsoleLinkForUI(namespace, "https://"+foundRoute.Spec.Host)
			if err := controllerutil.SetControllerReference(operatorDeployment, consoleLink, mgr.GetScheme()); err != nil {
				setupLog.Error(err, "Error setting Operator Deployment as owner of UI ConsoleLink")
			}
			foundConsoleLink, err = consolev1client.ConsoleLinks().Create(context.TODO(), consoleLink, metav1.CreateOptions{})
			if err != nil {
				setupLog.Error(err, "Error creating ConsoleLink for the UI")
			}

			// ConsoleLink created successfully
			setupLog.Info("ConsoleLink for the UI created successfully")
		} else if err != nil {
			setupLog.Error(err, "Error fetching ConsoleLink for the UI")
		} else {
			// ConsoleLink already exists - don't requeue
			setupLog.Info("Skip reconcile: ConsoleLink for the UI already exists", "ConsoleLink.Namespace", foundConsoleLink.Namespace, "ConsoleLink.Name", foundConsoleLink.Name)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
