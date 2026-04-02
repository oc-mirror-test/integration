package integration_test

import (
	"archive/tar"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/distribution/reference"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/openshift/oc-mirror/integration/pkg/ocmirror"
	"github.com/openshift/oc-mirror/integration/pkg/registry"
)

const (
	// releasePublicKeyFile is the GPG public key filename in the keys directory.
	releasePublicKeyFile = "release-pk.asc"
	platformReleaseRepo     = "openshift/release"

	dirWorkingDir = "working-dir"

	// Working dir subfolders
	dirOperatorCatalog = "operator-catalogs"
	dirHelm            = "helm"

	// cacheRepositoriesSubdir is the path within the oc-mirror cache directory to the local cache repositories.
	cacheRepositoriesSubdir = "docker/registry/v2/repositories"

	// tarRepositoriesPath is the path prefix for OCI repositories inside a tar archive.
	tarRepositoriesPath = "docker/registry/v2/repositories/"
)

// TODO: Remove these structs once we move the integration tests into the oc-mirror repo
type ImageSetConfiguration struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Mirror     MirrorConfig `yaml:"mirror"`
}

type MirrorConfig struct {
	Platform         PlatformConfig   `yaml:"platform"`
	Helm             HelmConfig       `yaml:"helm"`
	AdditionalImages []ImageRef       `yaml:"additionalImages"`
	Operators        []OperatorConfig `yaml:"operators"`
}

type PlatformConfig struct {
	Graph   bool   `yaml:"graph"`
	Release string `yaml:"release"`
}

type HelmConfig struct {
	Repositories []HelmRepository `yaml:"repositories"`
}

type HelmRepository struct {
	Name   string      `yaml:"name"`
	URL    string      `yaml:"url"`
	Charts []HelmChart `yaml:"charts"`
}

type HelmChart struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ImageRef struct {
	Name string `yaml:"name"`
}

type OperatorConfig struct {
	Catalog  string            `yaml:"catalog"`
	Packages []OperatorPackage `yaml:"packages"`
}

type OperatorPackage struct {
	Name string `yaml:"name"`
}

// expectOcMirrorCommandSuccess asserts that the oc-mirror command completed successfully.
func expectOcMirrorCommandSuccess(result *ocmirror.Result, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(result.ExitCode).To(Equal(0), "oc-mirror failed:\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
}

// expectCorrectTarArchiveContents verifies that the tar archive contains the expected
// repositories for the given ISC.
func expectCorrectTarArchiveContents(isc string, workDir string) {
	cfg := parseImageSetConfig(isc)

	matches, err := filepath.Glob(filepath.Join(workDir, "mirror_*.tar"))
	Expect(err).NotTo(HaveOccurred())
	Expect(matches).NotTo(BeEmpty(), "no tar archive found")

	entries := listTarEntries(matches[0])
	Expect(entries).NotTo(BeEmpty(), "tar archive has no entries")

	for _, p := range collectExpectedTarPaths(cfg) {
		expectTarContainsPath(entries, p)
	}
}

// listTarEntries opens a tar file and returns all entry names.
func listTarEntries(tarPath string) []string {
	f, err := os.Open(tarPath)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		closeErr := f.Close()
		Expect(closeErr).NotTo(HaveOccurred())
	}()

	tr := tar.NewReader(f)
	var entries []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		Expect(err).NotTo(HaveOccurred())
		entries = append(entries, hdr.Name)
	}
	return entries
}

// expectTarContainsPath verifies that at least one tar entry equals prefix or is
// nested inside it (i.e. the directory or any of its contents is present).
func expectTarContainsPath(entries []string, prefix string) {
	prefix = filepath.ToSlash(prefix)
	for _, entry := range entries {
		e := filepath.ToSlash(entry)
		if e == prefix || e == prefix+"/" || strings.HasPrefix(e, prefix+"/") {
			return
		}
	}
	Expect(false).To(BeTrue(), "tar archive is missing expected path: %s", prefix)
}

// collectExpectedTarPaths returns all paths expected to be present in the tar archive
// for a given ImageSetConfig. Each content type is checked at its actual location:
// OCI artifacts under docker/registry/v2/repositories/, helm charts as .tgz files
// under working-dir/helm/charts/, and operator catalogs under working-dir/operator-catalogs/.
func collectExpectedTarPaths(cfg ImageSetConfiguration) []string {
	var paths []string

	// Platform release OCI repository
	if cfg.Mirror.Platform.Release != "" {
		paths = append(paths, tarRepositoriesPath+platformReleaseRepo)
	}

	// Operator catalog OCI repositories and working-dir catalog directories
	for _, op := range cfg.Mirror.Operators {
		paths = append(paths, tarRepositoriesPath+extractRepositoryName(op.Catalog))
		paths = append(paths, filepath.Join(dirWorkingDir, dirOperatorCatalog, extractImageName(op.Catalog)))
	}

	// Additional images OCI repositories
	for _, img := range cfg.Mirror.AdditionalImages {
		paths = append(paths, tarRepositoriesPath+extractRepositoryName(img.Name))
	}

	// Helm charts stored as .tgz files (not OCI repositories)
	for _, helmRepo := range cfg.Mirror.Helm.Repositories {
		for _, chart := range helmRepo.Charts {
			paths = append(paths, filepath.Join(dirWorkingDir, dirHelm, "charts", chart.Name+"-"+chart.Version+".tgz"))
		}
	}

	return paths
}

