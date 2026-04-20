package integration_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/oc-mirror/integration/pkg/ocmirror"
)

// OCPBUGS-57461 and test case OCP-83817
var _ = Describe("oc-mirror should error out with empty tar files", func() {
	var workDir string

	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	It("first tar file is empty", func() {
		iscHappyPath := filepath.Join("happy_path", "isc-happy-path.yaml")

		By("creating an empty tar file")
		createEmptyTar(workDir, "mirror_000001.tar")

		By("running diskToMirror")
		result, err := runner.DiskToMirror(ctx, filepath.Join(iscDir, iscHappyPath), workDir, testRegistry.Endpoint(),
			"--remove-signatures=true", "--dest-tls-verify=false")
		expectOcMirrorEmptyTarError(result, err)
	})

	It("tar file other than first is empty", func() {
		iscHappyPath := filepath.Join("happy_path", "isc-happy-path.yaml")

		By("running mirrorToDisk to generate a valid first tar file")
		result, err := runner.MirrorToDisk(ctx, filepath.Join(iscDir, iscHappyPath), workDir, "--remove-signatures=true")
		expectOcMirrorCommandSuccess(result, err)

		By("verifying tar archive contents")
		expectCorrectTarArchiveContents(filepath.Join(iscDir, iscHappyPath), workDir)

		By("creating an extra empty tar file")
		createEmptyTar(workDir, "mirror_000002.tar")

		By("running diskToMirror")
		result, err = runner.DiskToMirror(ctx, filepath.Join(iscDir, iscHappyPath), workDir, testRegistry.Endpoint(),
			"--remove-signatures=true", "--dest-tls-verify=false")
		expectOcMirrorEmptyTarError(result, err)
	})
})

func createEmptyTar(workdir, name string) {
	filename := filepath.Join(workdir, name)
	file, err := os.Create(filename)
	Expect(err).ToNot(HaveOccurred(), "should create tar file")
	err = file.Close()
	Expect(err).ToNot(HaveOccurred(), "should close tar file")
	Expect(filename).To(BeAnExistingFile(), "tar file shoud exist")
	stat, err := os.Stat(filename)
	Expect(err).ToNot(HaveOccurred(), "should stat tar file")
	Expect(stat.Size()).To(BeZero(), "tar file should be empty")
}

func expectOcMirrorEmptyTarError(result *ocmirror.Result, err error) {
	Expect(err).ToNot(HaveOccurred())
	Expect(result.ExitCode).ToNot(BeZero(), "expected non-zero exit code")
	Expect(result.Stdout).To(ContainSubstring("empty archive file"), "should contain empty file error")
}
