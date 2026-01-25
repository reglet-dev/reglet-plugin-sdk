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
