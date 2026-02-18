package integration_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("mirrorToMirror", func() {
	var workDir string
	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	Describe("mirrorToMirror happy path", func() {
		It("should mirror from remote registry to a local registry", func() {
			By("running mirrorToMirror")
			result, err := runner.MirrorToMirror(ctx, filepath.Join(iscDir, "isc-happy-path.yaml"), workDir, testRegistry.Endpoint(),
				"--remove-signatures=true", "--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying images are mirrored in the local cache registry")
			expectSuccessfulMirrorInLocalCache(filepath.Join(iscDir, "isc-happy-path.yaml"), defaultCacheDir())

			By("verifying images are mirrored in the local registry")
			expectSuccessfulMirrorInRegistry(filepath.Join(iscDir, "isc-happy-path.yaml"), *testRegistry)
		})
	})
})
