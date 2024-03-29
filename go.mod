module github.com/ibm/starter-kit-operator

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	github.com/google/go-github/v39 v39.2.0
	github.com/miekg/dns v1.1.43 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/openshift/api v0.0.0-20200623075207-eb651a5bb0ad
	github.com/openshift/client-go v0.0.0-20200422192633-6f6c07fc2a70
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
)
