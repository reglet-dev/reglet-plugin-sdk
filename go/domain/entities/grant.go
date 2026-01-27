package entities

// GrantSet is a structured collection of rules representing all capabilities granted to a plugin.
type GrantSet struct {
	Network *NetworkCapability     `json:"network,omitempty" yaml:"network,omitempty"`
	FS      *FileSystemCapability  `json:"fs,omitempty" yaml:"fs,omitempty"`
	Env     *EnvironmentCapability `json:"env,omitempty" yaml:"env,omitempty"`
	Exec    *ExecCapability        `json:"exec,omitempty" yaml:"exec,omitempty"`
	KV      *KeyValueCapability    `json:"kv,omitempty" yaml:"kv,omitempty"`
}

// IsEmpty returns true if no capabilities are present.
func (g *GrantSet) IsEmpty() bool {
	if g == nil {
		return true
	}
	if g.Network != nil && len(g.Network.Rules) > 0 {
		return false
	}
	if g.FS != nil && len(g.FS.Rules) > 0 {
		return false
	}
	if g.Env != nil && len(g.Env.Variables) > 0 {
		return false
	}
	if g.Exec != nil && len(g.Exec.Commands) > 0 {
		return false
	}
	if g.KV != nil && len(g.KV.Rules) > 0 {
		return false
	}
	return true
}

// Merge unions two grant sets.
func (g *GrantSet) Merge(other *GrantSet) {
	if other == nil {
		return
	}
	g.mergeNetwork(other.Network)
	g.mergeFS(other.FS)
	g.mergeEnv(other.Env)
	g.mergeExec(other.Exec)
	g.mergeKV(other.KV)
}

func (g *GrantSet) mergeNetwork(other *NetworkCapability) {
	if other == nil || len(other.Rules) == 0 {
		return
	}
	if g.Network == nil {
		g.Network = &NetworkCapability{}
	}
	g.Network.Rules = append(g.Network.Rules, other.Rules...)
}

func (g *GrantSet) mergeFS(other *FileSystemCapability) {
	if other == nil || len(other.Rules) == 0 {
		return
	}
	if g.FS == nil {
		g.FS = &FileSystemCapability{}
	}
	g.FS.Rules = append(g.FS.Rules, other.Rules...)
}

func (g *GrantSet) mergeEnv(other *EnvironmentCapability) {
	if other == nil || len(other.Variables) == 0 {
		return
	}
	if g.Env == nil {
		g.Env = &EnvironmentCapability{}
	}
	g.Env.Variables = append(g.Env.Variables, other.Variables...)
}

func (g *GrantSet) mergeExec(other *ExecCapability) {
	if other == nil || len(other.Commands) == 0 {
		return
	}
	if g.Exec == nil {
		g.Exec = &ExecCapability{}
	}
	g.Exec.Commands = append(g.Exec.Commands, other.Commands...)
}

func (g *GrantSet) mergeKV(other *KeyValueCapability) {
	if other == nil || len(other.Rules) == 0 {
		return
	}
	if g.KV == nil {
		g.KV = &KeyValueCapability{}
	}
	g.KV.Rules = append(g.KV.Rules, other.Rules...)
}

// Clone returns a deep copy of the GrantSet.
func (g *GrantSet) Clone() *GrantSet {
	if g == nil {
		return nil
	}
	clone := &GrantSet{}
	if g.Network != nil {
		clone.Network = &NetworkCapability{
			Rules: make([]NetworkRule, len(g.Network.Rules)),
		}
		for i, rule := range g.Network.Rules {
			clone.Network.Rules[i] = NetworkRule{
				Hosts: append([]string(nil), rule.Hosts...),
				Ports: append([]string(nil), rule.Ports...),
			}
		}
	}
	if g.FS != nil {
		clone.FS = &FileSystemCapability{
			Rules: make([]FileSystemRule, len(g.FS.Rules)),
		}
		for i, rule := range g.FS.Rules {
			clone.FS.Rules[i] = FileSystemRule{
				Read:  append([]string(nil), rule.Read...),
				Write: append([]string(nil), rule.Write...),
			}
		}
	}
	if g.Env != nil {
		clone.Env = &EnvironmentCapability{
			Variables: append([]string(nil), g.Env.Variables...),
		}
	}
	if g.Exec != nil {
		clone.Exec = &ExecCapability{
			Commands: append([]string(nil), g.Exec.Commands...),
		}
	}
	if g.KV != nil {
		clone.KV = &KeyValueCapability{
			Rules: make([]KeyValueRule, len(g.KV.Rules)),
		}
		for i, rule := range g.KV.Rules {
			clone.KV.Rules[i] = KeyValueRule{
				Operation: rule.Operation,
				Keys:      append([]string(nil), rule.Keys...),
			}
		}
	}
	return clone
}

