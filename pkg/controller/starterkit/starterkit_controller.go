package starterkit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"os"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/api/legacyscheme"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"

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
	reqLogger.Info("Reconciling Starte=rKit")
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
		reqLogger.Info("StarterKit error")
		return reconcile.Result{}, err
	}

	// Fetch public API URL
	reqLogger.Info("Fetching k8s API URL")
	kubernetesApiURL := &configv1.Infrastructure{}
	infrastructureName := &types.NamespacedName{
		Name: "cluster",
	}
	err = r.client.Get(ctx, *infrastructureName, kubernetesApiURL)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Infrastructure not found")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("Infrastructure error")
		return reconcile.Result{}, err
	}
	kubernetesApiURLValue := kubernetesApiURL.Status.APIServerURL
	reqLogger.Info("Found Kubernetes public URL", "kubernetesApiURL", kubernetesApiURLValue)

	// Fetch GitHub secret
	reqLogger.Info("Fetching GitHub secret")
	githubTokenSecret := &corev1.Secret{}
	secretNamespaceName := &types.NamespacedName{
		Namespace: request.Namespace,
		Name:      instance.Spec.TemplateRepo.SecretKeyRef.Name,
	}
	err = r.client.Get(ctx, *secretNamespaceName, githubTokenSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	githubTokenValue := string(githubTokenSecret.Data[instance.Spec.TemplateRepo.SecretKeyRef.Key])

	// Initialize GitHub Client
	reqLogger.Info("Initializing GitHub client")
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubTokenValue},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Read starter kit specification
	reqLogger.Info("Reading starter kit specification")
	if instance.Status.TargetRepo == "" {
		// Create a repo
		req := github.TemplateRepoRequest{
			Name:        &instance.Spec.TemplateRepo.Name,
			Owner:       &instance.Spec.TemplateRepo.Owner,
			Description: &instance.Spec.TemplateRepo.Description,
		}

		createdRepo, _, err := client.Repositories.CreateFromTemplate(ctx, instance.Spec.TemplateRepo.TemplateOwner, instance.Spec.TemplateRepo.TemplateRepoName, &req)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Repo created successfully", "GitHub URL", *createdRepo.HTMLURL)

		// Set the TargetRepo to the repo created
		instance.Status.TargetRepo = *createdRepo.HTMLURL

		if err := r.client.Status().Update(ctx, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Create ImageStream
	reqLogger.Info("Configuring ImageStream")
	image := newImageStreamForCR(instance)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, image, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Image already exists
	foundImage := &imagev1.ImageStream{}
	err = r.client.Get(ctx, types.NamespacedName{Name: image.Name, Namespace: image.Namespace}, foundImage)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Image", "Image.Namespace", image.Namespace, "Image.Name", image.Name)
		err = r.client.Create(ctx, image)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Image created successfully
		reqLogger.Info("Image created successfully")
	} else if err != nil {
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
		return reconcile.Result{}, err
	}

	// Check if this Route already exists
	foundRoute := &routev1.Route{}
	err = r.client.Get(ctx, types.NamespacedName{Name: route.Name, Namespace: route.Namespace}, foundRoute)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Route", "Route.Namespace", route.Namespace, "Route.Name", route.Name)
		err = r.client.Create(ctx, route)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Route created successfully
		reqLogger.Info("Route created successfully")
	} else if err != nil {
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
		return reconcile.Result{}, err
	}

	// Check if this Service already exists
	foundService := &corev1.Service{}
	err = r.client.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)
		err = r.client.Create(ctx, service)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Service created successfully
		reqLogger.Info("Service created successfully")
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// Service already exists - don't requeue
		reqLogger.Info("Skip reconcile: Service already exists", "Service.Namespace", foundService.Namespace, "Service.Name", foundService.Name)
	}

	// Create Secret
	reqLogger.Info("Configuring CR Secret")
	token, err := GenerateRandomString(32)
	if err != nil {
		return reconcile.Result{}, err
	}
	secret := newSecretForCR(instance, token)

	// Set StarterKit instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Secret already exists
	foundSecret := &corev1.Secret{}
	err = r.client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
		err = r.client.Create(ctx, secret)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Secret created successfully
		reqLogger.Info("Secret created successfully")
	} else if err != nil {
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
		return reconcile.Result{}, err
	}

	// Check if this Build already exists
	foundBuild := &buildv1.BuildConfig{}

	err = r.client.Get(ctx, types.NamespacedName{Name: build.Name, Namespace: build.Namespace}, foundBuild)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Build", "Build.Namespace", build.Namespace, "Build.Name", build.Name)
		err = r.client.Create(ctx, build)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Build created successfully
		reqLogger.Info("Build created successfully")

		// Create webhook
		cfg := config.GetConfigOrDie()
		cfg.Host = kubernetesApiURLValue
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
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	foundDeployment := &appsv1.DeploymentConfig{}
	err = r.client.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.client.Create(ctx, deployment)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Deployment created successfully
		reqLogger.Info("Deployment created successfully")
	} else if err != nil {
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
            if err := r.finalizeStarterKit(reqLogger, instance); err != nil {
                return reconcile.Result{}, err
            }

            // Remove starterkitFinalizer. Once all finalizers have been
            // removed, the object will be deleted.
            controllerutil.RemoveFinalizer(instance, starterkitFinalizer)
            err := r.Update(ctx, instance)
            if err != nil {
                return reconcile.Result{}, err
            }
        }
        return reconcile.Result{}, nil
    }

    // Add finalizer for this CR
    if !contains(instance.GetFinalizers(), starterkitFinalizer) {
		reqLogger.Info("Adding finalizer to starterkit")
        if err := r.addFinalizer(reqLogger, instance); err != nil {
            return reconcile.Result{}, err
        }
    } else {
		reqLogger.Info("Starterkit already has finalizer")
    }

	return reconcile.Result{}, nil
}

