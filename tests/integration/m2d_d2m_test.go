package integration_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("mirrorToDisk + diskToMirror", func() {
	var workDir string

	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	Describe("mirrorToDisk + diskToMirror happy path", func() {
		iscHappyPath := "isc-happy-path.yaml"

		It("should mirror from remote registry to disk and then from disk to local registry", func() {
			By("running mirrorToDisk")
			result, err := runner.MirrorToDisk(ctx, filepath.Join(iscDir, iscHappyPath), workDir, "--remove-signatures=true")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying images are mirrored in the local cache registry")
			expectSuccessfulMirrorInLocalCache(filepath.Join(iscDir, iscHappyPath), defaultCacheDir())

			By("verifying tar archive contents")
			expectCorrectTarArchiveContents(filepath.Join(iscDir, iscHappyPath), workDir)

			By("running diskToMirror")
			result, err = runner.DiskToMirror(ctx, filepath.Join(iscDir, iscHappyPath), workDir, testRegistry.Endpoint(),
				"--remove-signatures=true", "--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying images are mirrored in the local registry")
			expectSuccessfulMirrorInRegistry(filepath.Join(iscDir, iscHappyPath), *testRegistry)
		})
	})
})
