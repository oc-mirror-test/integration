package integration_test

import (
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("operators catalog digest rebuild tagging", func() {
	var workDir string

	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	It("should tag rebuilt catalog images with a value matching the manifest digest", func() {
		By("resolving source catalog references to digest-pinned values")
		digestISCPath := createDigestPinnedOperatorISC(
			filepath.Join(iscDir, "operators", "isc-operator-pinned-version.yaml"),
			workDir,
		)

		By("running mirrorToMirror with digest-pinned catalog references")
		result, err := runner.MirrorToMirror(ctx, digestISCPath, workDir, testRegistry.Endpoint(),
			"--remove-signatures=true", "--dest-tls-verify=false")
		expectOcMirrorCommandSuccess(result, err)

		By("verifying mirrored content exists in the registry")
		expectSuccessfulMirrorInRegistry(digestISCPath, *testRegistry)

		By("verifying the rebuilt catalog tag matches the fetched manifest digest")
		expectRebuiltTagMatchesDigest(ctx, *testRegistry, digestISCPath)
	})
})

func createDigestPinnedOperatorISC(baseISCPath, workDir string) string {
	cfg := parseImageSetConfig(baseISCPath)
	Expect(cfg.Mirror.Operators).NotTo(BeEmpty(), "expected at least one operator catalog in ISC")

	for i, op := range cfg.Mirror.Operators {
		ref, err := name.ParseReference(op.Catalog)
		Expect(err).NotTo(HaveOccurred(), "failed to parse catalog reference %q", op.Catalog)

		desc, err := remote.Get(ref, remote.WithAuth(authn.Anonymous), remote.WithContext(ctx))
		Expect(err).NotTo(HaveOccurred(), "failed to resolve digest for catalog %q", op.Catalog)

		cfg.Mirror.Operators[i].Catalog = ref.Context().Name() + "@" + desc.Digest.String()
	}

	data, err := yaml.Marshal(&cfg)
	Expect(err).NotTo(HaveOccurred())

	digestISCPath := filepath.Join(workDir, "isc-operator-catalog-digest.yaml")
	err = os.WriteFile(digestISCPath, data, 0o644)
	Expect(err).NotTo(HaveOccurred())

	return digestISCPath
}
