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
		iscHappyPath := "isc-happy-path.yaml"
		discHappyPath := "disc-happy-path.yaml"
		deleteId := "delete-test"

		It("should mirror from remote registry to a local registry", func() {
			deleteYaml := filepath.Join(workDir, "working-dir", "delete", "delete-images-"+deleteId+".yaml")

			By("running mirrorToMirror")
			result, err := runner.MirrorToMirror(ctx, filepath.Join(iscDir, iscHappyPath), workDir, testRegistry.Endpoint(),
				"--remove-signatures=true", "--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying images are mirrored in the local registry")
			expectSuccessfulMirrorInRegistry(filepath.Join(iscDir, iscHappyPath), *testRegistry)

			By("running delete workflow - phase 1: generating delete yaml")
			result, err = runner.DeletePhaseOne(ctx, filepath.Join(iscDir, discHappyPath), workDir, deleteId, testRegistry.Endpoint())
			expectOcMirrorCommandSuccess(result, err)

			By("verifying delete images yaml was created after phase 1")
			expectDeleteImagesYamlExists(deleteYaml)

			By("running delete workflow - phase 2: delete images from registry")
			result, err = runner.DeletePhaseTwo(ctx, deleteYaml, testRegistry.Endpoint(),
				"--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("verifying local registry is empty after delete")
			expectEmptyRegistry(*testRegistry)

		})
	})
})
