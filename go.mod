module github.com/IBM/starter-kit-operator

go 1.13

require (
	github.com/go-logr/logr v0.2.1
	github.com/google/go-github/v32 v32.0.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v0.0.0-20200326152221-912866ddb162
	github.com/openshift/client-go v0.0.0-20200422192633-6f6c07fc2a70
	github.com/prometheus/client_golang v1.5.1 // indirect
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/sys v0.0.0-20200803150936-fd5f0c170ac3 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v0.19.0
	k8s.io/kubernetes v1.13.0
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20200623075207-eb651a5bb0ad

// new paths for APIs and controller
replace github.com/IBM/starter-kit-operator/pkg/apis/devx/v1alpha1 => github.com/IBM/starter-kit-operator/api/v1alpha1 v0.0.2

replace github.com/IBM/starter-kit-operator/pkg/controller/starterkit => github.com/IBM/starter-kit-operator/controllers v0.0.2

replace github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
