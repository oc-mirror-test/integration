package integration_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

		It("should mirror from remote registry to to disk and then from disk to local registry", func() {
			By("running mirrorToDisk")
			result, err := runner.MirrorToDisk(ctx, filepath.Join(iscDir, iscHappyPath), workDir, "--remove-signatures=true")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying working-dir structure")
			expectWorkingDirStructure(workDir, []string{
				"hold-release", "release-images", "operator-catalogs", "helm", "cluster-resources", "signatures", "logs",
			})

			By("verifying tar archive was created")
			expectTarArchiveExists(workDir)

			By("running diskToMirror")
			result, err = runner.DiskToMirror(ctx, filepath.Join(iscDir, iscHappyPath), workDir, testRegistry.Endpoint(),
				"--remove-signatures=true", "--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying images are in the local registry")
			repos, err := testRegistry.ListRepositories()
			Expect(err).NotTo(HaveOccurred())
			expectRepositoriesExist(repos, []string{
				"openshifttest/hello-openshift", "openshift/release", "stefanprodan/podinfo",
			})
		})
	})
})
