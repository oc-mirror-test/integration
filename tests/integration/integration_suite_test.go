package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/oc-mirror/integration/pkg/ocmirror"
	"github.com/openshift/oc-mirror/integration/pkg/registry"
)

var (
	testRegistry   *registry.Registry
	runner         *ocmirror.Runner
	registryConfig string
	iscDir         string
	keysDir        string
	cacheDir       string
	graphDataDir   string
	ctx            context.Context
	cancel         context.CancelFunc
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "oc-mirror integration tests")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithTimeout(context.Background(), suiteTimeout())

	// Determine paths - when running in container, use ARTIFACTS_DIR
	// When running locally, use relative path to project root
	artifactsDir := os.Getenv("ARTIFACTS_DIR")
	if artifactsDir == "" {
		// Test runs in tests/integration/, go up two levels to project root
		artifactsDir = filepath.Clean("../..")
	}

	registryConfig = filepath.Join(artifactsDir, "registry-config.yaml")
	iscDir = filepath.Join("testdata", "imagesetconfigs")
	graphDataDir = filepath.Join("testdata", "graphdatas")
	keysDir = filepath.Join(artifactsDir, "keys")

	// Setup GPG keys for release signature verification
	os.Setenv("OCP_SIGNATURE_VERIFICATION_PK", filepath.Join(keysDir, "release-pk.asc"))

	// Initialize oc-mirror runner
	binaryPath := os.Getenv("OC_MIRROR_BINARY")
	runner = ocmirror.NewRunner(binaryPath)
})

var _ = BeforeEach(func() {
	// Reset cache dir to default before each test.
	// Individual tests can override this and pass --cache-dir to the runner commands
	cacheDir = defaultCacheDir()

	// Start a fresh registry for each test
	var err error
	testRegistry, err = registry.Start(ctx, registryConfig, 5000, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred(), "Failed to start registry")

	err = testRegistry.WaitReady(ctx, 30*time.Second)
	Expect(err).NotTo(HaveOccurred(), "Registry not ready")
})

var _ = AfterEach(func() {
	// Stop the registry and clean up storage
	if testRegistry != nil {
		if err := testRegistry.Stop(); err != nil {
			GinkgoWriter.Printf("Failed to stop registry: %v\n", err)
		}
	}

	// Clean up the oc-mirror cache
	if err := os.RemoveAll(cacheDir); err != nil {
		GinkgoWriter.Printf("Failed to clean cache dir %s: %v\n", cacheDir, err)
	}
})

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel()
	}
})

func suiteTimeout() time.Duration {
	if v := os.Getenv("TEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 30 * time.Minute
}
