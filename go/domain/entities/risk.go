package entities

import (
	"fmt"
)

// RiskLevel represents the security risk level of a capability.
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskNone:
		return "NONE"
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	case RiskCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// RiskAnalyzer assesses the risk of requested capabilities.
type RiskAnalyzer interface {
	// Analyze now accepts a GrantSet pointer
	Analyze(grants *GrantSet) RiskReport
}

// RiskReport contains the risk assessment results.
type RiskReport struct {
	Level       RiskLevel
	RiskFactors []RiskFactor
}

// RiskFactor describes a specific risky capability.
type RiskFactor struct {
	Level       RiskLevel
	Description string
	// Rule is a human-readable representation of the specific rule causing this risk
	Rule string
}

// SimpleRiskAnalyzer implements basic heuristic risk analysis.
type SimpleRiskAnalyzer struct{}

func NewSimpleRiskAnalyzer() RiskAnalyzer {
	return &SimpleRiskAnalyzer{}
}

func (a *SimpleRiskAnalyzer) Analyze(grants *GrantSet) RiskReport {
	report := RiskReport{
		Level: RiskNone,
	}

	if grants == nil {
		return report
	}

	// Helper to add a factor
	addFactor := func(level RiskLevel, desc, rule string) {
		if level > RiskNone {
			report.RiskFactors = append(report.RiskFactors, RiskFactor{
				Level:       level,
				Description: desc,
				Rule:        rule,
			})
			if level > report.Level {
				report.Level = level
			}
		}
	}

	// 1. Analyze Network
	if grants.Network != nil {
		for _, rule := range grants.Network.Rules {
			ruleStr := fmt.Sprintf("Network: %s:%s", rule.Hosts, rule.Ports)

			// Critical: Wildcard host
			isWildcardHost := false
			for _, h := range rule.Hosts {
				if h == "*" || h == "0.0.0.0" {
					isWildcardHost = true
					break
				}
			}

			if isWildcardHost {
				addFactor(RiskCritical, "Unrestricted network access", ruleStr)
			} else {
				addFactor(RiskMedium, "Outbound network access", ruleStr)
			}
		}
	}

	// 2. Analyze FS
	if grants.FS != nil {
		for _, rule := range grants.FS.Rules {
			if len(rule.Write) > 0 {
				ruleStr := fmt.Sprintf("FS Write: %v", rule.Write)
				addFactor(RiskHigh, "Filesystem write access", ruleStr)
			}
			if len(rule.Read) > 0 {
				ruleStr := fmt.Sprintf("FS Read: %v", rule.Read)
				addFactor(RiskMedium, "Filesystem read access", ruleStr)
			}
		}
	}

	// 3. Analyze Exec
	if grants.Exec != nil && len(grants.Exec.Commands) > 0 {
		ruleStr := fmt.Sprintf("Exec: %v", grants.Exec.Commands)
		addFactor(RiskCritical, "Arbitrary command execution", ruleStr)
	}

	// 4. Analyze Env
	if grants.Env != nil && len(grants.Env.Variables) > 0 {
		ruleStr := fmt.Sprintf("Env: %v", grants.Env.Variables)
		addFactor(RiskLow, "Environment variable access", ruleStr)
	}

	return report
}