// expectSuccessfulMirrorInRegistry verifies that all the content specified on a given ImageSetConfig has been successfully mirrored into a registry
func expectSuccessfulMirrorInRegistry(isc string, registry registry.Registry) {
	cfg := parseImageSetConfig(isc)
	expectedRepos := collectExpectedRepos(cfg)

	// TODO: We need to verify individual operator images, not only the catalog
	expectRepositoriesExist(registry, expectedRepos)
}

// expectRepositoriesExist verifies that each expected repository substring is found in the registry.
func expectRepositoriesExist(registry registry.Registry, expected []string) {
	repos, err := registry.ListRepositories(ctx)
	Expect(err).NotTo(HaveOccurred())
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

// expectEmptyRegistry verifies that no non-catalog repos have tags remaining after a delete operation.
func expectEmptyRegistry(reg registry.Registry) {
	repos, err := reg.ListRepositories(ctx)
	Expect(err).NotTo(HaveOccurred())

	for _, repo := range repos {
		tags, err := reg.ListTags(ctx, repo)
		Expect(err).NotTo(HaveOccurred())
		if len(tags) == 0 {
			continue
		}
		isCatalog, err := reg.IsCatalog(ctx, repo, tags[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(isCatalog).To(BeTrue(), "non-catalog repo %q still has tags after delete", repo)
	}
}

// parseImageSetConfig gets the path of an ImageSetConfig YAML and returns a parsed ImageSetConfig
func parseImageSetConfig(isc string) ImageSetConfiguration {
	data, err := os.ReadFile(isc)
	Expect(err).NotTo(HaveOccurred())

	var cfg ImageSetConfiguration
	err = yaml.Unmarshal(data, &cfg)
	Expect(err).NotTo(HaveOccurred())
	return cfg
}

// collectExpectedRepositories collects all the repos from an ImageSetConfig
// And returns a slice to be verified later
func collectExpectedRepos(cfg ImageSetConfiguration) []string {
	var expected []string

	// Collect releases
	if cfg.Mirror.Platform.Release != "" {
		// The name of the repo for releases is hardcoded.
		expected = append(expected, platformReleaseRepo)
	}

	// Collect operator catalogs
	for _, op := range cfg.Mirror.Operators {
		expected = append(expected, extractRepositoryName(op.Catalog))
	}

	// Collect helm charts
	for _, helmRepo := range cfg.Mirror.Helm.Repositories {
		if len(helmRepo.Charts) > 0 {
			expected = append(expected, helmRepo.Name)
		}
	}

	// Collect additional images
	for _, img := range cfg.Mirror.AdditionalImages {
		expected = append(expected, extractRepositoryName(img.Name))
	}

	return expected

}

// extractRepositoryName parses an image reference and extracts the repository part.
// It removes the registry prefix and any tag/digest suffix.
// Examples:
//   - "quay.io/oc-mirror/oc-mirror-dev:test-catalog-latest" -> "oc-mirror/oc-mirror-dev"
//   - "quay.io/openshifttest/hello-openshift@sha256:..." -> "openshifttest/hello-openshift"
func extractRepositoryName(imageRef string) string {
	ref, err := reference.ParseNormalizedNamed(imageRef)
	Expect(err).NotTo(HaveOccurred(), "failed to parse image ref: %s", imageRef)
	return reference.Path(ref)
}

// extractImageName parses an image reference and extracts just the image name (final component).
// It removes the registry prefix, organization/namespace, and any tag/digest suffix.
// Examples:
//   - "quay.io/oc-mirror/oc-mirror-dev:test-catalog-latest" -> "oc-mirror-dev"
//   - "registry.redhat.io/redhat/redhat-operator-index:v4.17" -> "redhat-operator-index"
func extractImageName(imageRef string) string {
	ref, err := reference.ParseNormalizedNamed(imageRef)
	Expect(err).NotTo(HaveOccurred(), "failed to parse image ref: %s", imageRef)
	return path.Base(reference.Path(ref))
}

// defaultCacheDir returns the default oc-mirror cache directory (~/.oc-mirror/.cache),
// used when oc-mirror is run without --cache-dir.
func defaultCacheDir() string {
	home, err := os.UserHomeDir()
	Expect(err).NotTo(HaveOccurred())
	return filepath.Join(home, ".oc-mirror", ".cache")
}

// expectSuccessfulMirrorInLocalCache verifies that all content specified in a given ImageSetConfig
// has been cached locally by walking the cache dir.
func expectSuccessfulMirrorInLocalCache(isc string, cacheDir string) {
	cfg := parseImageSetConfig(isc)
	expectedRepos := collectExpectedRepos(cfg)

	repos, err := listLocalCacheRepositories(cacheDir)
	Expect(err).NotTo(HaveOccurred())
	Expect(repos).NotTo(BeEmpty(), "local cache has no repositories")

	for _, exp := range expectedRepos {
		found := false
		for _, repo := range repos {
			if strings.Contains(repo, exp) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "missing repository %q in local cache, got: %v", exp, repos)
	}
}

// listLocalCacheRepositories walks <cacheDir>/docker/registry/v2/repositories/ and
// returns repository paths by locating _manifests directories.
func listLocalCacheRepositories(cacheDir string) ([]string, error) {
	reposRoot := filepath.Join(cacheDir, cacheRepositoriesSubdir)

	var repos []string
	err := filepath.Walk(reposRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "_manifests" {
			repoPath, relErr := filepath.Rel(reposRoot, filepath.Dir(path))
			if relErr == nil {
				repos = append(repos, repoPath)
			}
		}
		return nil
	})
	return repos, err
}

// copyFile copies a single file from src to dst, preserving file permissions
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Get the source file's permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, info.Mode().Perm())
}

// setupWorkDir creates a temporary working directory and copies signature files.
// Returns the working directory path.
func setupWorkDir() string {
	workDir, err := os.MkdirTemp("", "oc-mirror-test-*")
	Expect(err).NotTo(HaveOccurred())

	// Copy release signature files to working directory
	sigDir := filepath.Join(workDir, dirWorkingDir, "signatures")
	err = os.MkdirAll(sigDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	// Find the single signature file in keys/ (any file that isn't the public key)
	entries, err := os.ReadDir(keysDir)
	Expect(err).NotTo(HaveOccurred())

	var sigFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != releasePublicKeyFile {
			sigFiles = append(sigFiles, entry.Name())
		}
	}
	Expect(sigFiles).To(HaveLen(1), "expected exactly one signature file in keys/, found: %v", sigFiles)

	err = copyFile(filepath.Join(keysDir, sigFiles[0]), filepath.Join(sigDir, sigFiles[0]))
	Expect(err).NotTo(HaveOccurred())

	return workDir
}

// expectDeleteImagesYamlExists verifies that the delete images yaml file was created after delete phase 1.
func expectDeleteImagesYamlExists(path string) {
	_, err := os.Stat(path)
	Expect(err).NotTo(HaveOccurred(), "delete images yaml not found at: %s", path)
}

// expectValidMappingFile verifies that the dry-run mapping.txt file was created, that every line
// follows the source=destination format, and that all expected repositories from the ISC are represented.
func expectValidMappingFile(workDir, iscPath string) {
	mappingPath := filepath.Join(workDir, dirWorkingDir, "dry-run", "mapping.txt")
	data, err := os.ReadFile(mappingPath)
	Expect(err).NotTo(HaveOccurred(), "mapping.txt not found at: %s", mappingPath)

	content := strings.TrimSpace(string(data))
	Expect(content).NotTo(BeEmpty(), "mapping.txt is empty")

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		Expect(parts).To(HaveLen(2), "mapping line does not follow source=destination format: %s", line)
		Expect(parts[0]).NotTo(BeEmpty(), "source is empty in mapping line: %s", line)
		Expect(parts[1]).NotTo(BeEmpty(), "destination is empty in mapping line: %s", line)
	}

	cfg := parseImageSetConfig(iscPath)
	expectedRepos := collectExpectedRepos(cfg)
	for _, repo := range expectedRepos {
		Expect(content).To(ContainSubstring(repo),
			"mapping.txt does not contain expected repository %q", repo)
	}
}

// expectNoTarArchive verifies that no tar archive was created in the working directory.
func expectNoTarArchive(workDir string) {
	matches, err := filepath.Glob(filepath.Join(workDir, "mirror_*.tar"))
	Expect(err).NotTo(HaveOccurred())
	Expect(matches).To(BeEmpty(), "expected no tar archive but found: %v", matches)
}

// expectNoRepositoriesInRegistry verifies that the registry contains no repositories at all.
func expectNoRepositoriesInRegistry(reg registry.Registry) {
	repos, err := reg.ListRepositories(ctx)
	Expect(err).NotTo(HaveOccurred())
	Expect(repos).To(BeEmpty(), "expected registry to have no repositories, but found: %v", repos)
}

// cleanupWorkDir removes the working directory.
func cleanupWorkDir(workDir string) {
	if workDir != "" {
		os.RemoveAll(workDir)
	}
}
