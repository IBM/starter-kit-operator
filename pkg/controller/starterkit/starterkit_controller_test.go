package starterkit

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"

	appsv1 "github.com/openshift/api/apps/v1"
	buildv1 "github.com/openshift/api/build/v1"
	configv1 "github.com/openshift/api/config/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

// TestStarterKitController runs ReconcileStarterKit.Reconcile() against a
// fake client that tracks a StarterKit object.
func TestStarterKitController(t *testing.T) {
	// Set the logger to development mode for verbose logs.
	logf.SetLogger(logf.ZapLogger(true))

	var (
		name            = "starterkit-operator"
		namespace       = "starterkit"
		replicas  int32 = 3
	)

	secretKeyRef := corev1.SecretKeySelector{
		Key: "apikey",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "slack",
		},
	}
	skitOptions := devxv1alpha1.StarterKitSpecOptions{
		Port: 8080,
	}
	templateRepo := devxv1alpha1.StarterKitSpecTemplate{
		TemplateOwner:    "IBM",
		TemplateRepoName: "java-spring-app",
		Owner:            "devx-test",
		Name:             "devx-test-java-spring-app",
		Description:      "DevX Skit Operator Test - Java Spring App",
		SecretKeyRef:     secretKeyRef,
	}

	// A StarterKit resource with metadata and spec.
	starterkit := &devxv1alpha1.StarterKit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: devxv1alpha1.StarterKitSpec{
			Options:      skitOptions,
			TemplateRepo: templateRepo,
		},
	}

	// Register operator types with the runtime scheme.
	kubernetesAPIURL := &configv1.Infrastructure{}
	imgStream := &imagev1.ImageStream{}

	// Objects to track in the fake client.
	objs := []runtime.Object{
		starterkit,
		kubernetesAPIURL,
		imgStream,
	}
	
	s := scheme.Scheme
	s.AddKnownTypes(devxv1alpha1.SchemeGroupVersion, starterkit, kubernetesAPIURL, imgStream)
	
	// Create a fake client to mock API calls.
	cl := fake.NewFakeClient(objs...)
	// Create a ReconcileStarterKit object with the scheme and fake client.
	r := &ReconcileStarterKit{client: cl, scheme: s}

	// Mock request to simulate Reconcile() being called on an event for a
	// watched resource .
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
	res, err := r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	// Check the result of reconciliation to make sure it has the desired state.
	if !res.Requeue {
		t.Error("reconcile did not requeue request as expected")
	}

	// Check for ImageStream
	err = cl.Get(context.TODO(), req.NamespacedName, imgStream)
	if err != nil {
		t.Fatalf("get image stream: (%v)", err)
	}

	// Check for Route
	route := &routev1.Route{}
	err = cl.Get(context.TODO(), req.NamespacedName, route)
	if err != nil {
		t.Fatalf("get route: (%v)", err)
	}

	// Check if Service has been created.
	ser := &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, ser)
	if err != nil {
		t.Fatalf("get service: (%v)", err)
	}

	// Check for Secret
	secret := &corev1.Secret{}
	err = cl.Get(context.TODO(), req.NamespacedName, secret)
	if err != nil {
		t.Fatalf("get secret: (%v)", err)
	}

	// Check for BuildConfig and webhook
	build := &buildv1.BuildConfig{}
	err = cl.Get(context.TODO(), req.NamespacedName, build)
	if err != nil {
		t.Fatalf("get secret: (%v)", err)
	}

	// Check if DeploymentConfig has been created and has the correct size.
	dep := &appsv1.DeploymentConfig{}
	err = cl.Get(context.TODO(), req.NamespacedName, dep)
	if err != nil {
		t.Fatalf("get deployment: (%v)", err)
	}
	dsize := dep.Spec.Replicas
	if dsize != replicas {
		t.Errorf("dep size (%d) is not the expected size (%d)", dsize, replicas)
	}

	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}

	// Check the result of reconciliation to make sure it has the desired state.
	if res.Requeue {
		t.Error("reconcile requeue which is not expected")
	}

	// Reconcile again so Reconcile() checks pods and updates the StarterKit resources' Status.
	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if res != (reconcile.Result{}) {
		t.Error("reconcile did not return an empty Result")
	}

	// Get the updated StarterKit object.
	starterkit = &devxv1alpha1.StarterKit{}
	err = r.client.Get(context.TODO(), req.NamespacedName, starterkit)
	if err != nil {
		t.Errorf("get starterkit: (%v)", err)
	}

	// Ensure Reconcile() updated the starterkit's Status as expected.
	targetRepo := starterkit.Status.TargetRepo
	expectedTargetRepo := "github.com/devx-test/devx-test-java-spring-app"
	if targetRepo != expectedTargetRepo {
		t.Errorf("target repo %v did not match expected %v", targetRepo, expectedTargetRepo)
	}
}