func contains(list []string, s string) bool {
    for _, v := range list {
        if v == s {
            return true
        }
    }
    return false
}

func (r *ReconcileStarterKit) addFinalizer(reqLogger logf.Logger, s *devxv1alpha1.StarterKit) error {
    reqLogger.Info("Adding Finalizer for the StarterKit")
    controllerutil.AddFinalizer(s, starterkitFinalizer)

    // Update CR
    err := r.Update(context.TODO(), s)
    if err != nil {
        reqLogger.Error(err, "Failed to update StarterKit with finalizer")
        return err
    }
    return nil
}

func (r *ReconcileStarterKit) finalizeStarterKit(reqLogger logf.Logger, s *devxv1alpha1.StarterKit) error {
	// if we're running in development mode, cleanup the github repo if present
	if devxDevMode, ok := os.LookupEnv("DEVX_DEV_MODE"); ok {
		if devxDevMode == "true" {
			reqLogger.Info("Running in development mode")
		}
	}
    reqLogger.Info("Successfully finalized starterkit")
    return nil
}

// Create a new Secret
func newSecretForCR(cr *devxv1alpha1.StarterKit, token string) *corev1.Secret {
	labels := map[string]string{
		"app": cr.Name,
	}
	stringData := map[string]string{
		"WebHookSecretKey": token,
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "k8s.io/api/core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		StringData: stringData,
	}
}

// Create a new Service
func newServiceForCR(cr *devxv1alpha1.StarterKit) *corev1.Service {
	labels := map[string]string{
		"app": cr.Name,
	}
	selector := map[string]string{
		"name": cr.Name,
	}
	port := int32(3000)
	if cr.Spec.Options.Port > 0 {
		port = cr.Spec.Options.Port
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "k8s.io/api/core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
				},
			},
			Selector: selector,
		},
	}
}

// Create a new Route
func newRouteForCR(cr *devxv1alpha1.StarterKit) *routev1.Route {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "github.com/openshift/api/route/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: cr.Name,
			},
		},
	}
}

// Create a new ImageStream
func newImageStreamForCR(cr *devxv1alpha1.StarterKit) *imagev1.ImageStream {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ImageStream",
			APIVersion: "github.com/openshift/api/image/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
	}
}

// Create a new BuildConfig
func newBuildForCR(cr *devxv1alpha1.StarterKit) *buildv1.BuildConfig {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &buildv1.BuildConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "BuildConfig",
			APIVersion: "build.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: buildv1.BuildConfigSpec{
			CommonSpec: buildv1.CommonSpec{
				Source: buildv1.BuildSource{
					Git: &buildv1.GitBuildSource{
						URI: cr.Status.TargetRepo,
						Ref: "master",
					},
				},
				Strategy: buildv1.BuildStrategy{
					Type: buildv1.SourceBuildStrategyType,
					DockerStrategy: &buildv1.DockerBuildStrategy{
						DockerfilePath: "Dockerfile",
					},
				},
				Output: buildv1.BuildOutput{
					To: &corev1.ObjectReference{
						Kind: "ImageStreamTag",
						Name: cr.Name + ":latest",
					},
				},
			},
			Triggers: []buildv1.BuildTriggerPolicy{
				{
					Type: buildv1.ImageChangeBuildTriggerType,
				},
				{
					Type: buildv1.ConfigChangeBuildTriggerType,
				},
				{
					Type: buildv1.GitHubWebHookBuildTriggerType,
					GitHubWebHook: &buildv1.WebHookTrigger{
						SecretReference: &buildv1.SecretLocalReference{
							Name: cr.Name,
						},
					},
				},
			},
		},
	}
}

// Create a new Deployment
func newDeploymentForCR(cr *devxv1alpha1.StarterKit) *appsv1.DeploymentConfig {
	labels := map[string]string{
		"app":  cr.Name,
		"name": cr.Name,
	}
	selector := map[string]string{
		"app":  cr.Name,
		"name": cr.Name,
	}
	annotations := map[string]string{
		"app.openshift.io/vcs-uri": cr.Status.TargetRepo,
	}
	port := int32(3000)
	if cr.Spec.Options.Port > 0 {
		port = cr.Spec.Options.Port
	}
	env := cr.Spec.Options.Env

	return &appsv1.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentConfig",
			APIVersion: "github.com/openshift/api/apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Labels:      labels,
		},
		Spec: appsv1.DeploymentConfigSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.DeploymentStrategyTypeRolling,
			},
			Triggers: appsv1.DeploymentTriggerPolicies{
				{
					Type: appsv1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &appsv1.DeploymentTriggerImageChangeParams{
						Automatic:      true,
						ContainerNames: []string{cr.Name},
						From: corev1.ObjectReference{
							Kind: "ImageStreamTag",
							Name: cr.Name + ":latest",
						},
					},
				},
				{
					Type: appsv1.DeploymentTriggerOnConfigChange,
				},
			},
			Replicas: 1,
			Selector: selector,
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  cr.Name,
							Image: cr.Name,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(port),
								},
							},
							Env: env,
						},
					},
				},
			},
		},
	}
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}
