package ports

// TemplateEngine renders templates with configuration values.
type TemplateEngine interface {
	// Render processes the raw manifest bytes with the provided config.
	// Returns resolved bytes with all template placeholders replaced.
	Render(raw []byte, config map[string]interface{}) ([]byte, error)
}

// ConfigProvider supplies configuration values for template resolution.
type ConfigProvider interface {
	// GetConfig returns the configuration map for a plugin.
	GetConfig(pluginName string) (map[string]interface{}, error)
}
