package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/oc-mirror/integration/pkg/registry"
)

// OCP-87266
var _ = Describe("enclave OCI catalog filtering", func() {
	var workDir string

	// Expected bundles for foo package with beta channel versions 0.2.0-0.3.1
	// These are determined by the actual catalog content, not the ISC filter
	var expectedBundles = []string{"foo.v0.2.0", "foo.v0.3.0", "foo.v0.3.1"}

	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	It("should preserve filtered content and avoid duplicate channels when re-filtering an already filtered catalog", func() {
		By("parsing ISC to extract catalog, package, and channel info")
		isc1Path := filepath.Join(iscDir, "operators", "isc-operator-version-range.yaml")
		cfg := parseImageSetConfig(isc1Path)
		Expect(cfg.Mirror.Operators).NotTo(BeEmpty(), "ISC must have at least one operator")
		Expect(cfg.Mirror.Operators[0].Packages).NotTo(BeEmpty(), "operator must have at least one package")

		sourceCatalog := cfg.Mirror.Operators[0].Catalog
		packageName := cfg.Mirror.Operators[0].Packages[0].Name
		var expectedChannels []string
		for _, ch := range cfg.Mirror.Operators[0].Packages[0].Channels {
			expectedChannels = append(expectedChannels, ch.Name)
		}
		GinkgoWriter.Printf("ISC: catalog=%s, package=%s, channels=%v\n", sourceCatalog, packageName, expectedChannels)

		By("starting registry2 to model registry1->registry2 mirror flow")
		registry2ConfigPath := filepath.Join(workDir, "registry2-config.yaml")
		createAdditionalRegistryConfig(registry2ConfigPath, filepath.Join(workDir, "registry2-storage"), 5001)
		registry2, err := registry.Start(ctx, registry2ConfigPath, 5001, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred(), "failed to start registry2")
		defer func() {
			stopErr := registry2.Stop()
			if stopErr != nil {
				GinkgoWriter.Printf("Failed to stop registry2: %v\n", stopErr)
			}
		}()
		Expect(registry2.WaitReady(ctx, 30*time.Second)).To(Succeed(), "registry2 not ready")

		By("running mirrorToMirror using ISC with version range filter")
		result, err := runner.MirrorToMirror(ctx, isc1Path, workDir, testRegistry.Endpoint(),
			"--remove-signatures=true", "--dest-tls-verify=false")
		expectOcMirrorCommandSuccess(result, err)
		expectSuccessfulMirrorInRegistry(isc1Path, *testRegistry)

		By("capturing the mirrored filtered catalog reference from registry1")
		filteredCatalogRef := resolveMirroredCatalogReference(*testRegistry, sourceCatalog)
		filteredCatalogRef = ensureNonDigestCatalogTag(*testRegistry, filteredCatalogRef)
		GinkgoWriter.Printf("Filtered catalog in registry1: %s\n", filteredCatalogRef)
		reposAfterISC1 := listSortedRepositories(*testRegistry)

		By("verifying registry1 catalog content: bundles, channels, no duplicates")
		expectCatalogContentValid(ctx, *testRegistry, sourceCatalog, packageName, expectedBundles, expectedChannels)

		By("creating ISC with full: true from registry1 mirrored catalog")
		isc2Path := filepath.Join(workDir, "isc-enclave-full.yaml")
		writeISC(isc2Path, fmt.Sprintf(`kind: ImageSetConfiguration
apiVersion: mirror.openshift.io/v2alpha1
mirror:
  operators:
    - catalog: %s
      full: true
`, filteredCatalogRef))

		By("running mirrorToMirror from registry1 to registry2")
		result, err = runner.MirrorToMirror(ctx, isc2Path, workDir, registry2.Endpoint(),
			"--remove-signatures=true", "--dest-tls-verify=false", "--src-tls-verify=false")
		expectOcMirrorCommandSuccess(result, err)

		By("verifying registry2 has the same repositories as registry1")
		reposAfterISC2 := listSortedRepositories(*registry2)
		Expect(reposAfterISC2).To(Equal(reposAfterISC1), "registry2 mirrored repository set should match registry1")

		By("verifying registry2 catalog content after re-filtering: bundles preserved, channels preserved, no duplicates")
		expectCatalogContentValid(ctx, *registry2, filteredCatalogRef, packageName, expectedBundles, expectedChannels)
	})
})

func writeISC(path, content string) {
	err := os.WriteFile(path, []byte(content), 0o644)
	Expect(err).NotTo(HaveOccurred())
}

func resolveMirroredCatalogReference(registryRef registry.Registry, sourceCatalogRef string) string {
	repo := extractRepositoryName(sourceCatalogRef)
	tags, err := registryRef.ListTags(ctx, repo)
	Expect(err).NotTo(HaveOccurred())
	Expect(tags).NotTo(BeEmpty(), "expected mirrored catalog repository %q to contain at least one tag", repo)

	sort.Strings(tags)
	preferredTag := extractReferenceTag(sourceCatalogRef)
	if preferredTag != "" {
		for _, tag := range tags {
			if tag == preferredTag {
				return registryRef.Endpoint() + "/" + repo + ":" + tag
			}
		}
	}

	digestTagPattern := regexp.MustCompile(`^sha256-[a-f0-9]{64}$`)
	for _, tag := range tags {
		if !digestTagPattern.MatchString(tag) {
			return registryRef.Endpoint() + "/" + repo + ":" + tag
		}
	}

	return registryRef.Endpoint() + "/" + repo + ":" + tags[0]
}

func extractReferenceTag(ref string) string {
	if strings.Contains(ref, "@") {
		return ""
	}

	lastColon := strings.LastIndex(ref, ":")
	lastSlash := strings.LastIndex(ref, "/")
	if lastColon <= lastSlash {
		return ""
	}

	return ref[lastColon+1:]
}

func ensureNonDigestCatalogTag(registryRef registry.Registry, catalogRef string) string {
	digestTagPattern := regexp.MustCompile(`^sha256-[a-f0-9]{64}$`)

	parsedRef, err := name.ParseReference(catalogRef, name.Insecure)
	Expect(err).NotTo(HaveOccurred(), "failed to parse catalog reference %q", catalogRef)

	tagRef, ok := parsedRef.(name.Tag)
	Expect(ok).To(BeTrue(), "expected catalog reference with tag, got %q", catalogRef)
	if !digestTagPattern.MatchString(tagRef.TagStr()) {
		return catalogRef
	}

	img, err := remote.Image(tagRef, remote.WithAuth(authn.Anonymous), remote.WithContext(ctx))
	Expect(err).NotTo(HaveOccurred(), "failed to fetch catalog image for %q", catalogRef)

	retaggedRef, err := name.NewTag(
		registryRef.Endpoint()+"/"+tagRef.RepositoryStr()+":refilter-source",
		name.Insecure,
	)
	Expect(err).NotTo(HaveOccurred(), "failed to create retagged catalog reference for %q", catalogRef)

	err = remote.Write(retaggedRef, img, remote.WithAuth(authn.Anonymous), remote.WithContext(ctx))
	Expect(err).NotTo(HaveOccurred(), "failed to push retagged catalog %q", retaggedRef.Name())

	return retaggedRef.Name()
}

func listSortedRepositories(registryRef registry.Registry) []string {
	repos, err := registryRef.ListRepositories(ctx)
	Expect(err).NotTo(HaveOccurred())
	sort.Strings(repos)
	return repos
}
