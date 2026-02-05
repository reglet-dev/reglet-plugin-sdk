package entities

import (
	"testing"
)

func TestGrantSet_Merge_Deduplication(t *testing.T) {
	tests := []struct {
		name     string
		initial  *GrantSet
		toMerge  *GrantSet
		expected *GrantSet
	}{
		{
			name: "Network rules deduplicated",
			initial: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"80"}},
					},
				},
			},
			toMerge: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"80"}}, // duplicate
						{Hosts: []string{"google.com"}, Ports: []string{"443"}},
					},
				},
			},
			expected: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"80"}},
						{Hosts: []string{"google.com"}, Ports: []string{"443"}},
					},
				},
			},
		},
		{
			name: "FS rules deduplicated",
			initial: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/tmp"}},
					},
				},
			},
			toMerge: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/tmp"}}, // duplicate
						{Read: []string{"/etc"}},
					},
				},
			},
			expected: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/tmp"}},
						{Read: []string{"/etc"}},
					},
				},
			},
		},
		{
			name: "Env variables deduplicated",
			initial: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"FOO"},
				},
			},
			toMerge: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"FOO", "BAR"}, // FOO is duplicate
				},
			},
			expected: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"FOO", "BAR"},
				},
			},
		},
		{
			name: "Exec commands deduplicated",
			initial: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"/bin/sh"},
				},
			},
			toMerge: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"/bin/sh", "uname"}, // /bin/sh is duplicate
				},
			},
			expected: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"/bin/sh", "uname"},
				},
			},
		},
		{
			name: "KV rules deduplicated",
			initial: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "read", Keys: []string{"key1"}},
					},
				},
			},
			toMerge: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "read", Keys: []string{"key1"}}, // duplicate
						{Operation: "write", Keys: []string{"key2"}},
					},
				},
			},
			expected: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "read", Keys: []string{"key1"}},
						{Operation: "write", Keys: []string{"key2"}},
					},
				},
			},
		},
		{
			name: "Multiple merges deduplicated",
			initial: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"*"}, Ports: []string{"443", "80"}},
					},
				},
			},
			toMerge: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"*"}, Ports: []string{"443", "80"}}, // duplicate
					},
				},
			},
			expected: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"*"}, Ports: []string{"443", "80"}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Merge(tt.toMerge)

			// Compare results
			if !grantsetsEqual(tt.initial, tt.expected) {
				t.Errorf("GrantSet.Merge() deduplication failed\ngot:  %+v\nwant: %+v", tt.initial, tt.expected)
			}
		})
	}
}

// Helper function to compare GrantSets
func grantsetsEqual(a, b *GrantSet) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare Network
	if !networkCapabilitiesEqual(a.Network, b.Network) {
		return false
	}

	// Compare FS
	if !fsCapabilitiesEqual(a.FS, b.FS) {
		return false
	}

	// Compare Env
	if !envCapabilitiesEqual(a.Env, b.Env) {
		return false
	}

	// Compare Exec
	if !execCapabilitiesEqual(a.Exec, b.Exec) {
		return false
	}

	// Compare KV
	if !kvCapabilitiesEqual(a.KV, b.KV) {
		return false
	}

	return true
}

func networkCapabilitiesEqual(a, b *NetworkCapability) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a.Rules) != len(b.Rules) {
		return false
	}
	for i := range a.Rules {
		if !networkRulesEqual(a.Rules[i], b.Rules[i]) {
			return false
		}
	}
	return true
}

func fsCapabilitiesEqual(a, b *FileSystemCapability) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a.Rules) != len(b.Rules) {
		return false
	}
	for i := range a.Rules {
		if !fsRulesEqual(a.Rules[i], b.Rules[i]) {
			return false
		}
	}
	return true
}

func envCapabilitiesEqual(a, b *EnvironmentCapability) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a.Variables) != len(b.Variables) {
		return false
	}
	for i := range a.Variables {
		if a.Variables[i] != b.Variables[i] {
			return false
		}
	}
	return true
}

func execCapabilitiesEqual(a, b *ExecCapability) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a.Commands) != len(b.Commands) {
		return false
	}
	for i := range a.Commands {
		if a.Commands[i] != b.Commands[i] {
			return false
		}
	}
	return true
}

func kvCapabilitiesEqual(a, b *KeyValueCapability) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil || len(a.Rules) != len(b.Rules) {
		return false
	}
	for i := range a.Rules {
		if !kvRulesEqual(a.Rules[i], b.Rules[i]) {
			return false
		}
	}
	return true
}
