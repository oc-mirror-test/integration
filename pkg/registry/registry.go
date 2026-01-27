package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Registry manages a distribution/distribution registry instance.
type Registry struct {
	cmd         *exec.Cmd
	configPath  string
	storagePath string
	port        int
	client      *http.Client
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

	return &Registry{
		cmd:         cmd,
		configPath:  configPath,
		storagePath: storagePath,
		port:        port,
		cancel:      cancel,
		client:      &http.Client{Timeout: time.Second},
	}, nil
}

// WaitReady polls the registry health endpoint until it responds or timeout is reached.
func (r *Registry) WaitReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	healthURL := fmt.Sprintf("http://localhost:%d/v2/", r.port)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := r.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
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
		return fmt.Errorf("failed to remove registry storage: %w", err)
	}

	return waitErr
}

// Endpoint returns the registry endpoint address.
func (r *Registry) Endpoint() string {
	return fmt.Sprintf("localhost:%d", r.port)
}

// ListRepositories returns all repositories in the registry.
func (r *Registry) ListRepositories() ([]string, error) {
	catalogURL := fmt.Sprintf("http://localhost:%d/v2/_catalog", r.port)

	resp, err := r.client.Get(catalogURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog returned status %d", resp.StatusCode)
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to decode catalog: %w", err)
	}

	return catalog.Repositories, nil
}
