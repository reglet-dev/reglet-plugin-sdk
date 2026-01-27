package hostfuncs

import (
	"context"
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
)

func TestCapabilityChecker_Check_NoGrants(t *testing.T) {
	checker := NewCapabilityChecker(nil)

	err := checker.Check("unknown-plugin", "exec", "ls")
	if err == nil {
		t.Error("expected error for plugin with no grants")
	}
}

func TestCapabilityChecker_Check_UnknownKind(t *testing.T) {
	grants := map[string]*entities.GrantSet{
		"test-plugin": {},
	}
	checker := NewCapabilityChecker(grants)

	err := checker.Check("test-plugin", "unknown", "pattern")
	if err == nil {
		t.Error("expected error for unknown capability kind")
	}
}

func TestCapabilityChecker_ExecCapability(t *testing.T) {
	grants := map[string]*entities.GrantSet{
		"test-plugin": {
			Exec: &entities.ExecCapability{
				Commands: []string{"ls", "cat"},
			},
		},
	}
	checker := NewCapabilityChecker(grants)

	tests := []struct {
		name       string
		command    string
		wantErr    bool
	}{
		{"allowed command", "ls", false},
		{"allowed command 2", "cat", false},
		{"denied command", "rm", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Check("test-plugin", "exec", tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCapabilityChecker_EnvironmentCapability(t *testing.T) {
	grants := map[string]*entities.GrantSet{
		"test-plugin": {
			Env: &entities.EnvironmentCapability{
				Variables: []string{"HOME", "PATH"},
			},
		},
	}
	checker := NewCapabilityChecker(grants)

	tests := []struct {
		name     string
		variable string
		wantErr  bool
	}{
		{"allowed var", "HOME", false},
		{"allowed var 2", "PATH", false},
		{"denied var", "SECRET_KEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.Check("test-plugin", "env", tt.variable)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCapabilityChecker_ToCapabilityGetter(t *testing.T) {
	grants := map[string]*entities.GrantSet{
		"test-plugin": {
			Env: &entities.EnvironmentCapability{
				Variables: []string{"PATH"},
			},
			Exec: &entities.ExecCapability{
				Commands: []string{"ls"},
			},
		},
	}
	checker := NewCapabilityChecker(grants)
	getter := checker.ToCapabilityGetter("test-plugin")

	tests := []struct {
		name       string
		capability string
		want       bool
	}{
		{"env:PATH allowed", "env:PATH", true},
		{"env:HOME denied", "env:HOME", false},
		{"exec ls allowed", "ls", true},
		{"exec rm denied", "rm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getter("test-plugin", tt.capability)
			if got != tt.want {
				t.Errorf("CapabilityGetter(%q) = %v, want %v", tt.capability, got, tt.want)
			}
		})
	}
}

func TestCapabilityPluginNameContext(t *testing.T) {
	ctx := context.Background()

	// Should not be present initially
	if _, ok := CapabilityPluginNameFromContext(ctx); ok {
		t.Error("expected no plugin name in empty context")
	}

	// Add plugin name
	ctx = WithCapabilityPluginName(ctx, "my-plugin")

	// Should be present now
	name, ok := CapabilityPluginNameFromContext(ctx)
	if !ok {
		t.Error("expected plugin name to be present")
	}
	if name != "my-plugin" {
		t.Errorf("plugin name = %q, want %q", name, "my-plugin")
	}
}

func TestNewCapabilityChecker_Options(t *testing.T) {
	grants := map[string]*entities.GrantSet{}

	// Test with custom options
	checker := NewCapabilityChecker(grants,
		WithCapabilityWorkingDirectory("/custom/path"),
		WithCapabilitySymlinkResolution(false),
	)

	if checker.cwd != "/custom/path" {
		t.Errorf("cwd = %q, want %q", checker.cwd, "/custom/path")
	}
}
