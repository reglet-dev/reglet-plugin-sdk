package domain_test

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDomainHasNoExternalDependencies verifies that the domain layer
// does not import from application or infrastructure layers.
// This is a critical hexagonal architecture requirement.
func TestDomainHasNoExternalDependencies(t *testing.T) {
	domainPath := "../domain"

	// Parse all Go files in domain/ subdirectories
	fset := token.NewFileSet()

	// Check entities/
	entitiesPattern := filepath.Join(domainPath, "entities", "*.go")
	entitiesFiles, err := filepath.Glob(entitiesPattern)
	require.NoError(t, err, "failed to glob entities files")

	for _, file := range entitiesFiles {
		checkFileImports(t, fset, file, "entities")
	}

	// Check errors/
	errorsPattern := filepath.Join(domainPath, "errors", "*.go")
	errorsFiles, err := filepath.Glob(errorsPattern)
	require.NoError(t, err, "failed to glob errors files")

	for _, file := range errorsFiles {
		checkFileImports(t, fset, file, "errors")
	}

	// Check ports/
	portsPattern := filepath.Join(domainPath, "ports", "*.go")
	portsFiles, err := filepath.Glob(portsPattern)
	require.NoError(t, err, "failed to glob ports files")

	for _, file := range portsFiles {
		// Skip test files for ports (they can import testing and testify)
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		checkFileImports(t, fset, file, "ports")
	}
}

func checkFileImports(t *testing.T, fset *token.FileSet, filename, pkg string) {
	t.Helper()

	f, err := parser.ParseFile(fset, filename, nil, parser.ImportsOnly)
	require.NoError(t, err, "failed to parse %s", filename)

	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Check for forbidden patterns
		forbiddenPackages := []string{
			"github.com/reglet-dev/reglet-sdk/application",
			"github.com/reglet-dev/reglet-sdk/infrastructure",
			"github.com/reglet-dev/reglet-sdk/net",
			"github.com/reglet-dev/reglet-sdk/exec",
			"github.com/reglet-dev/reglet-sdk/log",
			"github.com/reglet-dev/reglet-sdk/helpers",
			// domain/entities CAN be imported by other domain packages (ports, errors)
			"github.com/reglet-dev/reglet-sdk/internal/abi",
		}

		for _, forbidden := range forbiddenPackages {
			assert.NotContains(t, importPath, forbidden,
				"domain/%s package (%s) must not import from %s (violates hexagonal architecture)",
				pkg, filepath.Base(filename), forbidden)
		}

		// Domain can only import:
		// - Standard library
		// - Other domain packages
		// - testify (for tests only, but we skip test files)
		if strings.Contains(importPath, "github.com/reglet-dev/reglet-sdk/") {
			// Must be importing from domain/
			assert.True(t,
				strings.Contains(importPath, "/domain/"),
				"domain/%s package (%s) imports non-domain SDK package: %s",
				pkg, filepath.Base(filename), importPath)
		}
	}
}

// TestDomainEntitiesPortsErrorsExist verifies that required domain packages exist
func TestDomainEntitiesPortsErrorsExist(t *testing.T) {
	domainPath := "../domain"

	// Check that subdirectories exist
	requiredDirs := []string{"entities", "errors", "ports"}

	for _, dir := range requiredDirs {
		fullPath := filepath.Join(domainPath, dir)
		pattern := filepath.Join(fullPath, "*.go")
		files, err := filepath.Glob(pattern)

		require.NoError(t, err, "failed to check %s directory", dir)
		assert.NotEmpty(t, files, "domain/%s should contain Go files", dir)
	}
}