// Difference returns capabilities in g that are not covered by other.
// Useful for determining what capabilities still need to be granted.
func (g *GrantSet) Difference(other *GrantSet) *GrantSet {
	if g == nil {
		return nil
	}
	if other == nil {
		return g.Clone()
	}

	result := &GrantSet{}

	// Network difference
	if g.Network != nil {
		for _, rule := range g.Network.Rules {
			if !other.containsNetworkRule(rule) {
				if result.Network == nil {
					result.Network = &NetworkCapability{}
				}
				result.Network.Rules = append(result.Network.Rules, rule)
			}
		}
	}

	// FS difference
	if g.FS != nil {
		for _, rule := range g.FS.Rules {
			if !other.containsFSRule(rule) {
				if result.FS == nil {
					result.FS = &FileSystemCapability{}
				}
				result.FS.Rules = append(result.FS.Rules, rule)
			}
		}
	}

	// Env difference
	if g.Env != nil {
		for _, v := range g.Env.Variables {
			if !other.containsEnvVar(v) {
				if result.Env == nil {
					result.Env = &EnvironmentCapability{}
				}
				result.Env.Variables = append(result.Env.Variables, v)
			}
		}
	}

	// Exec difference
	if g.Exec != nil {
		for _, cmd := range g.Exec.Commands {
			if !other.containsExecCmd(cmd) {
				if result.Exec == nil {
					result.Exec = &ExecCapability{}
				}
				result.Exec.Commands = append(result.Exec.Commands, cmd)
			}
		}
	}

	// KV difference
	if g.KV != nil {
		for _, rule := range g.KV.Rules {
			if !other.containsKVRule(rule) {
				if result.KV == nil {
					result.KV = &KeyValueCapability{}
				}
				result.KV.Rules = append(result.KV.Rules, rule)
			}
		}
	}

	return result
}

// Contains returns true if g covers all capabilities in other.
func (g *GrantSet) Contains(other *GrantSet) bool {
	if other == nil || other.IsEmpty() {
		return true
	}
	if g == nil {
		return false
	}
	diff := other.Difference(g)
	return diff.IsEmpty()
}

// Helper methods for checking containment
func (g *GrantSet) containsNetworkRule(rule NetworkRule) bool {
	if g.Network == nil {
		return false
	}
	for _, r := range g.Network.Rules {
		if networkRulesEqual(r, rule) {
			return true
		}
	}
	return false
}

func (g *GrantSet) containsFSRule(rule FileSystemRule) bool {
	if g.FS == nil {
		return false
	}
	for _, r := range g.FS.Rules {
		if fsRulesEqual(r, rule) {
			return true
		}
	}
	return false
}

func (g *GrantSet) containsEnvVar(v string) bool {
	if g.Env == nil {
		return false
	}
	for _, ev := range g.Env.Variables {
		if ev == v {
			return true
		}
	}
	return false
}

func (g *GrantSet) containsExecCmd(cmd string) bool {
	if g.Exec == nil {
		return false
	}
	for _, c := range g.Exec.Commands {
		if c == cmd {
			return true
		}
	}
	return false
}

func (g *GrantSet) containsKVRule(rule KeyValueRule) bool {
	if g.KV == nil {
		return false
	}
	for _, r := range g.KV.Rules {
		if kvRulesEqual(r, rule) {
			return true
		}
	}
	return false
}

// Equality helpers
func networkRulesEqual(a, b NetworkRule) bool {
	if len(a.Hosts) != len(b.Hosts) || len(a.Ports) != len(b.Ports) {
		return false
	}
	for i := range a.Hosts {
		if a.Hosts[i] != b.Hosts[i] {
			return false
		}
	}
	for i := range a.Ports {
		if a.Ports[i] != b.Ports[i] {
			return false
		}
	}
	return true
}

func fsRulesEqual(a, b FileSystemRule) bool {
	if len(a.Read) != len(b.Read) || len(a.Write) != len(b.Write) {
		return false
	}
	for i := range a.Read {
		if a.Read[i] != b.Read[i] {
			return false
		}
	}
	for i := range a.Write {
		if a.Write[i] != b.Write[i] {
			return false
		}
	}
	return true
}

func kvRulesEqual(a, b KeyValueRule) bool {
	if a.Operation != b.Operation || len(a.Keys) != len(b.Keys) {
		return false
	}
	for i := range a.Keys {
		if a.Keys[i] != b.Keys[i] {
			return false
		}
	}
	return true
}
