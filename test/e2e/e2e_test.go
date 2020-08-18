package e2e_test

import (
	"time"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
	skitcv1alpha1 "github.com/ibm/starter-kit-operator/pkg/controller/starterkit"
)

var _ = Describe("E2e", func() {

	const timeout = time.Second * 60
	const interval = time.Second * 1

	secretKeyRef := corev1.SecretKeySelector{
		Key: "apikey",
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "devx-test-secret",
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

			// Expect repo creation
			Expect()
			// Expect 
		})
	
	})
})
