---
description: Convert a manual test case description into a Ginkgo oc-v2 integration test for oc-mirror
user-invocable: true
---

# Automate Manual Test Case to Go

Convert a manual test case description into a Ginkgo v2 integration test for the oc-mirror integration test suite.

## Your task

The user will provide a manual test case — either a free-form description, a numbered step list, or a structured spec. Read it carefully, then generate a complete, compilable Go Ginkgo v2 test that fits naturally into this codebase.

## Step-by-step process

### 1. Clarify if needed

If the test case is ambiguous on any of the following, ask before writing code:
- Which mirror mode? (`mirrorToMirror`, `mirrorToDisk` + `diskToMirror`, or both)
- Does the test include a delete workflow (phase 1 + phase 2)?
- Is a new `ImageSetConfiguration` or `DeleteImageSetConfiguration` YAML needed, or does an existing one suffice?
- Should the test go in an existing file or a new one?

### 2. Identify the test structure

Extract from the description:
- **Describe label** — the high-level feature or scenario group (e.g., `"mirrorToMirror"`, `"mirrorToDisk + diskToMirror"`)
- **Inner Describe label** — the specific scenario (e.g., `"happy path with additional images only"`)
- **It label** — what the test asserts in one sentence (e.g., `"should mirror additional images to a local registry"`)
- **Steps** — each logical action becomes a `By(...)` + corresponding runner call and assertion

### 3. Discover available APIs

Before writing code, read these files to learn the current runner methods, assertion helpers, and global variables:

- `tests/integration/integration_suite_test.go` — suite setup, global variables (`runner`, `ctx`, `testRegistry`, `iscDir`, `cacheDir`, `workDir`, etc.)
- `tests/integration/helpers_test.go` — assertion/helper functions (e.g., `expect*` helpers)
- An existing test file (e.g., `tests/integration/m2m_test.go`) — to see how runner methods and helpers are called in practice, including common extra args

Prefer existing APIs. If the test scenario requires a runner method or assertion helper that doesn't exist yet, add it following the patterns in the files above.

### 5. Analyze oc-mirror APIs if needed

If the technical details of the scenario are not clear, take a look at the oc-mirror code:
- First, check if there is already a copy of oc-mirror in `./oc-mirror`.
- If not, ask the user for a path to the oc-mirror repo, and ask for permission to read that repo.
- If the user doesn't have a copy of the oc-mirror repo, ask for permission to clone it under `./oc-mirror`. This is the link to the repo: `github.com:openshift/oc-mirror.git`.

### 4. Write the test

Follow this exact template, adapting names and steps:

```go
package integration_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("<describe-label>", func() {
	var workDir string
	BeforeEach(func() {
		workDir = setupWorkDir()
	})

	AfterEach(func() {
		cleanupWorkDir(workDir)
	})

	Describe("<inner-describe-label>", func() {
		iscFile := "<isc-filename>.yaml"

		It("<it-label>", func() {
			By("<step description>")
			result, err := runner.MirrorToMirror(ctx, filepath.Join(iscDir, iscFile), workDir, testRegistry.Endpoint(),
				"--remove-signatures=true", "--dest-tls-verify=false")
			expectOcMirrorCommandSuccess(result, err)

			By("<next step description>")
			expectSuccessfulMirrorInRegistry(filepath.Join(iscDir, iscFile), *testRegistry)
		})
	})
})
```

Rules:
- Consider if the test scenario covers all critical paths, and suggest new ones.
- Implement robust tests that do not cause false positives.
- Avoid meaningless or redundant tests, we aim for quality. If the scenario proposed is already covered, say so.
- Implement meaningful assertions, do not stay at surface level ones.
- Always use `BeforeEach`/`AfterEach` with `setupWorkDir()`/`cleanupWorkDir(workDir)` — never inline setup.
- Use `filepath.Join` for all path construction.
- Never import packages that are already covered by the test helpers (e.g., `os`, `yaml`) unless the new test truly needs them directly.
- The package is always `integration_test`.

### 5. Choose or create ISC/DISC YAML configs

- Check existing configs in `tests/integration/testdata/imagesetconfigs/` first.
- If a new config is needed, create it there following the same YAML structure as `isc-happy-path.yaml` (kind `ImageSetConfiguration`) or `disc-happy-path.yaml` (kind `DeleteImageSetConfiguration`).
- Use images that already exist in the test infrastructure where possible: `quay.io/oc-mirror/release/test-release-index:v0.0.1`, `quay.io/oc-mirror/oc-mirror-dev:test-catalog-latest`, `quay.io/openshifttest/hello-openshift@sha256:...`.

### 6. Choose the right file

| Scenario type | Target file |
|---|---|
| Mirror-to-mirror only | `tests/integration/m2m_test.go` |
| MirrorToDisk + DiskToMirror | `tests/integration/m2d_d2m_test.go` |
| New distinct feature/workflow | New file: `tests/integration/<feature>_test.go` |

Add the new `Describe` block to the chosen file. Never create a new file if an existing one is a natural fit.

### 7. Run the tests

Ask the user for permission to run the tests. If the user agrees, run the newly implemented tests by running a command like this


### 7. Output

Provide:
1. The complete Go test code (ready to paste or write).
2. Any new ISC/DISC YAML config files needed.
3. A one-line summary of where each file goes.

Do not explain what Ginkgo is or how the project works — the user knows. Be concise.