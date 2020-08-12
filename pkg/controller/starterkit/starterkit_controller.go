package starterkit

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"github.com/google/go-github/v32/github"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_starterkit")

// Add creates a new StarterKit Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStarterKit{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("starterkit-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource StarterKit
	err = c.Watch(&source.Kind{Type: &devxv1alpha1.StarterKit{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources and requeue the owner StarterKit
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &routev1.Route{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &imagev1.ImageStream{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &buildv1.BuildConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &appsv1.DeploymentConfig{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &devxv1alpha1.StarterKit{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileStarterKit implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileStarterKit{}

// ReconcileStarterKit reconciles a StarterKit object
type ReconcileStarterKit struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

const starterkitFinalizer = "finalizer.devx.ibm.com"

// Reconcile reads that state of the cluster for a StarterKit object and makes changes based on the state read
// and what is in the StarterKit.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStarterKit) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling StarterKit")
	ctx := context.Background()
	// Fetch the StarterKit instance
	instance := &devxv1alpha1.StarterKit{}
	err := r.client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("StarterKit not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error reading StarterKit")
		return reconcile.Result{}, err
	}

	// Fetch public API URL
	reqLogger.Info("Fetching k8s API URL")
	kubernetesAPIURL := &configv1.Infrastructure{}
	infrastructureName := &types.NamespacedName{
		Name: "cluster",
	}
	err = r.client.Get(ctx, *infrastructureName, kubernetesAPIURL)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Infrastructure not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error reading Infrastructure")
		return reconcile.Result{}, err
	}
	kubernetesAPIURLValue := kubernetesAPIURL.Status.APIServerURL
	reqLogger.Info("Found Kubernetes public URL", "kubernetesAPIURL", kubernetesAPIURLValue)

	// Fetch GitHub secret
	githubTokenValue, err := r.fetchGitHubSecret(instance, &request, reqLogger)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("GitHub secret not found", "SecretKeyRef.Name", instance.Spec.TemplateRepo.SecretKeyRef.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error fetching GitHub secret")
		return reconcile.Result{}, err
	}

	// Initialize GitHub Client
	client := r.getGitHubClient(githubTokenValue, reqLogger)

	// Read starter kit specification
	reqLogger.Info("Reading StarterKit specification")
	err = r.createTargetGitHubRepo(client, instance, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error creating target GitHub repo")
		return reconcile.Result{}, err
	}

	// Create ImageStream
	reqLogger.Info("Configuring ImageStream")
	image := newImageStreamForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, image, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting ImageStream on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Image already exists
	foundImage := &imagev1.ImageStream{}
	err = r.client.Get(ctx, types.NamespacedName{Name: image.Name, Namespace: image.Namespace}, foundImage)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Image", "Image.Namespace", image.Namespace, "Image.Name", image.Name)
		err = r.client.Create(ctx, image)
		if err != nil {
			reqLogger.Error(err, "Error creating ImageStream")
			return reconcile.Result{}, err
		}

		// Image created successfully
		reqLogger.Info("Image created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching ImageStream")
		return reconcile.Result{}, err
	} else {
		// Image already exists - don't requeue
		reqLogger.Info("Skip reconcile: Image already exists", "Image.Namespace", foundImage.Namespace, "Image.Name", foundImage.Name)
	}

	// Create Route
	reqLogger.Info("Configuring Route")
	route := newRouteForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, route, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting Route on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Route already exists
	foundRoute := &routev1.Route{}
	err = r.client.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, foundRoute)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.client.Create(ctx, route)
		if err != nil {
			reqLogger.Error(err, "Error creating Route")
			return reconcile.Result{}, err
		}

		// Route created successfully
		reqLogger.Info("Route created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Route")
		return reconcile.Result{}, err
	} else {
		// Route already exists - don't requeue
		reqLogger.Info("Skip reconcile: Route already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
	}

	// Create Service
	reqLogger.Info("Configuring Service")
	service := newServiceForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting Service on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundService := &corev1.Service{}
	err = r.client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(ctx, service)
		if err != nil {
			reqLogger.Error(err, "Error creating Service")
			return reconcile.Result{}, err
		}

		// Service created successfully
		reqLogger.Info("Service created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Service")
		return reconcile.Result{}, err
	} else {
		// Service already exists - don't requeue
		reqLogger.Info("Skip reconcile: Service already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
	}

	// Create Secret
	reqLogger.Info("Configuring CR Secret")
	token, err := GenerateRandomString(32)
	if err != nil {
		reqLogger.Error(err, "Error creating random string")
		return reconcile.Result{}, err
	}
	secret := newSecretForCR(instance, token)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting Secret on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	foundSecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.client.Create(ctx, secret)
		if err != nil {
			reqLogger.Error(err, "Error creating Secret")
			return reconcile.Result{}, err
		}

		// Secret created successfully
		reqLogger.Info("Secret created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Secret")
		return reconcile.Result{}, err
	} else {
		// Secret already exists - don't requeue
		reqLogger.Info("Skip reconcile: Secret already exists", "Secret.Namespace", foundSecret.Namespace, "Secret.Name", foundSecret.Name)
	}

	// Create BuildConfig
	reqLogger.Info("Configuring BuildConfig")
	build := newBuildForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, build, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting BuildConfig on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Build already exists
	foundBuild := &buildv1.BuildConfig{}
	err = r.client.Get(ctx, types.NamespacedName{Name: build.Name, Namespace: build.Namespace}, foundBuild)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Build", "Build.Namespace", build.Namespace, "Build.Name", build.Name)
		err = r.client.Create(ctx, build)
		if err != nil {
			reqLogger.Info("Error creating new Build")
			return reconcile.Result{}, err
		}

		// Build created successfully
		reqLogger.Info("Build created successfully")

		// Create webhook
		cfg := config.GetConfigOrDie()
		cfg.Host = kubernetesAPIURLValue
		cfg.APIPath = "/apis"
		cfg.ContentConfig.GroupVersion = &buildv1.SchemeGroupVersion
		cfg.ContentConfig.NegotiatedSerializer = legacyscheme.Codecs
		rc, err := rest.RESTClientFor(cfg)
		if err != nil {
			return reconcile.Result{}, err
		}
		hooks := rc.Get().Namespace(build.Namespace).Resource("buildConfigs").Name(build.Name).SubResource("webhooks")
		githubHook, err := hooks.Suffix(secret.StringData["WebHookSecretKey"], "github").URL(), nil
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Generated Webhook", "Webhook", githubHook)

		hook := github.Hook{
			Config: map[string]interface{}{
				"content_type": "json",
				"url":          githubHook.String(),
			},
			Events: []string{"push"},
		}

		createdHook, _, err := client.Repositories.CreateHook(ctx, instance.Spec.TemplateRepo.Owner, instance.Spec.TemplateRepo.Name, &hook)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Webhook created successfully", "Hook URL", *createdHook.URL)
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Build")
		return reconcile.Result{}, err
	} else {
		// Build already exists - don't requeue
		reqLogger.Info("Skip reconcile: Build already exists", "Build.Namespace", foundBuild.Namespace, "Build.Name", foundBuild.Name)
	}

	// Create Deployment
	reqLogger.Info("Configuring Deployment")
	deployment := newDeploymentForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
		reqLogger.Error(err, "Error setting Deployment on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundDeployment := &appsv1.DeploymentConfig{}
	err = r.client.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.client.Create(ctx, deployment)
		if err != nil {
			reqLogger.Info("Error creating new DeploymentConfig")
			return reconcile.Result{}, err
		}

		// Deployment created successfully
		reqLogger.Info("Deployment created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching DeploymentConfig")
		return reconcile.Result{}, err
	} else {
		// Deployment already exists - don't requeue
		reqLogger.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
	}

	// Check if the StarterKit instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {
		if contains(instance.GetFinalizers(), starterkitFinalizer) {
			// Run finalization logic for starterkitFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeStarterKit(reqLogger, request, instance, client); err != nil {
				return reconcile.Result{}, err
			}

			// Remove starterkitFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, starterkitFinalizer)
			err := r.client.Update(ctx, instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), starterkitFinalizer) {
		reqLogger.Info("Adding finalizer to StarterKit")
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		reqLogger.Info("StarterKit already has finalizer")
	}

	return reconcile.Result{}, nil
}

