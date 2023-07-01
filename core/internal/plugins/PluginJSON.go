package plugins

type PluginType string

const (
	CodeManager PluginType = "code_manager"
	Builder     PluginType = "builder"
)

type Plugin struct {
	Name          string                 `json:"name"`
	PluginExePath string                 `json:"executable"`
	Settings      map[string]interface{} `json:"settings_schema"`
	PluginTypes   []PluginType           `json:"plugin_types"`
}
