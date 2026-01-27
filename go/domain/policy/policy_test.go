package policy_test

import (
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/policy"
	"github.com/stretchr/testify/assert"
)

func TestPolicy_CheckNetwork(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))

	grants := &entities.GrantSet{
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"example.com", "*.internal"}, Ports: []string{"80", "443", "8000-8010", "*"}},
			},
		},
	}

	tests := []struct {
		name string
		req  entities.NetworkRequest
		want bool
	}{
		{"Allowed host and port", entities.NetworkRequest{Host: "example.com", Port: 80}, true},
		{"Allowed wildcard host", entities.NetworkRequest{Host: "svc.internal", Port: 443}, true},
		{"Allowed range port", entities.NetworkRequest{Host: "example.com", Port: 8005}, true},
		{"Allowed wildcard port", entities.NetworkRequest{Host: "example.com", Port: 9999}, true},
		{"Denied host", entities.NetworkRequest{Host: "google.com", Port: 80}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, p.CheckNetwork(tt.req, grants))
		})
	}
}

func TestPolicy_CheckNetwork_SpecificPorts(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"example.com"}, Ports: []string{"80", "8000-8010"}},
			},
		},
	}

	assert.True(t, p.CheckNetwork(entities.NetworkRequest{Host: "example.com", Port: 80}, grants))
	assert.True(t, p.CheckNetwork(entities.NetworkRequest{Host: "example.com", Port: 8005}, grants))
	assert.False(t, p.CheckNetwork(entities.NetworkRequest{Host: "example.com", Port: 443}, grants))
	assert.False(t, p.CheckNetwork(entities.NetworkRequest{Host: "example.com", Port: 8011}, grants))
}

func TestPolicy_CheckNetwork_MultipleRules(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	// Test that multiple rules work correctly - each rule is independent
	grants := &entities.GrantSet{
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"api.internal"}, Ports: []string{"80"}},
				{Hosts: []string{"*.external.com"}, Ports: []string{"443"}},
			},
		},
	}

	// Should match first rule
	assert.True(t, p.CheckNetwork(entities.NetworkRequest{Host: "api.internal", Port: 80}, grants))
	// Should match second rule
	assert.True(t, p.CheckNetwork(entities.NetworkRequest{Host: "www.external.com", Port: 443}, grants))
	// Should NOT match (port 443 on api.internal not in any rule)
	assert.False(t, p.CheckNetwork(entities.NetworkRequest{Host: "api.internal", Port: 443}, grants))
	// Should NOT match (port 80 on external.com not in any rule)
	assert.False(t, p.CheckNetwork(entities.NetworkRequest{Host: "www.external.com", Port: 80}, grants))
}

func TestPolicy_CheckFileSystem(t *testing.T) {
	p := policy.NewPolicy(
		policy.WithDenialHandler(&policy.NopDenialHandler{}),
		policy.WithSymlinkResolution(false), // Disable for deterministic tests
	)

	grants := &entities.GrantSet{
		FS: &entities.FileSystemCapability{
			Rules: []entities.FileSystemRule{
				{Read: []string{"/data/**", "/etc/hosts"}, Write: []string{"/tmp/*"}},
			},
		},
	}

	tests := []struct {
		name string
		req  entities.FileSystemRequest
		want bool
	}{
		{"Allowed read exact", entities.FileSystemRequest{Path: "/etc/hosts", Operation: "read"}, true},
		{"Allowed read glob", entities.FileSystemRequest{Path: "/data/foo/bar", Operation: "read"}, true},
		{"Allowed write glob", entities.FileSystemRequest{Path: "/tmp/foo", Operation: "write"}, true},
		{"Denied read", entities.FileSystemRequest{Path: "/etc/passwd", Operation: "read"}, false},
		{"Denied write", entities.FileSystemRequest{Path: "/data/foo", Operation: "write"}, false},
		{"Denied write outside glob", entities.FileSystemRequest{Path: "/tmp/foo/bar", Operation: "write"}, false},
		{"Cleaned path match", entities.FileSystemRequest{Path: "/data/../data/foo/bar", Operation: "read"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, p.CheckFileSystem(tt.req, grants))
		})
	}
}

func TestPolicy_CheckFileSystem_RelativePath(t *testing.T) {
	// Test that relative paths are denied without cwd
	p := policy.NewPolicy(
		policy.WithDenialHandler(&policy.NopDenialHandler{}),
		policy.WithSymlinkResolution(false),
	)
	grants := &entities.GrantSet{
		FS: &entities.FileSystemCapability{
			Rules: []entities.FileSystemRule{
				{Read: []string{"/app/**"}},
			},
		},
	}

	// Relative path without cwd should be denied
	assert.False(t, p.CheckFileSystem(entities.FileSystemRequest{Path: "data/file.txt", Operation: "read"}, grants))

	// With cwd set, relative path should work
	pWithCwd := policy.NewPolicy(
		policy.WithDenialHandler(&policy.NopDenialHandler{}),
		policy.WithWorkingDirectory("/app"),
		policy.WithSymlinkResolution(false),
	)
	assert.True(t, pWithCwd.CheckFileSystem(entities.FileSystemRequest{Path: "data/file.txt", Operation: "read"}, grants))
}

func TestPolicy_CheckEnvironment(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Env: &entities.EnvironmentCapability{
			Variables: []string{"APP_*", "DEBUG"},
		},
	}

	assert.True(t, p.CheckEnvironment(entities.EnvironmentRequest{Variable: "DEBUG"}, grants))
	assert.True(t, p.CheckEnvironment(entities.EnvironmentRequest{Variable: "APP_ENV"}, grants))
	assert.False(t, p.CheckEnvironment(entities.EnvironmentRequest{Variable: "PATH"}, grants))
}

func TestPolicy_CheckExec(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Exec: &entities.ExecCapability{
			Commands: []string{"/usr/bin/*"},
		},
	}

	assert.True(t, p.CheckExec(entities.ExecCapabilityRequest{Command: "/usr/bin/ls"}, grants))
	assert.False(t, p.CheckExec(entities.ExecCapabilityRequest{Command: "/bin/sh"}, grants))
}

func TestPolicy_CheckKeyValue(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		KV: &entities.KeyValueCapability{
			Rules: []entities.KeyValueRule{
				{Keys: []string{"config/*"}, Operation: "read"},
			},
		},
	}

	assert.True(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "config/db", Operation: "read"}, grants))
	assert.False(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "config/db", Operation: "write"}, grants))
	assert.False(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "secret", Operation: "read"}, grants))
}

func TestPolicy_CheckKeyValue_MultipleRules(t *testing.T) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		KV: &entities.KeyValueCapability{
			Rules: []entities.KeyValueRule{
				{Keys: []string{"config/*"}, Operation: "read"},
				{Keys: []string{"cache/*"}, Operation: "read-write"},
			},
		},
	}

	// config/* is read-only
	assert.True(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "config/db", Operation: "read"}, grants))
	assert.False(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "config/db", Operation: "write"}, grants))

	// cache/* is read-write
	assert.True(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "cache/session", Operation: "read"}, grants))
	assert.True(t, p.CheckKeyValue(entities.KeyValueRequest{Key: "cache/session", Operation: "write"}, grants))
}
