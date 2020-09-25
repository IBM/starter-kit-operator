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

package controllers

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"github.com/google/go-github/v32/github"
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	devxv1alpha1 "github.com/IBM/starter-kit-operator/api/v1alpha1"
)

var log = logf.Log.WithName("controller_starterkit")

// ReconcileStarterKit reconciles a StarterKit object
type ReconcileStarterKit struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles requests
// +kubebuilder:rbac:groups=devx.my.domain,resources=starterkits,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=devx.my.domain,resources=starterkits/status,verbs=get;update;patch
func (r *ReconcileStarterKit) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	reqLogger := r.Log.WithValues("starterkit", request.NamespacedName)

	// reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling StarterKit")
	ctx := context.Background()
	// Fetch the StarterKit instance
	instance := &devxv1alpha1.StarterKit{}
	err := r.Client.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("StarterKit not found")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error reading StarterKit")
		return ctrl.Result{}, err
	}

	// Fetch public API URL
	reqLogger.Info("Fetching k8s API URL")
	kubernetesAPIURL := &configv1.Infrastructure{}
	infrastructureName := &types.NamespacedName{
		Name: "cluster",
	}
	err = r.Client.Get(ctx, *infrastructureName, kubernetesAPIURL)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Infrastructure not found")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error reading Infrastructure")
		return ctrl.Result{}, err
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
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error fetching GitHub secret")
		return ctrl.Result{}, err
	}

	// Initialize GitHub Client
	client := r.getGitHubClient(githubTokenValue, reqLogger)

	// Read starter kit specification
	reqLogger.Info("Reading StarterKit specification")
	err = r.createTargetGitHubRepo(client, instance, reqLogger)
	if err != nil {
		reqLogger.Error(err, "Error creating target GitHub repo")
		return ctrl.Result{}, err
	}

	// Create ImageStream
	reqLogger.Info("Configuring ImageStream")
	image := newImageStreamForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, image, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting ImageStream on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Image already exists
	foundImage := &imagev1.ImageStream{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: image.Name, Namespace: image.Namespace}, foundImage)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Image", "Image.Namespace", image.Namespace, "Image.Name", image.Name)
		err = r.Client.Create(ctx, image)
		if err != nil {
			reqLogger.Error(err, "Error creating ImageStream")
			return ctrl.Result{}, err
		}

		// Image created successfully
		reqLogger.Info("Image created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching ImageStream")
		return ctrl.Result{}, err
	} else {
		// Image already exists - don't requeue
		reqLogger.Info("Skip reconcile: Image already exists", "Image.Namespace", foundImage.Namespace, "Image.Name", foundImage.Name)
	}

	// Create Route
	reqLogger.Info("Configuring Route")
	route := newRouteForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, route, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Route on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Route already exists
	foundRoute := &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, foundRoute)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.Client.Create(ctx, route)
		if err != nil {
			reqLogger.Error(err, "Error creating Route")
			return ctrl.Result{}, err
		}

		// Route created successfully
		reqLogger.Info("Route created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Route")
		return ctrl.Result{}, err
	} else {
		// Route already exists - don't requeue
		reqLogger.Info("Skip reconcile: Route already exists", "Route.Namespace", foundRoute.Namespace, "Route.Name", foundRoute.Name)
	}

	// Create Service
	reqLogger.Info("Configuring Service")
	service := newServiceForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Service on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Service already exists
	foundService := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(ctx, service)
		if err != nil {
			reqLogger.Error(err, "Error creating Service")
			return ctrl.Result{}, err
		}

		// Service created successfully
		reqLogger.Info("Service created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Service")
		return ctrl.Result{}, err
	} else {
		// Service already exists - don't requeue
		reqLogger.Info("Skip reconcile: Service already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
	}

	// Create Secret
	reqLogger.Info("Configuring CR Secret")
	token, err := GenerateRandomString(32)
	if err != nil {
		reqLogger.Error(err, "Error creating random string")
		return ctrl.Result{}, err
	}
	secret := newSecretForCR(instance, token)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Secret on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Secret already exists
	foundSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(ctx, secret)
		if err != nil {
			reqLogger.Error(err, "Error creating Secret")
			return ctrl.Result{}, err
		}

		// Secret created successfully
		reqLogger.Info("Secret created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Secret")
		return ctrl.Result{}, err
	} else {
		// Secret already exists - don't requeue
		reqLogger.Info("Skip reconcile: Secret already exists", "Secret.Namespace", foundSecret.Namespace, "Secret.Name", foundSecret.Name)
	}

	// Create BuildConfig
	reqLogger.Info("Configuring BuildConfig")
	build := newBuildForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, build, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting BuildConfig on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Build already exists
	foundBuild := &buildv1.BuildConfig{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: build.Name, Namespace: build.Namespace}, foundBuild)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Build", "Build.Namespace", build.Namespace, "Build.Name", build.Name)
		err = r.Client.Create(ctx, build)
		if err != nil {
			reqLogger.Info("Error creating new Build")
			return ctrl.Result{}, err
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
			return ctrl.Result{}, err
		}
		hooks := rc.Get().Namespace(build.Namespace).Resource("buildConfigs").Name(build.Name).SubResource("webhooks")
		githubHook, err := hooks.Suffix(secret.StringData["WebHookSecretKey"], "github").URL(), nil
		if err != nil {
			return ctrl.Result{}, err
		}
		reqLogger.Info("Generated Webhook", "Webhook", githubHook)

		hook := &github.Hook{
			Config: map[string]interface{}{
				"content_type": "json",
				"url":          githubHook.String(),
			},
			Events: []string{"push"},
		}

		createdHook, _, err := client.Repositories.CreateHook(ctx, instance.Spec.TemplateRepo.Owner, instance.Spec.TemplateRepo.Name, hook)
		if err != nil {
			return ctrl.Result{}, err
		}
		reqLogger.Info("Webhook created successfully", "Hook URL", *createdHook.URL)
	} else if err != nil {
		reqLogger.Error(err, "Error fetching Build")
		return ctrl.Result{}, err
	} else {
		// Build already exists - don't requeue
		reqLogger.Info("Skip reconcile: Build already exists", "Build.Namespace", foundBuild.Namespace, "Build.Name", foundBuild.Name)
	}

	// Create Deployment
	reqLogger.Info("Configuring Deployment")
	deployment := newDeploymentForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Deployment on StarterKit")
		return ctrl.Result{}, err
	}

	// Check if this Deployment already exists
	foundDeployment := &appsv1.DeploymentConfig{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Client.Create(ctx, deployment)
		if err != nil {
			reqLogger.Error(err, "Error creating new DeploymentConfig")
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		reqLogger.Info("Deployment created successfully")
	} else if err != nil {
		reqLogger.Error(err, "Error fetching DeploymentConfig")
		return ctrl.Result{}, err
	} else {
		// Deployment already exists - don't requeue
		reqLogger.Info("Skip reconcile: Deployment already exists", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
	}

	// ========================================================================
	// *** handle cleanup of other resources ***

	// Check if the StarterKit instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isMarkedToBeDeleted {
		if contains(instance.GetFinalizers(), starterkitFinalizer) {
			// Run finalization logic for starterkitFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeStarterKit(reqLogger, request, instance, client); err != nil {
				return ctrl.Result{}, err
			}

			// Remove starterkitFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, starterkitFinalizer)
			err := r.Client.Update(ctx, instance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), starterkitFinalizer) {
		reqLogger.Info("Adding finalizer to StarterKit")
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		reqLogger.Info("StarterKit already has finalizer")
	}

	return ctrl.Result{}, nil
}

const starterkitFinalizer = "finalizer.devx.ibm.com"

// Adds the 'finalizeStarterKit' finalizer to the specified StarterKit. The finalizer is responsible for additional cleanup when
// deleting a StarterKit.
func (r *ReconcileStarterKit) addFinalizer(reqLogger logr.Logger, s *devxv1alpha1.StarterKit) error {
	reqLogger.Info("Adding Finalizer for the StarterKit")
	controllerutil.AddFinalizer(s, starterkitFinalizer)

	// Update CR
	err := r.Client.Update(context.TODO(), s)
	if err != nil {
		reqLogger.Error(err, "Failed to update StarterKit with finalizer")
		return err
	}
	return nil
}

// Finalizer that runs during Reconcile() if the StarterKit has been marked for deletion.
// This function performs additional cleanup, namely deleting the created GitHub repo if the DEVX_DEV_MODE environment variable is set to 'true'.
func (r *ReconcileStarterKit) finalizeStarterKit(reqLogger logr.Logger, request ctrl.Request, s *devxv1alpha1.StarterKit, githubClient *github.Client) error {
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

func (r *ReconcileStarterKit) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devxv1alpha1.StarterKit{}).
		Complete(r)
}
