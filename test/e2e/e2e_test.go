package e2e_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"
)

var _ = Describe("E2e", func() {

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
})
