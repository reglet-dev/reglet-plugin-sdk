package entities

import (
	"time"
)

// RunMetadata contains execution metadata for SDK operations.
type RunMetadata struct {
	// StartTime is when the operation started.
	StartTime time.Time `json:"start_time"`

	// EndTime is when the operation completed.
	EndTime time.Time `json:"end_time"`

	// SDKVersion is the version of the SDK that executed the operation.
	SDKVersion string `json:"sdk_version,omitempty"`

	// PluginID identifies the plugin that requested the operation (if known).
	PluginID string `json:"plugin_id,omitempty"`

	// Duration is the total execution time.
	Duration time.Duration `json:"duration_ns"`
}

// NewRunMetadata creates a new RunMetadata with the given start and end times.
func NewRunMetadata(start, end time.Time) *RunMetadata {
	return &RunMetadata{
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start),
	}
}

// WithSDKVersion returns a copy of the RunMetadata with the SDK version set.
func (m *RunMetadata) WithSDKVersion(version string) *RunMetadata {
	m.SDKVersion = version
	return m
}

// WithPluginID returns a copy of the RunMetadata with the plugin ID set.
func (m *RunMetadata) WithPluginID(pluginID string) *RunMetadata {
	m.PluginID = pluginID
	return m
}
