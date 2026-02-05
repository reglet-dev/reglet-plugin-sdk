package prompter_test

import (
	"bytes"
	"testing"

	"github.com/reglet-dev/reglet-sdk/domain/entities"
	"github.com/reglet-dev/reglet-sdk/infrastructure/prompter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliPrompter_PromptForCapability(t *testing.T) {
	req := entities.CapabilityRequest{
		Description: "Connect to google.com",
		RiskLevel:   entities.RiskLow,
	}

	t.Run("Grant", func(t *testing.T) {
		in := bytes.NewBufferString("y\n")
		out := &bytes.Buffer{}
		p := prompter.NewCliPrompter(in, out)

		granted, always, err := p.PromptForCapability(req)
		require.NoError(t, err)
		assert.True(t, granted)
		assert.False(t, always)
		assert.Contains(t, out.String(), "Plugin Request: Connect to google.com")
	})

	t.Run("Grant Always", func(t *testing.T) {
		in := bytes.NewBufferString("always\n")
		out := &bytes.Buffer{}
		p := prompter.NewCliPrompter(in, out)

		granted, always, err := p.PromptForCapability(req)
		require.NoError(t, err)
		assert.True(t, granted)
		assert.True(t, always)
	})

	t.Run("Deny", func(t *testing.T) {
		in := bytes.NewBufferString("n\n")
		out := &bytes.Buffer{}
		p := prompter.NewCliPrompter(in, out)

		granted, always, err := p.PromptForCapability(req)
		require.NoError(t, err)
		assert.False(t, granted)
		assert.False(t, always)
	})
}

func TestCliPrompter_PromptForCapabilities(t *testing.T) {
	reqs := []entities.CapabilityRequest{
		{
			Description: "Network access",
			RiskLevel:   entities.RiskLow,
			Rule: &entities.NetworkCapability{
				Rules: []entities.NetworkRule{
					{Hosts: []string{"example.com"}, Ports: []string{"443"}},
				},
			},
		},
		{
			Description: "File write",
			RiskLevel:   entities.RiskMedium,
			Rule: &entities.FileSystemCapability{
				Rules: []entities.FileSystemRule{
					{Write: []string{"/tmp/out"}},
				},
			},
		},
	}

	t.Run("Grant All", func(t *testing.T) {
		in := bytes.NewBufferString("y\n")
		out := &bytes.Buffer{}
		p := prompter.NewCliPrompter(in, out)

		gs, err := p.PromptForCapabilities(reqs)
		require.NoError(t, err)
		assert.NotNil(t, gs)
		assert.False(t, gs.IsEmpty())
		assert.Len(t, gs.Network.Rules, 1)
		assert.Equal(t, []string{"example.com"}, gs.Network.Rules[0].Hosts)
		assert.Len(t, gs.FS.Rules, 1)
		assert.Equal(t, []string{"/tmp/out"}, gs.FS.Rules[0].Write)
		assert.Contains(t, out.String(), "Grant all? [y/n]:")
	})

	t.Run("Deny All", func(t *testing.T) {
		in := bytes.NewBufferString("n\n")
		out := &bytes.Buffer{}
		p := prompter.NewCliPrompter(in, out)

		gs, err := p.PromptForCapabilities(reqs)
		require.NoError(t, err)
		assert.NotNil(t, gs)
		assert.True(t, gs.IsEmpty())
	})
}

func TestCliPrompter_FormatNonInteractiveError(t *testing.T) {
	p := prompter.NewCliPrompter(nil, nil)
	err := p.FormatNonInteractiveError(nil)
	assert.ErrorContains(t, err, "plugin requires capabilities in non-interactive mode")
}
