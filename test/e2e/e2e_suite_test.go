package e2e_test

import (
	"testing"
	"flag"
	"fmt"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"

	devxv1alpha1 "github.com/ibm/starter-kit-operator/pkg/apis/devx/v1alpha1"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/google/go-github/v32/github"
)

var cfg *rest.Config
var k8sClient client.Client
var k8sManager ctrl.Manager
var ghClient *github.Client
var testEnv *envtest.Environment
var log = logf.Log.WithName("controller_starterkit")

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter(fmt.Sprintf("../TEST-ginkgo-junit_%d.xml", config.GinkgoConfig.ParallelNode))
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	useCluster := true

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		UseExistingCluster:       &useCluster,
		AttachControlPlaneOutput: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// err = devxv1alpha1.AddToSchemes(scheme.Scheme)
	// Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	// make the metrics listen address different for each parallel thread to avoid clashes when running with -p
	var metricsAddr string
	metricsPort := 8090 + config.GinkgoConfig.ParallelNode
	flag.StringVar(&metricsAddr, "metrics-addr", fmt.Sprintf(":%d", metricsPort), "The address the metric endpoint binds to.")
	flag.Parse()
	
	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: metricsAddr,
	})
	Expect(err).ToNot(HaveOccurred())

	// Uncomment the block below to run the operator locally and enable breakpoints / debug during tests
	/*
		err = (&PreScaledCronJobReconciler{
			Client:             k8sManager.GetClient(),
			Log:                ctrl.Log.WithName("controllers").WithName("StarterKit"),
			Recorder:           k8sManager.GetEventRecorderFor("starterkit-controller"),
			InitContainerImage: "initcontainer:1",
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	*/
	  
	err = k8sManager.Start(ctrl.SetupSignalHandler())
	Expect(err).ToNot(HaveOccurred())
	
	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	ghClient = github.NewClient(nil)
	Expect(ghClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
