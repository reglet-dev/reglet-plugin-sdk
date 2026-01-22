package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExamplePlugin_Describe(t *testing.T) {
	p := &ExamplePlugin{}
	ctx := context.Background()

	metadata, err := p.Describe(ctx)
	require.NoError(t, err)

	assert.Equal(t, "Example Compliance Plugin", metadata.Name)
	assert.Equal(t, "1.0.0", metadata.Version)
	assert.Len(t, metadata.Capabilities, 1)
	assert.Equal(t, "custom", metadata.Capabilities[0].Category)
	assert.Equal(t, "tls_check", metadata.Capabilities[0].Resource)
}

func TestExamplePlugin_Schema(t *testing.T) {
	p := &ExamplePlugin{}
	ctx := context.Background()

	schema, err := p.Schema(ctx)
	require.NoError(t, err)

	var schemaMap map[string]any
	err = json.Unmarshal(schema, &schemaMap)
	require.NoError(t, err)

	assert.Equal(t, "http://json-schema.org/draft-07/schema#", schemaMap["$schema"])
	assert.Contains(t, schemaMap["properties"], "target_host")
}

func TestLoadConfig(t *testing.T) {
	t.Run("Default values", func(t *testing.T) {
		cfg, err := LoadConfig(nil)
		require.NoError(t, err)
		assert.Equal(t, "example.com", cfg.TargetHost)
		assert.Equal(t, 443, cfg.TargetPort)
		assert.Equal(t, 30, cfg.MinDays)
	})

	t.Run("Override values", func(t *testing.T) {
		input := map[string]any{
			"target_host": "google.com",
			"target_port": 8443,
			"min_days":    60,
		}
		cfg, err := LoadConfig(input)
		require.NoError(t, err)
		assert.Equal(t, "google.com", cfg.TargetHost)
		assert.Equal(t, 8443, cfg.TargetPort)
		assert.Equal(t, 60, cfg.MinDays)
	})
}
