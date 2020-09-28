package controllers

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/go-logr/logr"
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	consolev1 "github.com/openshift/api/console/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	devxv1alpha1 "github.com/IBM/starter-kit-operator/api/v1alpha1"
	coreappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/google/go-github/v32/github"
)

// Returns the GitHub secret defined in the specified StarterKit.
func (r *ReconcileStarterKit) fetchGitHubSecret(skit *devxv1alpha1.StarterKit, request *reconcile.Request, reqLogger logr.Logger) (*string, error) {
	ctx := context.Background()
	githubTokenSecret := &corev1.Secret{}
	secretNamespaceName := &types.NamespacedName{
		Namespace: request.Namespace,
		Name:      skit.Spec.TemplateRepo.SecretKeyRef.Name,
	}
	reqLogger.Info("Fetching GitHub secret")
	err := r.Client.Get(ctx, *secretNamespaceName, githubTokenSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("GitHub secret not found", "SecretKeyRef.Name", skit.Spec.TemplateRepo.SecretKeyRef.Name)
			return nil, err
		}
		// Error reading the object - requeue the request.
		reqLogger.Info("GitHub secret error")
		return nil, err
	}

	githubTokenValue := string(githubTokenSecret.Data[skit.Spec.TemplateRepo.SecretKeyRef.Key])
	return &githubTokenValue, nil
}

// Returns a GitHub Client that can be used to make GitHub API calls.
func (r *ReconcileStarterKit) getGitHubClient(githubTokenValue *string, reqLogger logr.Logger) *github.Client {
	reqLogger.Info("Initializing GitHub client")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *githubTokenValue},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}

// Creates and sets the target GitHub repo defined in the specified StarterKit if it has not been previously created and set on the StarterKit.
func (r *ReconcileStarterKit) createTargetGitHubRepo(client *github.Client, skit *devxv1alpha1.StarterKit, reqLogger logr.Logger) error {
	ctx := context.Background()
	if skit.Status.TargetRepo == "" {
		// Create a repo
		req := github.TemplateRepoRequest{
			Name:        &skit.Spec.TemplateRepo.Name,
			Owner:       &skit.Spec.TemplateRepo.Owner,
			Description: &skit.Spec.TemplateRepo.Description,
		}

		createdRepo, _, err := client.Repositories.CreateFromTemplate(ctx, skit.Spec.TemplateRepo.TemplateOwner, skit.Spec.TemplateRepo.TemplateRepoName, &req)
		if err != nil {
			return err
		}
		reqLogger.Info("Repo created successfully", "GitHub URL", *createdRepo.HTMLURL)

		// Set the TargetRepo to the repo created
		skit.Status.TargetRepo = *createdRepo.HTMLURL

		if err := r.Client.Status().Update(ctx, skit); err != nil {
			return err
		}

		return nil
	}
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
		"app":  cr.Name,
		"devx": "",
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
		"app":  cr.Name,
		"devx": "",
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
		"app":  cr.Name,
		"devx": "",
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
		"app":  cr.Name,
		"devx": "",
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
		"devx": "",
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
	// needed to install additional resources at operator install time
	namespaceEnv := corev1.EnvVar{
		Name: "WATCH_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.annotations['olm.targetNamespaces']",
			},
		},
	}
	env = append(env, namespaceEnv)

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

// ========================================================================
// UI resources

// UIName is the name of most main UI resources
const UIName = "starter-kit-operator-ui"

// SwaggerUIName is the name of Swagger UI resources
const SwaggerUIName = "starter-kit-operator-swagger-ui"

// DefaultUIImageAccount the Docker Hub account hosting the UI image. TODO this needs to live in an IBM account eventually
const DefaultUIImageAccount = "ibmcom"

// DefaultUIImageVersion The version of the UI image to use
const DefaultUIImageVersion = "0.1.0"

// DockerRegistryURL The URL of the Docker Hub registry accessible within the operator deployment
const DockerRegistryURL = "docker-registry.default.svc:5000/"
const uiPort = int32(5000)

// NewDeploymentForUI returns a new DeploymentConfig for the skit operator UI
func NewDeploymentForUI(namespace string, imageAccount string, imageVersion string) *coreappsv1.Deployment {
	var uiImage = imageAccount + "/" + UIName + ":" + imageVersion
	labels := map[string]string{
		"app":                       UIName,
		"name":                      UIName,
		"devx":                      "",
		"app.kubernetes.io/part-of": "starter-kit-operator",
	}
	selector := map[string]string{
		"app":  UIName,
		"name": UIName,
	}
	numReplicas := int32(1)
	return &coreappsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "k8s.io/api/apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: coreappsv1.DeploymentSpec{
			Strategy: coreappsv1.DeploymentStrategy{
				Type: coreappsv1.RollingUpdateDeploymentStrategyType,
			},
			Replicas: &numReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  UIName,
							Image: uiImage,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: uiPort,
									Name:          "ui",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/api/v1/health",
										Port:   intstr.FromInt(int(uiPort)),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// NewServiceForUI returns a new Service for the skit operator UI
func NewServiceForUI(namespace string) *corev1.Service {
	labels := map[string]string{
		"app":  UIName,
		"devx": "",
	}
	selector := map[string]string{
		"name": UIName,
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "k8s.io/api/core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "ui",
					Port:       uiPort,
					TargetPort: intstr.FromInt(int(uiPort)),
				},
			},
			Selector: selector,
		},
	}
}

// NewRouteForUI returns a new Route for the skit operator UI
func NewRouteForUI(namespace string) *routev1.Route {
	labels := map[string]string{
		"app":  UIName,
		"devx": "",
	}

	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "github.com/openshift/api/route/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: UIName,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt(int(uiPort)),
			},
			TLS: &routev1.TLSConfig{
				Termination:                   routev1.TLSTerminationEdge,
				InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyRedirect,
			},
		},
	}
}

// NewConsoleLinkForUI returns a new ConsoleLink for the starter kit operator UI
func NewConsoleLinkForUI(namespace string, href string) *consolev1.ConsoleLink {
	labels := map[string]string{
		"app":  UIName,
		"devx": "",
	}

	return &consolev1.ConsoleLink{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConsoleLink",
			APIVersion: "github.com/openshift/api/console/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      UIName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: consolev1.ConsoleLinkSpec{
			Link: consolev1.Link{
				Href: href,
				Text: "Starter Kit Operator UI",
			},
			Location: consolev1.ApplicationMenu,
			ApplicationMenu: &consolev1.ApplicationMenuSpec{
				Section:  "App Development",
				ImageURL: "https://cloud.ibm.com/favicon.ico",
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

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
