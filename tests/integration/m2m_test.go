package integration_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

			By("verifying working-dir structure")
			expectWorkingDirStructure(workDir, []string{
				"hold-release", "release-images", "operator-catalogs", "helm", "cluster-resources", "signatures", "logs",
			})

			By("verifying images are in the local registry")
			repos, err := testRegistry.ListRepositories()
			Expect(err).NotTo(HaveOccurred())
			expectRepositoriesExist(repos, []string{
				"openshifttest/hello-openshift", "openshift/release", "stefanprodan/podinfo",
			})
		})
	})

})
