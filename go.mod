module github.com/IBM/starter-kit-operator

go 1.13

require (
	github.com/go-logr/logr v0.2.0
	github.com/go-openapi/spec v0.19.4
	github.com/google/go-github/v32 v32.0.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v0.0.0-20200326152221-912866ddb162
	github.com/openshift/client-go v0.0.0-20200422192633-6f6c07fc2a70
	github.com/operator-framework/operator-sdk v1.0.1
	github.com/spf13/pflag v1.0.5
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/sys v0.0.0-20200803150936-fd5f0c170ac3 // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20200623075207-eb651a5bb0ad
