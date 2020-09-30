package controllers

import (
	devxv1alpha1 "github.com/IBM/starter-kit-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("StarterKit controller", func() {
	Context("When creating a StarterKit", func() {
		It("Should update the StarterKit status", func() {
			secretKeyRef := corev1.SecretKeySelector{
				Key: "apikey",
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "gee4vee-gh-token", // TODO change this to a functional ID later
				},
			}
			skitOptions := devxv1alpha1.StarterKitSpecOptions{
				Port: 8080,
				Env: []corev1.EnvVar{
					corev1.EnvVar{
						Name:  "test",
						Value: "val",
					},
				},
			}
			templateRepo := devxv1alpha1.StarterKitSpecTemplate{
				TemplateOwner:    "IBM",
				TemplateRepoName: "java-spring-app",
				Owner:            "gee4vee", // TODO change this to a functional ID later
				Name:             "devx-test-java-spring-app",
				Description:      "DevX Skit Operator Test - Java Spring App",
				SecretKeyRef:     secretKeyRef,
			}
			starterkit := &devxv1alpha1.StarterKit{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "devx.ibm.com/v1alpha1",
					Kind:       "StarterKit",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      StarterKitName,
					Namespace: StarterKitNamespace,
				},
				Spec: devxv1alpha1.StarterKitSpec{
					Options:      skitOptions,
					TemplateRepo: templateRepo,
				},
			}
			Expect(k8sClient.Create(ctx, starterkit)).Should(Succeed())

			skitLookupKey := types.NamespacedName{Name: StarterKitName, Namespace: StarterKitNamespace}
			createdSkit := &devxv1alpha1.StarterKit{}

			// We'll need to retry getting this newly created starter kit, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, skitLookupKey, createdSkit)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// target repo URL should be set
			Expect(createdSkit.Status.TargetRepo).Should(Equal("github.com/devx-test/devx-test-java-spring-app"))
		})
	})
})
