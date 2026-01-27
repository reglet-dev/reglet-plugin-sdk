package policy_test

import (
	"testing"

	"github.com/reglet-dev/reglet-sdk/go/domain/entities"
	"github.com/reglet-dev/reglet-sdk/go/domain/policy"
)

func BenchmarkCheckNetwork(b *testing.B) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"example.com", "*.internal"}, Ports: []string{"80", "443"}},
			},
		},
	}
	req := entities.NetworkRequest{Host: "example.com", Port: 80}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CheckNetwork(req, grants)
	}
}

func BenchmarkCheckFileSystem(b *testing.B) {
	p := policy.NewPolicy(
		policy.WithDenialHandler(&policy.NopDenialHandler{}),
		policy.WithSymlinkResolution(false),
	)
	grants := &entities.GrantSet{
		FS: &entities.FileSystemCapability{
			Rules: []entities.FileSystemRule{
				{Read: []string{"/data/**", "/etc/hosts"}},
			},
		},
	}
	req := entities.FileSystemRequest{Path: "/data/foo/bar", Operation: "read"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CheckFileSystem(req, grants)
	}
}

func BenchmarkCheckEnvironment(b *testing.B) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Env: &entities.EnvironmentCapability{
			Variables: []string{"APP_*"},
		},
	}
	req := entities.EnvironmentRequest{Variable: "APP_DEBUG"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CheckEnvironment(req, grants)
	}
}

func BenchmarkCheckExec(b *testing.B) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		Exec: &entities.ExecCapability{
			Commands: []string{"/usr/bin/*", "/opt/tools/**"},
		},
	}
	req := entities.ExecCapabilityRequest{Command: "/usr/bin/ls"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CheckExec(req, grants)
	}
}

func BenchmarkCheckKeyValue(b *testing.B) {
	p := policy.NewPolicy(policy.WithDenialHandler(&policy.NopDenialHandler{}))
	grants := &entities.GrantSet{
		KV: &entities.KeyValueCapability{
			Rules: []entities.KeyValueRule{
				{Keys: []string{"config/*", "cache/**"}, Operation: "read-write"},
			},
		},
	}
	req := entities.KeyValueRequest{Key: "config/database", Operation: "read"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.CheckKeyValue(req, grants)
	}
}
