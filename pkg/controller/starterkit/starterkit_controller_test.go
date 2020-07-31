package starterkit

import (
	"context"
	"crypto/rand"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "github.com/openshift/api/apps/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("controller_starterkit_test")

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

	// A StarterKit resource with metadata and spec.
	starterkit := &cachev1alpha1.StarterKit{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cachev1alpha1.StarterKitSpec{
			Size: replicas, // Set desired number of StarterKit replicas.
		},
	}
	// Objects to track in the fake client.
	objs := []runtime.Object{
		starterkit,
	}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(cachev1alpha1.SchemeGroupVersion, starterkit)
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

	// Check if Deployment has been created and has the correct size.
	dep := &appsv1.Deployment{}
	err = cl.Get(context.TODO(), req.NamespacedName, dep)
	if err != nil {
		t.Fatalf("get deployment: (%v)", err)
	}
	dsize := *dep.Spec.Replicas
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

	// Check if Service has been created.
	ser := &corev1.Service{}
	err = cl.Get(context.TODO(), req.NamespacedName, ser)
	if err != nil {
		t.Fatalf("get service: (%v)", err)
	}

	// Create the 3 expected pods in namespace and collect their names to check
	// later.
	podLabels := labelsForMemcached(name)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Labels:    podLabels,
		},
	}
	podNames := make([]string, 3)
	for i := 0; i < 3; i++ {
		pod.ObjectMeta.Name = name + ".pod." + strconv.Itoa(rand.Int())
		podNames[i] = pod.ObjectMeta.Name
		if err = cl.Create(context.TODO(), pod.DeepCopy()); err != nil {
			t.Fatalf("create pod %d: (%v)", i, err)
		}
	}

	// Reconcile again so Reconcile() checks pods and updates the StarterKit
	// resources' Status.
	res, err = r.Reconcile(req)
	if err != nil {
		t.Fatalf("reconcile: (%v)", err)
	}
	if res != (reconcile.Result{}) {
		t.Error("reconcile did not return an empty Result")
	}

	// Get the updated StarterKit object.
	starterkit = &cachev1alpha1.StarterKit{}
	err = r.client.Get(context.TODO(), req.NamespacedName, starterkit)
	if err != nil {
		t.Errorf("get starterkit: (%v)", err)
	}

	// Ensure Reconcile() updated the starterkit's Status as expected.
	nodes := starterkit.Status.Nodes
	if !reflect.DeepEqual(podNames, nodes) {
		t.Errorf("pod names %v did not match expected %v", nodes, podNames)
	}
}
