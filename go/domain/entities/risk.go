package entities

import "strings"

// RiskLevel represents the security risk level of a capability or grant set.
type RiskLevel int

const (
	RiskLevelLow    RiskLevel = iota // Specific, narrow permissions
	RiskLevelMedium                  // Network access, sensitive reads
	RiskLevelHigh                    // Broad permissions, arbitrary execution
)

// String returns the human-readable name of the risk level.
func (r RiskLevel) String() string {
	switch r {
	case RiskLevelLow:
		return "Low"
	case RiskLevelMedium:
		return "Medium"
	case RiskLevelHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// Dangerous patterns (security domain knowledge)
var (
	// Broad filesystem patterns that grant excessive access
	BroadFilesystemPatterns = []string{
		"**", "/**", "read:**", "write:**", "read:/", "write:/",
		"read:/etc/**", "write:/etc/**",
		"read:/root/**", "write:/root/**",
		"read:/home/**", "write:/home/**",
	}

	// Shell interpreters that allow arbitrary command execution
	DangerousShells = []string{
		"bash", "sh", "zsh", "fish",
		"/bin/bash", "/bin/sh", "/bin/zsh",
	}

	// Script interpreters (matches base + versioned variants)
	DangerousInterpreters = []string{
		"python", "perl", "ruby", "node", "nodejs",
		"php", "lua", "awk", "gawk", "tclsh",
	}

	// Broad environment variable patterns
	BroadEnvPatterns = []string{"*", "AWS_*", "AZURE_*", "GCP_*"}
)

// riskAssessorConfig holds configuration for the RiskAssessor.
type riskAssessorConfig struct {
	customBroadPatterns map[string][]string
}

func defaultRiskAssessorConfig() riskAssessorConfig {
	return riskAssessorConfig{
		customBroadPatterns: make(map[string][]string),
	}
}

// RiskAssessorOption configures a RiskAssessor instance.
type RiskAssessorOption func(*riskAssessorConfig)

// WithCustomBroadPatterns adds additional patterns considered "broad" for a kind.
func WithCustomBroadPatterns(kind string, patterns []string) RiskAssessorOption {
	return func(c *riskAssessorConfig) {
		c.customBroadPatterns[kind] = append(c.customBroadPatterns[kind], patterns...)
	}
}

// RiskAssessor evaluates the security risk of capabilities.
type RiskAssessor struct {
	config riskAssessorConfig
}

// NewRiskAssessor creates a new RiskAssessor with the given options.
func NewRiskAssessor(opts ...RiskAssessorOption) *RiskAssessor {
	cfg := defaultRiskAssessorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &RiskAssessor{config: cfg}
}

// AssessGrantSet evaluates the overall risk level of a GrantSet.
func (r *RiskAssessor) AssessGrantSet(g *GrantSet) RiskLevel {
	if g == nil {
		return RiskLevelLow
	}

	// Check for high-risk patterns first (early exit)
	if level := r.assessExec(g.Exec); level == RiskLevelHigh {
		return RiskLevelHigh
	}
	if level := r.assessNetwork(g.Network); level == RiskLevelHigh {
		return RiskLevelHigh
	}
	if level := r.assessFS(g.FS); level == RiskLevelHigh {
		return RiskLevelHigh
	}
	if level := r.assessEnv(g.Env); level == RiskLevelHigh {
		return RiskLevelHigh
	}

	// Determine highest medium risk
	highest := RiskLevelLow
	for _, level := range []RiskLevel{
		r.assessExec(g.Exec),
		r.assessNetwork(g.Network),
		r.assessFS(g.FS),
		r.assessKV(g.KV),
	} {
		if level > highest {
			highest = level
		}
	}
	return highest
}

func (r *RiskAssessor) assessExec(exec *ExecCapability) RiskLevel {
	if exec == nil || len(exec.Commands) == 0 {
		return RiskLevelLow
	}
	for _, cmd := range exec.Commands {
		if cmd == "*" || cmd == "**" {
			return RiskLevelHigh
		}
		if matchesAny(cmd, DangerousShells) || matchesInterpreter(cmd) {
			return RiskLevelHigh
		}
	}
	return RiskLevelMedium
}

func (r *RiskAssessor) assessNetwork(network *NetworkCapability) RiskLevel {
	if network == nil || len(network.Rules) == 0 {
		return RiskLevelLow
	}
	for _, rule := range network.Rules {
		for _, h := range rule.Hosts {
			if h == "*" {
				return RiskLevelHigh
			}
		}
	}
	return RiskLevelMedium
}

func (r *RiskAssessor) assessFS(fs *FileSystemCapability) RiskLevel {
	if fs == nil || len(fs.Rules) == 0 {
		return RiskLevelLow
	}
	allPatterns := r.getBroadFilesystemPatterns()
	hasWrite := false
	hasSensitiveRead := false

	for _, rule := range fs.Rules {
		for _, p := range rule.Read {
			if strings.Contains(p, "**") || matchesAny(p, allPatterns) {
				return RiskLevelHigh
			}
			if strings.HasPrefix(p, "/etc/") {
				hasSensitiveRead = true
			}
		}
		for _, p := range rule.Write {
			if strings.Contains(p, "**") || matchesAny(p, allPatterns) {
				return RiskLevelHigh
			}
			hasWrite = true
		}
	}

	if hasWrite || hasSensitiveRead {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

func (r *RiskAssessor) assessEnv(env *EnvironmentCapability) RiskLevel {
	if env == nil || len(env.Variables) == 0 {
		return RiskLevelLow
	}
	allPatterns := r.getBroadEnvPatterns()
	for _, v := range env.Variables {
		if matchesAny(v, allPatterns) {
			return RiskLevelHigh
		}
	}
	return RiskLevelLow
}

func (r *RiskAssessor) assessKV(kv *KeyValueCapability) RiskLevel {
	if kv == nil || len(kv.Rules) == 0 {
		return RiskLevelLow
	}
	for _, rule := range kv.Rules {
		if rule.Operation == "write" || rule.Operation == "read-write" {
			return RiskLevelMedium
		}
	}
	return RiskLevelLow
}

// getBroadFilesystemPatterns returns base + custom broad filesystem patterns.
func (r *RiskAssessor) getBroadFilesystemPatterns() []string {
	patterns := make([]string, len(BroadFilesystemPatterns))
	copy(patterns, BroadFilesystemPatterns)
	if custom, ok := r.config.customBroadPatterns["fs"]; ok {
		patterns = append(patterns, custom...)
	}
	return patterns
}

// getBroadEnvPatterns returns base + custom broad env patterns.
func (r *RiskAssessor) getBroadEnvPatterns() []string {
	patterns := make([]string, len(BroadEnvPatterns))
	copy(patterns, BroadEnvPatterns)
	if custom, ok := r.config.customBroadPatterns["env"]; ok {
		patterns = append(patterns, custom...)
	}
	return patterns
}

// DescribeRisks returns a list of human-readable risk descriptions.
func (r *RiskAssessor) DescribeRisks(g *GrantSet) []string {
	if g == nil {
		return nil
	}

	var risks []string
	risks = append(risks, r.describeExecRisks(g.Exec)...)
	risks = append(risks, r.describeNetworkRisks(g.Network)...)
	risks = append(risks, r.describeFSRisks(g.FS)...)
	risks = append(risks, r.describeEnvRisks(g.Env)...)
	risks = append(risks, r.describeKVRisks(g.KV)...)
	return risks
}

func (r *RiskAssessor) describeExecRisks(exec *ExecCapability) []string {
	if exec == nil || len(exec.Commands) == 0 {
		return nil
	}
	return []string{"Executes external commands (High Risk)"}
}

func (r *RiskAssessor) describeNetworkRisks(network *NetworkCapability) []string {
	if network == nil {
		return nil
	}
	for _, rule := range network.Rules {
		for _, h := range rule.Hosts {
			if h == "*" {
				return []string{"Accesses any network host (High Risk)"}
			}
		}
	}
	return nil
}

func (r *RiskAssessor) describeFSRisks(fs *FileSystemCapability) []string {
	if fs == nil {
		return nil
	}
	var risks []string

	// Check for recursive access
	for _, rule := range fs.Rules {
		for _, p := range rule.Read {
			if strings.Contains(p, "**") {
				risks = append(risks, "Recursive read access to filesystem (High Risk)")
				break
			}
		}
	}
	for _, rule := range fs.Rules {
		for _, p := range rule.Write {
			if strings.Contains(p, "**") {
				risks = append(risks, "Recursive write access to filesystem (High Risk)")
				break
			}
		}
	}

	// General write check
	for _, rule := range fs.Rules {
		if len(rule.Write) > 0 {
			risks = append(risks, "Write access to filesystem")
			break
		}
	}
	return risks
}

func (r *RiskAssessor) describeEnvRisks(env *EnvironmentCapability) []string {
	if env == nil {
		return nil
	}
	for _, v := range env.Variables {
		if v == "*" {
			return []string{"Accesses all environment variables (High Risk)"}
		}
	}
	return nil
}

func (r *RiskAssessor) describeKVRisks(kv *KeyValueCapability) []string {
	if kv == nil {
		return nil
	}
	for _, rule := range kv.Rules {
		if rule.Operation == "write" || rule.Operation == "read-write" {
			return []string{"Write access to Key-Value store"}
		}
	}
	return nil
}

// matchesAny checks if value matches any pattern in the list.
func matchesAny(value string, patterns []string) bool {
	for _, p := range patterns {
		if value == p {
			return true
		}
	}
	return false
}

// matchesInterpreter checks if cmd matches a dangerous interpreter (base or versioned).
func matchesInterpreter(cmd string) bool {
	for _, interp := range DangerousInterpreters {
		if cmd == interp || strings.HasPrefix(cmd, interp) {
			return true
		}
	}
	return false
}
