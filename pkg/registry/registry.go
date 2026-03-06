package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"gopkg.in/yaml.v3"
)

// Registry manages a distribution/distribution registry instance.
type Registry struct {
	cmd         *exec.Cmd
	configPath  string
	storagePath string
	port        int
	client      name.Registry
	cancel      context.CancelFunc
}

// registryConfig represents the registry configuration file structure.
type registryConfig struct {
	Storage struct {
		Filesystem struct {
			RootDirectory string `yaml:"rootdirectory"`
		} `yaml:"filesystem"`
	} `yaml:"storage"`
}

// Start launches the registry with the given config file and waits for it to be ready.
func Start(ctx context.Context, configPath string, port int, logWriter io.Writer) (*Registry, error) {
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("registry config not found: %w", err)
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry config: %w", err)
	}

	var config registryConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse registry config: %w", err)
	}

	storagePath := filepath.Join(config.Storage.Filesystem.RootDirectory, "docker")

	ctx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(ctx, "registry", "serve", configPath)

	cmd.Stdout = logWriter
	cmd.Stderr = logWriter

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start registry: %w", err)
	}

	reg, err := name.NewRegistry(fmt.Sprintf("localhost:%d", port), name.Insecure)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to set up registry: %w", err)
	}

	return &Registry{
		cmd:         cmd,
		configPath:  configPath,
		storagePath: storagePath,
		port:        port,
		cancel:      cancel,
		client:      reg,
	}, nil
}

// WaitReady polls the registry health endpoint until it responds or timeout is reached.
func (r *Registry) WaitReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		_, err := remote.Catalog(ctx, r.client, remote.WithAuth(authn.Anonymous))
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("registry not ready after %v", timeout)
		case <-ticker.C:
		}

	}
}

// Stop terminates the registry process.
func (r *Registry) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}

	var waitErr error
	if r.cmd != nil && r.cmd.Process != nil {
		waitErr = r.cmd.Wait()
	}

	err := os.RemoveAll(r.storagePath)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to remove registry storage: %w", err), waitErr)
	}

	return waitErr
}

// Endpoint returns the registry endpoint address.
func (r *Registry) Endpoint() string {
	return fmt.Sprintf("localhost:%d", r.port)
}

// ListRepositories returns all repositories in the registry.
func (r *Registry) ListRepositories(ctx context.Context) ([]string, error) {
	return remote.Catalog(ctx, r.client, remote.WithAuth(authn.Anonymous))
}

// ListTags returns all the tags for a given repository in the registry
func (r *Registry) ListTags(ctx context.Context, repo string) ([]string, error) {
	ref, err := name.NewRepository(fmt.Sprintf("%s/%s", r.Endpoint(), repo), name.Insecure)
	if err != nil {
		return nil, fmt.Errorf("couldn't list tags for repo %s: %w", repo, err)
	}

	return remote.List(ref, remote.WithAuth(authn.Anonymous), remote.WithContext(ctx))
}

// IsCatalog returns true if the given repository and tag is an OLM operator catalog image.
func (r *Registry) IsCatalog(ctx context.Context, repo, tag string) (bool, error) {
	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", r.Endpoint(), repo, tag), name.Insecure)
	if err != nil {
		return false, fmt.Errorf("couldn't create tag reference for %s:%s: %w", repo, tag, err)
	}

	img, err := remote.Image(ref, remote.WithAuth(authn.Anonymous), remote.WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("couldn't fetch image %s:%s: %w", repo, tag, err)
	}

	cf, err := img.ConfigFile()
	if err != nil {
		return false, fmt.Errorf("couldn't get config file for %s:%s: %w", repo, tag, err)
	}

	_, ok := cf.Config.Labels["operators.operatorframework.io.index.configs.v1"]
	return ok, nil
}
