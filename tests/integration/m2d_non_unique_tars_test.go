package integration_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// OCP-83128
var _ = Describe("oc-mirror m2d generates non-unique tar file names", func() {
	var workDir string

	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	iscHappyPath := filepath.Join("happy_path", "isc-happy-path.yaml")
	It("should clean existing tar files", func() {
		By("running mirrorToDisk")
		result, err := runner.MirrorToDisk(ctx, filepath.Join(iscDir, iscHappyPath), workDir, "--remove-signatures=true")
		expectOcMirrorCommandSuccess(result, err)

		tarPath := filepath.Join(workDir, "mirror_000001.tar")
		newTarPath := filepath.Join(workDir, "mirror_000002.tar")
		By("verifying tar archive contents")
		expectCorrectTarArchiveContents(filepath.Join(iscDir, iscHappyPath), workDir)
		Expect(newTarPath).ToNot(BeAnExistingFile(), "should generate only one tar file")

		By("renaming existing tar")
		err = os.Rename(tarPath, newTarPath)
		Expect(err).ToNot(HaveOccurred(), "should rename tar file")

		By("running mirrorToDisk again")
		result, err = runner.MirrorToDisk(ctx, filepath.Join(iscDir, iscHappyPath), workDir, "--remove-signatures=true")
		expectOcMirrorCommandSuccess(result, err)

		By("verifying tar archive contents")
		expectCorrectTarArchiveContents(filepath.Join(iscDir, iscHappyPath), workDir)
		Expect(newTarPath).NotTo(BeAnExistingFile(), "previous tar files should be deleted")
		Expect(tarPath).To(BeAnExistingFile(), "mirror_000001.tar should exist")
	})
})
