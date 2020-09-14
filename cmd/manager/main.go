package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/ibm/starter-kit-operator/pkg/apis"
	"github.com/ibm/starter-kit-operator/pkg/controller"
	"github.com/ibm/starter-kit-operator/pkg/controller/starterkit"

	routev1 "github.com/openshift/api/route/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	coreappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	err = leader.Become(context.Background(), "starter-kit-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err = serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(context.Background(), cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}

	// ========================================================================
	if disableUI, ok := os.LookupEnv("DISABLE_SKIT_OPERATOR_UI"); ok {
		if disableUI == "true" {
			log.Info("The UI for the Starter Kit Operator will not be installed")
		}
	} else {
		log.Info("Installing UI resources")
		coreclient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
		routev1client := routev1client.NewForConfigOrDie(mgr.GetConfig())
		// Set operator deployment instance as the owner and controller of all resources so that they get deleted when the operator is uninstalled
		operatorDeployment := &coreappsv1.Deployment{}
		operatorDeployment, err = coreclient.AppsV1().Deployments(namespace).Get("starter-kit-operator", metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Error(err, "Could not find Operator Deployment")
		}

		var uiImageAccount, uiImageVersion string
		if imgAccountVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_ACCOUNT"); ok {
			log.Info("Using UI image account " + imgAccountVar)
			uiImageAccount = imgAccountVar
		} else {
			uiImageAccount = starterkit.DefaultUIImageAccount
		}
		if imgVersionVar, ok := os.LookupEnv("SKIT_OPERATOR_UI_IMAGE_VERSION"); ok {
			log.Info("Using UI image version " + imgVersionVar)
			uiImageVersion = imgVersionVar
		} else {
			uiImageVersion = starterkit.DefaultUIImageVersion
		}

		// deployment
		uiDeployment := starterkit.NewDeploymentForUI(namespace, uiImageAccount, uiImageVersion)
		if err := controllerutil.SetControllerReference(operatorDeployment, uiDeployment, mgr.GetScheme()); err != nil {
			log.Error(err, "Error setting Operator Deployment as owner of UI Deployment")
		}
		foundDeployment := &coreappsv1.Deployment{}
		foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new Deployment for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			foundDeployment, err = coreclient.AppsV1().Deployments(namespace).Create(uiDeployment)
			if err != nil {
				log.Error(err, "Error creating new Deployment for the UI")
			}

			// Deployment created successfully
			log.Info("Deployment for UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Deployment for the UI")
		} else {
			// Deployment already exists - don't requeue
			log.Info("Skip reconcile: Deployment for the UI already exists", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
		}

		// service
		uiService := starterkit.NewServiceForUI(namespace)
		if err := controllerutil.SetControllerReference(operatorDeployment, uiService, mgr.GetScheme()); err != nil {
			log.Error(err, "Error setting Operator Deployment as owner of UI Service")
		}
		foundService := &corev1.Service{}
		foundService, err = coreclient.CoreV1().Services(namespace).Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new Service for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			foundService, err = coreclient.CoreV1().Services(namespace).Create(uiService)
			if err != nil {
				log.Error(err, "Error creating Service for the UI")
			}

			// Service created successfully
			log.Info("Service for the UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Service for the UI")
		} else {
			// Service already exists - don't requeue
			log.Info("Skip reconcile: Service for the UI already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
		}

		// route for UI
		uiRoute := starterkit.NewRouteForUI(namespace)
		if err := controllerutil.SetControllerReference(operatorDeployment, uiRoute, mgr.GetScheme()); err != nil {
			log.Error(err, "Error setting Operator Deployment as owner of UI Route")
		}
		foundRoute := &routev1.Route{}
		foundRoute, err = routev1client.Routes(namespace).Get(starterkit.UIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new Route for the UI", "Namespace", namespace, "Name", starterkit.UIName)
			foundRoute, err = routev1client.Routes(namespace).Create(uiRoute)
			if err != nil {
				log.Error(err, "Error creating Route for the UI")
			}

			// Route created successfully
			log.Info("Route for the UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Route for the UI")
		} else {
			// Route already exists - don't requeue
			log.Info("Skip reconcile: Route for the UI already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
		}

		// route for swagger UI
		uiRoute = starterkit.NewRouteForSwaggerUI(namespace)
		if err := controllerutil.SetControllerReference(operatorDeployment, uiRoute, mgr.GetScheme()); err != nil {
			log.Error(err, "Error setting Operator Deployment as owner of Swagger UI Route")
		}
		foundRoute, err = routev1client.Routes(namespace).Get(starterkit.SwaggerUIName, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new Route for the Swagger UI", "Namespace", namespace, "Name", starterkit.SwaggerUIName)
			foundRoute, err = routev1client.Routes(namespace).Create(uiRoute)
			if err != nil {
				log.Error(err, "Error creating Route for the Swagger UI")
			}

			// Route created successfully
			log.Info("Route for the Swagger UI created successfully")
		} else if err != nil {
			log.Error(err, "Error fetching Route for the Swagger UI")
		} else {
			// Route already exists - don't requeue
			log.Info("Skip reconcile: Route for the Swagger UI already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
		}
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
