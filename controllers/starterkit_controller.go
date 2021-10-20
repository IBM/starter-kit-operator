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

package controllers

import (
	"context"
	"os"

	"github.com/google/go-github/v39/github"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/api/v1alpha1"
)

// StarterKit reconciles a StarterKit object
type StarterKitReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const starterkitFinalizer = "finalizer.devx.ibm.com"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *StarterKitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("starterkit", req.NamespacedName)
	reqLogger.Info("Reconciling StarterKit")

	// Fetch the StarterKit instance
	instance := &devxv1alpha1.StarterKit{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
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
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Error reading Infrastructure")
		return reconcile.Result{}, err
	}
	kubernetesAPIURLValue := kubernetesAPIURL.Status.APIServerURL
	reqLogger.Info("Found Kubernetes public URL", "kubernetesAPIURL", kubernetesAPIURLValue)

	// Fetch GitHub secret
	githubTokenValue, err := r.fetchGitHubSecret(instance, &req, reqLogger)
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
	if err := controllerutil.SetControllerReference(instance, image, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting ImageStream on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Image already exists
	foundImage := &imagev1.ImageStream{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: image.Name, Namespace: image.Namespace}, foundImage)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Image", "Image.Namespace", image.Namespace, "Image.Name", image.Name)
		err = r.Client.Create(ctx, image)
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
	if err := controllerutil.SetControllerReference(instance, route, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Route on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Route already exists
	foundRoute := &routev1.Route{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, foundRoute)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.Client.Create(ctx, route)
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
	if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Service on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundService := &corev1.Service{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.Client.Create(ctx, service)
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
	if err := controllerutil.SetControllerReference(instance, secret, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Secret on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	foundSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.Client.Create(ctx, secret)
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
	if err := controllerutil.SetControllerReference(instance, build, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting BuildConfig on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Build already exists
	foundBuild := &buildv1.BuildConfig{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: build.Name, Namespace: build.Namespace}, foundBuild)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Build", "Build.Namespace", build.Namespace, "Build.Name", build.Name)
		err = r.Client.Create(ctx, build)
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
		cfg.ContentConfig.NegotiatedSerializer = &serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
		rc, err := rest.RESTClientFor(cfg)
		if err != nil {
			return reconcile.Result{}, err
		}
		hooks := rc.Get().Namespace(build.Namespace).Resource("buildConfigs").Name(build.Name).SubResource("webhooks")
		githubHook, err := hooks.Suffix(string(secret.Data["WebHookSecretKey"]), "github").URL(), nil
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
	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		reqLogger.Error(err, "Error setting Deployment on StarterKit")
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundDeployment := &appsv1.DeploymentConfig{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Client.Create(ctx, deployment)
		if err != nil {
			reqLogger.Error(err, "Error creating new DeploymentConfig")
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
			if err := r.finalizeStarterKit(reqLogger, req, instance, client); err != nil {
				return reconcile.Result{}, err
			}

			// Remove starterkitFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(instance, starterkitFinalizer)
			err := r.Client.Update(ctx, instance)
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

	return ctrl.Result{}, nil
}

// Adds the 'finalizeStarterKit' finalizer to the specified StarterKit. The finalizer is responsible for additional cleanup when
// deleting a StarterKit.
func (r *StarterKitReconciler) addFinalizer(reqLogger logr.Logger, s *devxv1alpha1.StarterKit) error {
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
func (r *StarterKitReconciler) finalizeStarterKit(reqLogger logr.Logger, request reconcile.Request, s *devxv1alpha1.StarterKit, githubClient *github.Client) error {
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

// SetupWithManager sets up the controller with the Manager.
func (r *StarterKitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devxv1alpha1.StarterKit{}).
		Complete(r)
}
