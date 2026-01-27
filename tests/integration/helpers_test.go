package integration_test

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/gomega"

	"github.com/openshift/oc-mirror/integration/pkg/ocmirror"
)

// releaseSignaturePattern matches signature files for the test release.
// The pattern matches files with SHA256 hash prefix f817.
const releaseSignaturePattern = "*f817*"

// setupWorkDir creates a temporary working directory and copies signature files.
// Returns the working directory path.
func setupWorkDir() string {
	workDir, err := os.MkdirTemp("", "oc-mirror-test-*")
	Expect(err).NotTo(HaveOccurred())

	// Copy release signature files to working directory
	sigDir := filepath.Join(workDir, "working-dir", "signatures")
	err = os.MkdirAll(sigDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	// Copy signature files into working dir
	entries, err := os.ReadDir(keysDir)
	Expect(err).NotTo(HaveOccurred())

	for _, entry := range entries {
		if matched, _ := filepath.Match(releaseSignaturePattern, entry.Name()); matched {
			err = copyFile(filepath.Join(keysDir, entry.Name()), filepath.Join(sigDir, entry.Name()))
			Expect(err).NotTo(HaveOccurred())
		}
	}

	return workDir
}

// cleanupWorkDir removes the working directory.
func cleanupWorkDir(workDir string) {
	if workDir != "" {
		os.RemoveAll(workDir)
	}
}

// expectOcMirrorCommandSuccess asserts that the oc-mirror command completed successfully.
func expectOcMirrorCommandSuccess(result *ocmirror.Result, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(result.ExitCode).To(Equal(0), "oc-mirror failed:\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
}

// expectWorkingDirStructure verifies that the working-dir contains the expected subdirectories.
func expectWorkingDirStructure(workDir string, expectedDirs []string) {
	workingDir := filepath.Join(workDir, "working-dir")
	Expect(workingDir).To(BeADirectory())
	for _, dir := range expectedDirs {
		Expect(filepath.Join(workingDir, dir)).To(BeADirectory(), "missing directory: %s", dir)
	}
}

// expectTarArchiveExists verifies that at least one non-empty tar archive was created.
func expectTarArchiveExists(workDir string) {
	matches, err := filepath.Glob(filepath.Join(workDir, "mirror_*.tar"))
	Expect(err).NotTo(HaveOccurred())
	Expect(matches).NotTo(BeEmpty(), "no tar archive found")
	for _, tarFile := range matches {
		info, err := os.Stat(tarFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Size()).To(BeNumerically(">", 0), "empty tar: %s", tarFile)
	}
}

// expectRepositoriesExist verifies that each expected repository substring is found in the registry.
func expectRepositoriesExist(repos, expected []string) {
	Expect(repos).NotTo(BeEmpty(), "registry has no repositories")
	for _, exp := range expected {
		found := false
		for _, repo := range repos {
			if strings.Contains(repo, exp) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "missing repository %q, got: %v", exp, repos)
	}
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
