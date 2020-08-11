package starterkit

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/go-logr/logr"
	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

func (r *ReconcileStarterKit) fetchGitHubSecret(skit *devxv1alpha1.StarterKit, request *reconcile.Request, reqLogger logr.Logger) (*string, error) {
	ctx := context.Background()
	githubTokenSecret := &corev1.Secret{}
	secretNamespaceName := &types.NamespacedName{
		Namespace: request.Namespace,
		Name:      skit.Spec.TemplateRepo.SecretKeyRef.Name,
	}
	reqLogger.Info("Fetching GitHub secret")
	err := r.client.Get(ctx, *secretNamespaceName, githubTokenSecret)
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
