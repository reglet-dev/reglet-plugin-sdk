package grant_store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/ports"
	"gopkg.in/yaml.v3"
)

// fileStoreConfig holds configuration for the FileStore.
type fileStoreConfig struct {
	path     string      // Path to the grants file
	dirPerm  os.FileMode // Permission for created directories
	filePerm os.FileMode // Permission for the grants file
}

func defaultFileStoreConfig() fileStoreConfig {
	return fileStoreConfig{
		path:     filepath.Join(os.Getenv("HOME"), ".reglet", "grants.yaml"),
		dirPerm:  0o755, // User config directory
		filePerm: 0o600, // User-only read/write (secure default)
	}
}

// FileStoreOption configures a FileStore instance.
type FileStoreOption func(*fileStoreConfig)

// WithPath sets the path to the grants file.
func WithPath(path string) FileStoreOption {
	return func(c *fileStoreConfig) {
		c.path = path
	}
}

// WithFilePermissions sets the file permissions for the grants file.
// Default is 0o600 (user-only). Use with caution.
func WithFilePermissions(perm os.FileMode) FileStoreOption {
	return func(c *fileStoreConfig) {
		c.filePerm = perm
	}
}

// WithDirPermissions sets the directory permissions for the grants directory.
// Default is 0o755.
func WithDirPermissions(perm os.FileMode) FileStoreOption {
	return func(c *fileStoreConfig) {
		c.dirPerm = perm
	}
}

// FileStore provides file-based persistence for capability grants.
type FileStore struct {
	config fileStoreConfig
}

// NewFileStore creates a new FileStore with the given options.
func NewFileStore(opts ...FileStoreOption) ports.GrantStore {
	cfg := defaultFileStoreConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &FileStore{config: cfg}
}

// Load retrieves all granted capabilities.
func (s *FileStore) Load() (*entities.GrantSet, error) {
	data, err := os.ReadFile(s.config.path)
	if os.IsNotExist(err) {
		// Return empty set if file doesn't exist
		return &entities.GrantSet{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read grant store: %w", err)
	}

	var grants entities.GrantSet
	if err := yaml.Unmarshal(data, &grants); err != nil {
		return nil, fmt.Errorf("failed to parse grant store: %w", err)
	}
	return &grants, nil
}

// Save persists the granted capabilities.
func (s *FileStore) Save(grants *entities.GrantSet) error {
	data, err := yaml.Marshal(grants)
	if err != nil {
		return fmt.Errorf("failed to marshal grants: %w", err)
	}

	dir := filepath.Dir(s.config.path)
	if err := os.MkdirAll(dir, s.config.dirPerm); err != nil {
		return fmt.Errorf("failed to create grant store directory: %w", err)
	}

	if err := os.WriteFile(s.config.path, data, s.config.filePerm); err != nil {
		return fmt.Errorf("failed to write grant store: %w", err)
	}
	return nil
}

// ConfigPath returns the path to the backing store.
func (s *FileStore) ConfigPath() string {
	return s.config.path
}
