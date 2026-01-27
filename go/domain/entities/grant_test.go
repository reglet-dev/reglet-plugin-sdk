package entities

import (
	"reflect"
	"testing"
)

func TestGrantSet_Difference(t *testing.T) {
	tests := []struct {
		name     string
		g        *GrantSet
		other    *GrantSet
		expected *GrantSet
	}{
		{
			name:     "Both nil",
			g:        nil,
			other:    nil,
			expected: nil,
		},
		{
			name:     "G nil",
			g:        nil,
			other:    &GrantSet{},
			expected: nil,
		},
		{
			name:     "Other nil",
			g:        &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
			other:    nil,
			expected: &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
		},
		{
			name:     "Other empty",
			g:        &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
			other:    &GrantSet{},
			expected: &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
		},
		{
			name: "Network difference",
			g: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"80"}},
						{Hosts: []string{"google.com"}, Ports: []string{"443"}},
					},
				},
			},
			other: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"example.com"}, Ports: []string{"80"}},
					},
				},
			},
			expected: &GrantSet{
				Network: &NetworkCapability{
					Rules: []NetworkRule{
						{Hosts: []string{"google.com"}, Ports: []string{"443"}},
					},
				},
			},
		},
		{
			name: "FS difference",
			g: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/tmp"}, Write: []string{"/var"}},
						{Read: []string{"/etc"}, Write: []string{}},
					},
				},
			},
			other: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/tmp"}, Write: []string{"/var"}},
					},
				},
			},
			expected: &GrantSet{
				FS: &FileSystemCapability{
					Rules: []FileSystemRule{
						{Read: []string{"/etc"}, Write: []string{}},
					},
				},
			},
		},
		{
			name: "Env difference",
			g: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"FOO", "BAR"},
				},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"FOO"},
				},
			},
			expected: &GrantSet{
				Env: &EnvironmentCapability{
					Variables: []string{"BAR"},
				},
			},
		},
		{
			name: "Exec difference",
			g: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"ls", "grep"},
				},
			},
			other: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"ls"},
				},
			},
			expected: &GrantSet{
				Exec: &ExecCapability{
					Commands: []string{"grep"},
				},
			},
		},
		{
			name: "KV difference",
			g: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "read", Keys: []string{"key1"}},
						{Operation: "write", Keys: []string{"key2"}},
					},
				},
			},
			other: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "read", Keys: []string{"key1"}},
					},
				},
			},
			expected: &GrantSet{
				KV: &KeyValueCapability{
					Rules: []KeyValueRule{
						{Operation: "write", Keys: []string{"key2"}},
					},
				},
			},
		},
		{
			name: "No difference (subset)",
			g: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO", "BAR"}},
			},
			expected: &GrantSet{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.g.Difference(tt.other)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("GrantSet.Difference() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGrantSet_Contains(t *testing.T) {
	tests := []struct {
		name  string
		g     *GrantSet
		other *GrantSet
		want  bool
	}{
		{
			name:  "Nil contains nil",
			g:     nil,
			other: nil,
			want:  true,
		},
		{
			name:  "Nil contains empty",
			g:     nil,
			other: &GrantSet{},
			want:  true,
		},
		{
			name:  "Nil contains something",
			g:     nil,
			other: &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
			want:  false,
		},
		{
			name:  "Something contains nil",
			g:     &GrantSet{Env: &EnvironmentCapability{Variables: []string{"FOO"}}},
			other: nil,
			want:  true,
		},
		{
			name: "Equal sets",
			g: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			want: true,
		},
		{
			name: "Superset contains subset",
			g: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO", "BAR"}},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			want: true,
		},
		{
			name: "Subset does not contain superset",
			g: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO", "BAR"}},
			},
			want: false,
		},
		{
			name: "Disjoint sets",
			g: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"FOO"}},
			},
			other: &GrantSet{
				Env: &EnvironmentCapability{Variables: []string{"BAR"}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.Contains(tt.other); got != tt.want {
				t.Errorf("GrantSet.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
