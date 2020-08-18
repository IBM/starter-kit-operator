package e2e_test

import (
	"time"
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "github.com/openshift/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	buildv1 "github.com/openshift/api/build/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
)

var _ = Describe("E2e", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1
	const owner = "arftt"
	const repoName = owner + "-java-spring-app"

	var (
		name            = "starterkit-operator"
		namespace       = "e2e-testing"
	)

	secretKeyRef := corev1.SecretKeySelector{
		Key: "arftt-gh-token",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "arftt-secrets",
		},
	}
	skitOptions := devxv1alpha1.StarterKitSpecOptions{
		Port: 8080,
	}
	templateRepo := devxv1alpha1.StarterKitSpecTemplate{
		TemplateOwner:    "IBM",
		TemplateRepoName: "java-spring-app",
		Owner:            "arftt",
		Name:             "arftt-java-spring-app",
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
	fetchedSkit := &devxv1alpha1.StarterKit{}

	Context("with the new project layout", func() {

		ctx := context.Background()

		BeforeEach(func() {
			By("creating a new test context")
		})

		AfterEach(func() {
			By("deleting generated skits")
		})

		It("should create a new skit", func() {
			By("running create")
			Expect(k8sClient.Create(ctx, starterkit)).Should(Succeed())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, fetchedSkit)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// expect StarterKit
			Expect(fetchedSkit.Name).To(Equal(name))
			Expect(fetchedSkit.Namespace).To(Equal(namespace))
			Expect(fetchedSkit.Spec.TemplateRepo.Owner).To(Equal(owner))
			Expect(fetchedSkit.Spec.TemplateRepo.Name).To(Equal(repoName))

			// expect ImageStream
			imgStream := &imagev1.ImageStream{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, imgStream)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(imgStream.Name).To(Equal(name))
			Expect(imgStream.Namespace).To(Equal(namespace))

			// expect Route
			route := &routev1.Route{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, route)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(route.Name).To(Equal(name))
			Expect(route.Namespace).To(Equal(namespace))
			Expect(route.Spec.To.Name).To(Equal(name))

			// expect Service
			ser := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, ser)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(ser.Name).To(Equal(name))
			Expect(ser.Namespace).To(Equal(namespace))

			// expect Secret for CR
			secret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(secret.Name).To(Equal(fetchedSkit.Spec.TemplateRepo.SecretKeyRef.Name))
			Expect(secret.Namespace).To(Equal(namespace))

			// expect BuildConfig
			build := &buildv1.BuildConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, build)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(build.Name).To(Equal(name))
			Expect(build.Namespace).To(Equal(namespace))

			// Expect repo creation
			// Expect(ghClient.Repositories.List(ctx, "arftt", nil)

			// Expect Deployment
			dep := &appsv1.DeploymentConfig{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name:starterkit.Name, Namespace: namespace}, dep)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(dep.Name).To(Equal(name))
			Expect(dep.Namespace).To(Equal(namespace))
		})
	})
})