// Adds the 'finalizeStarterKit' finalizer to the specified StarterKit. The finalizer is responsible for additional cleanup when
// deleting a StarterKit.
func (r *ReconcileStarterKit) addFinalizer(reqLogger logr.Logger, s *devxv1alpha1.StarterKit) error {
	reqLogger.Info("Adding Finalizer for the StarterKit")
	controllerutil.AddFinalizer(s, starterkitFinalizer)

	// Update CR
	err := r.client.Update(context.TODO(), s)
	if err != nil {
		reqLogger.Error(err, "Failed to update StarterKit with finalizer")
		return err
	}
	return nil
}

// Finalizer that runs during Reconcile() if the StarterKit has been marked for deletion.
// This function performs additional cleanup, namely deleting the created GitHub repo if the DEVX_DEV_MODE environment variable is set to 'true'.
func (r *ReconcileStarterKit) finalizeStarterKit(reqLogger logr.Logger, request reconcile.Request, s *devxv1alpha1.StarterKit, githubClient *github.Client) error {
	// if we're running in development mode, cleanup the github repo if present
	ctx := context.Background()
	if devxDevMode, ok := os.LookupEnv("DEVX_DEV_MODE"); ok {
		if devxDevMode == "true" {
			reqLogger.Info("Running in development mode")
			reqLogger.Info("Deleting target GitHub repo", "TargetRepo", s.Status.TargetRepo)
			// note that this requires the GitHub access token to have admin or delete_repo rights
			_, err := githubClient.Repositories.Delete(ctx, s.Spec.TemplateRepo.Owner, s.Spec.TemplateRepo.Name)
			if err != nil {
				return err
			}
		}
	}
	reqLogger.Info("Successfully finalized StarterKit")
	return nil
}
