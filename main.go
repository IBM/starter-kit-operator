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

	coreappsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	devxv1alpha1 "github.com/IBM/starter-kit-operator/api/v1alpha1"
	"github.com/IBM/starter-kit-operator/controllers"
	"github.com/rs/zerolog/log"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	consolev1client "github.com/openshift/client-go/console/clientset/versioned/typed/console/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(devxv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "1501cde2.my.domain",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ReconcileStarterKit{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("StarterKit"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StarterKit")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// ========================================================================
	namespace, err := k8sutil.GetWatchNamespace()
	if disableUI, ok := os.LookupEnv("DISABLE_SKIT_OPERATOR_UI"); ok {
		if disableUI == "true" {
			log.Logger.Info().Msg("The UI for the Starter Kit Operator will not be installed")
		}
	} else {
		log.Logger.Info().Msg("Installing UI resources")
		coreclient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
		routev1client := routev1client.NewForConfigOrDie(mgr.GetConfig())
		consolev1client := consolev1client.NewForConfigOrDie(mgr.GetConfig())
		// Set operator deployment instance as the owner and controller of all resources so that they get deleted when the operator is uninstalled
		operatorDeployment := &coreappsv1.Deployment{}
		operatorDeployment, err = coreclient.AppsV1().Deployments(namespace).Get("starter-kit-operator", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Error().Msg("Could not find Operator Deployment: " + err.Error())
		}

		var uiImageAccount, uiImageVersion string
		if imgAccountVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_ACCOUNT"); ok {
			log.Logger.Info().Msg("Using UI image account " + imgAccountVar)
			uiImageAccount = imgAccountVar
		} else {
			uiImageAccount = controllers.DefaultUIImageAccount
		}
		if imgVersionVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_VERSION"); ok {
			log.Logger.Info().Msg("Using UI image version " + imgVersionVar)
			uiImageVersion = imgVersionVar
		} else {
			uiImageVersion = controllers.DefaultUIImageVersion
		}
		log.Logger.Info().Msg("Using UI image account " + uiImageAccount + ", version " + uiImageVersion)

		// deployment
		foundDeployment := &coreappsv1.Deployment{}
		foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Get(controllers.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Logger.Info().Msg("Creating a new Deployment for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			uiDeployment := controllers.NewDeploymentForUI(namespace, uiImageAccount, uiImageVersion)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiDeployment, mgr.GetScheme()); err != nil {
				log.Error(err, "Error setting Operator Deployment as owner of UI Deployment")
			}
			foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Create(uiDeployment)
			if err != nil {
				log.Error(err, "Error creating new Deployment for the UI")
			}

			// Deployment created successfully
			log.Logger.Info().Msg("Deployment for UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Deployment for the UI")
		} else {
			// Deployment already exists - don't requeue
			log.Logger.Info().Msg("Skip reconcile: Deployment for the UI already exists", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
		}

		// service
		foundService := &corev1.Service{}
		foundService, err = coreclient.CoreV1().Services(namespace).Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Logger.Info().Msg("Creating a new Service for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			uiService := starterkit.NewServiceForUI(namespace)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiService, mgr.GetScheme()); err != nil {
				log.Error(err, "Error setting Operator Deployment as owner of UI Service")
			}
			foundService, err = coreclient.CoreV1().Services(namespace).Create(uiService)
			if err != nil {
				log.Error(err, "Error creating Service for the UI")
			}

			// Service created successfully
			log.Logger.Info().Msg("Service for the UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Service for the UI")
		} else {
			// Service already exists - don't requeue
			log.Logger.Info().Msg("Skip reconcile: Service for the UI already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
		}

		// route for UI
		foundRoute := &routev1.Route{}
		foundRoute, err = routev1client.Routes(namespace).Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Logger.Info().Msg("Creating a new Route for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			uiRoute := starterkit.NewRouteForUI(namespace)
			if err := controllerutil.SetControllerReference(operatorDeployment, uiRoute, mgr.GetScheme()); err != nil {
				log.Error(err, "Error setting Operator Deployment as owner of UI Route")
			}
			foundRoute, err = routev1client.Routes(namespace).Create(uiRoute)
			if err != nil {
				log.Error(err, "Error creating Route for the UI")
			}

			// Route created successfully
			log.Logger.Info().Msg("Route for the UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Route for the UI")
		} else {
			// Route already exists - don't requeue
			log.Logger.Info().Msg("Skip reconcile: Route for the UI already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
		}

		// console link for UI
		foundConsoleLink := &consolev1.ConsoleLink{}
		foundConsoleLink, err = consolev1client.ConsoleLinks().Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Logger.Info().Msg("Creating a new ConsoleLink for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			consoleLink := starterkit.NewConsoleLinkForUI(namespace, "https://"+foundRoute.Spec.Host)
			if err := controllerutil.SetControllerReference(operatorDeployment, consoleLink, mgr.GetScheme()); err != nil {
				log.Error(err, "Error setting Operator Deployment as owner of UI ConsoleLink")
			}
			foundConsoleLink, err = consolev1client.ConsoleLinks().Create(consoleLink)
			if err != nil {
				log.Error(err, "Error creating ConsoleLink for the UI")
			}

			// ConsoleLink created successfully
			log.Logger.Info().Msg("ConsoleLink for the UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching ConsoleLink for the UI")
		} else {
			// ConsoleLink already exists - don't requeue
			log.Logger.Info().Msg("Skip reconcile: ConsoleLink for the UI already exists", "ConsoleLink.Namespace", foundConsoleLink.Namespace, "ConsoleLink.Name", foundConsoleLink.Name)
		}
	}

	setuplog.Logger.Info().Msg("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
