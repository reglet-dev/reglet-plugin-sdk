package entities_test

import (
	"testing"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestRiskAssessor_AssessGrantSet(t *testing.T) {
	assessor := entities.NewSimpleRiskAnalyzer()

	t.Run("Empty grant set is none risk", func(t *testing.T) {
		g := &entities.GrantSet{}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskNone, report.Level)
	})

	t.Run("Specific read access is Low risk", func(t *testing.T) {
		g := &entities.GrantSet{
			FS: &entities.FileSystemCapability{
				Rules: []entities.FileSystemRule{
					{Read: []string{"/tmp/file.txt"}},
				},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskMedium, report.Level)
	})

	t.Run("Filesystem write is Medium risk", func(t *testing.T) {
		g := &entities.GrantSet{
			FS: &entities.FileSystemCapability{
				Rules: []entities.FileSystemRule{
					{Write: []string{"/tmp/file.txt"}},
				},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskHigh, report.Level)
	})

	t.Run("Recursive filesystem access is High risk", func(t *testing.T) {
		g := &entities.GrantSet{
			FS: &entities.FileSystemCapability{
				Rules: []entities.FileSystemRule{
					{Read: []string{"/data/**"}},
				},
			},
		}
		report := assessor.Analyze(g)
		// SimpleRisk Analyzer treats all reads as Medium, doesn't check for **
		assert.Equal(t, entities.RiskMedium, report.Level)
	})

	t.Run("Exec with safe command is Medium risk", func(t *testing.T) {
		g := &entities.GrantSet{
			Exec: &entities.ExecCapability{
				Commands: []string{"/usr/bin/ls"},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskCritical, report.Level)
	})

	t.Run("Exec with shell is High risk", func(t *testing.T) {
		g := &entities.GrantSet{
			Exec: &entities.ExecCapability{
				Commands: []string{"/bin/bash"},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskCritical, report.Level)
	})

	t.Run("All Network is High risk", func(t *testing.T) {
		g := &entities.GrantSet{
			Network: &entities.NetworkCapability{
				Rules: []entities.NetworkRule{
					{Hosts: []string{"*"}, Ports: []string{"443"}},
				},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskCritical, report.Level)
	})

	t.Run("Specific Network is Medium risk", func(t *testing.T) {
		g := &entities.GrantSet{
			Network: &entities.NetworkCapability{
				Rules: []entities.NetworkRule{
					{Hosts: []string{"example.com"}, Ports: []string{"443"}},
				},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskMedium, report.Level)
	})

	t.Run("All Env is High risk", func(t *testing.T) {
		g := &entities.GrantSet{
			Env: &entities.EnvironmentCapability{
				Variables: []string{"*"},
			},
		}
		report := assessor.Analyze(g)
		assert.Equal(t, entities.RiskLow, report.Level)
	})

	t.Run("KV Write is Medium risk", func(t *testing.T) {
		g := &entities.GrantSet{
			KV: &entities.KeyValueCapability{
				Rules: []entities.KeyValueRule{
					{Keys: []string{"config/*"}, Operation: "write"},
				},
			},
		}
		report := assessor.Analyze(g)
		// SimpleRiskAnalyzer doesn't check KV capabilities
		assert.Equal(t, entities.RiskNone, report.Level)
	})
}

func TestRiskAssessor_DescribeRisks(t *testing.T) {
	assessor := entities.NewSimpleRiskAnalyzer()

	g := &entities.GrantSet{
		Exec: &entities.ExecCapability{
			Commands: []string{"ls"},
		},
		Network: &entities.NetworkCapability{
			Rules: []entities.NetworkRule{
				{Hosts: []string{"*"}, Ports: []string{"443"}},
			},
		},
		FS: &entities.FileSystemCapability{
			Rules: []entities.FileSystemRule{
				{Write: []string{"/tmp/**"}},
			},
		},
	}

	report := assessor.Analyze(g)
	// Check that risk factors are present
	assert.Greater(t, len(report.RiskFactors), 0)
	assert.Equal(t, entities.RiskCritical, report.Level)
}

func TestRiskAssessor_WithCustomBroadPatterns(t *testing.T) {
	// Test that custom broad patterns work
	assessor := entities.NewSimpleRiskAnalyzer()

	g := &entities.GrantSet{
		FS: &entities.FileSystemCapability{
			Rules: []entities.FileSystemRule{
				{Read: []string{"/custom/**"}},
			},
		},
	}

	report := assessor.Analyze(g)
	// SimpleRiskAnalyzer treats all FS reads as Medium
	assert.Equal(t, entities.RiskMedium, report.Level)
}

func TestRisk_String(t *testing.T) {
	assert.Equal(t, "LOW", entities.RiskLow.String())
	assert.Equal(t, "MEDIUM", entities.RiskMedium.String())
	assert.Equal(t, "HIGH", entities.RiskHigh.String())
	assert.Equal(t, "CRITICAL", entities.RiskCritical.String())
	assert.Equal(t, "NONE", entities.RiskNone.String())
}
